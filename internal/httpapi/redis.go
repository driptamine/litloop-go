package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisBus struct {
	client       *redis.Client
	subscription *redis.PubSub
	cancel       context.CancelFunc
	Once         sync.Once
}

func (h *Hub) ConnectRedis(raw string) error {
	opts, e := redis.ParseURL(raw)
	if e != nil {
		return e
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithCancel(context.Background())
	pingctx, stop := context.WithTimeout(ctx, 5*time.Second)
	e = client.Ping(pingctx).Err()
	stop()
	if e != nil {
		cancel()
		_ = client.Close()
		return e
	}
	sub := client.Subscribe(ctx, "litloop:realtime")
	if _, e = sub.Receive(ctx); e != nil {
		cancel()
		_ = sub.Close()
		_ = client.Close()
		return e
	}
	h.bus = &redisBus{client: client, subscription: sub, cancel: cancel}
	go func() {
		for msg := range sub.Channel() {
			var event struct {
				Room string          `json:"room"`
				Data json.RawMessage `json:"data"`
			}
			if json.Unmarshal([]byte(msg.Payload), &event) == nil {
				h.broadcastLocal(event.Room, event.Data)
			}
		}
	}()
	return nil
}
func (h *Hub) publish(room string, v any) {
	if h.bus == nil {
		h.broadcastLocal(room, v)
		return
	}
	raw, e := json.Marshal(map[string]any{"room": room, "data": v})
	if e != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if e = h.bus.client.Publish(ctx, "litloop:realtime", raw).Err(); e != nil {
		slog.Error("realtime publish", "error", e)
		h.broadcastLocal(room, v)
	}
}
func (h *Hub) Close() {
	if h.bus != nil {
		h.bus.Once.Do(func() { h.bus.cancel(); _ = h.bus.subscription.Close(); _ = h.bus.client.Close() })
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, room := range h.rooms {
		for p := range room {
			p.Close()
		}
	}
}
