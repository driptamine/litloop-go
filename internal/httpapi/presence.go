package httpapi

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const presenceLease = 80 * time.Second

type presenceSession struct {
	expires time.Time
}

func activePresence(sessions map[string]*presenceSession, now time.Time) bool {
	for _, session := range sessions {
		if session.expires.After(now) {
			return true
		}
	}
	return false
}

func (h *Hub) expirePresence(user int64, id string, session *presenceSession, delay time.Duration) {
	time.AfterFunc(delay, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		sessions := h.presence[user]
		if sessions[id] != session {
			return
		}
		if remaining := time.Until(session.expires); remaining > 0 {
			h.expirePresence(user, id, session, remaining)
			return
		}
		delete(sessions, id)
		if len(sessions) == 0 {
			delete(h.presence, user)
		}
	})
}

func presenceKey(user int64) string {
	return "litloop:presence:" + strconv.FormatInt(user, 10)
}

// SetPresence keeps one lease per authenticated notification connection.
func (h *Hub) SetPresence(user int64, id string, connected bool, lease time.Duration) (online, changed bool) {
	if lease <= 0 {
		lease = presenceLease
	}
	now := time.Now()
	h.mu.Lock()
	sessions := h.presence[user]
	wasOnline := activePresence(sessions, now)
	if connected {
		if sessions == nil {
			sessions = make(map[string]*presenceSession)
			h.presence[user] = sessions
		}
		session := sessions[id]
		if session == nil {
			session = &presenceSession{}
			sessions[id] = session
			h.expirePresence(user, id, session, lease)
		}
		session.expires = now.Add(lease)
	} else {
		delete(sessions, id)
		if len(sessions) == 0 {
			delete(h.presence, user)
		}
	}
	online = activePresence(sessions, now)
	h.mu.Unlock()
	if h.bus == nil {
		return online, wasOnline != online
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	key := presenceKey(user)
	cutoff := strconv.FormatInt(now.UnixMilli(), 10)
	var before, after *redis.IntCmd
	_, err := h.bus.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.ZRemRangeByScore(ctx, key, "-inf", cutoff)
		before = pipe.ZCard(ctx, key)
		if connected {
			pipe.ZAdd(ctx, key, redis.Z{Member: id, Score: float64(now.Add(lease).UnixMilli())})
			pipe.Expire(ctx, key, presenceLease)
		} else {
			pipe.ZRem(ctx, key, id)
		}
		after = pipe.ZCard(ctx, key)
		return nil
	})
	if err != nil {
		return online, wasOnline != online
	}
	online = after.Val() > 0
	return online, (before.Val() > 0) != online
}

func (h *Hub) Presence(users []int64) map[int64]bool {
	status := make(map[int64]bool, len(users))
	if h.bus != nil && len(users) != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		cutoff := "(" + strconv.FormatInt(time.Now().UnixMilli(), 10)
		counts := make([]*redis.IntCmd, len(users))
		_, err := h.bus.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			for i, user := range users {
				counts[i] = pipe.ZCount(ctx, presenceKey(user), cutoff, "+inf")
			}
			return nil
		})
		if err == nil {
			for i, user := range users {
				status[user] = counts[i].Val() > 0
			}
			return status
		}
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	now := time.Now()
	for _, user := range users {
		status[user] = activePresence(h.presence[user], now)
	}
	return status
}
