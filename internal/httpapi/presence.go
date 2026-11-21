package httpapi

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const presenceLease = 80 * time.Second

func presenceKey(user int64) string {
	return "litloop:presence:" + strconv.FormatInt(user, 10)
}

// SetPresence keeps one lease per authenticated notification connection.
func (h *Hub) SetPresence(user int64, session string, connected bool) (online, changed bool) {
	h.mu.Lock()
	sessions := h.presence[user]
	wasOnline := len(sessions) > 0
	if connected {
		if sessions == nil {
			sessions = make(map[string]struct{})
			h.presence[user] = sessions
		}
		sessions[session] = struct{}{}
	} else {
		delete(sessions, session)
		if len(sessions) == 0 {
			delete(h.presence, user)
		}
	}
	online = len(sessions) > 0
	h.mu.Unlock()
	if h.bus == nil {
		return online, wasOnline != online
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	key := presenceKey(user)
	now := time.Now()
	cutoff := strconv.FormatInt(now.UnixMilli(), 10)
	var before, after *redis.IntCmd
	_, err := h.bus.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.ZRemRangeByScore(ctx, key, "-inf", cutoff)
		before = pipe.ZCard(ctx, key)
		if connected {
			pipe.ZAdd(ctx, key, redis.Z{Member: session, Score: float64(now.Add(presenceLease).UnixMilli())})
			pipe.Expire(ctx, key, presenceLease)
		} else {
			pipe.ZRem(ctx, key, session)
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
	for _, user := range users {
		status[user] = len(h.presence[user]) > 0
	}
	return status
}
