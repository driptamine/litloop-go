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
	client		*redis.Client
	subscription	*redis.PubSub
	cancel		context.CancelFunc
	Once		sync.Once
}

func (hub *Hub) ConnectRedis(raw string) error {
	opts, err := redis.ParseURL(raw)
	if err != nil {
		return err
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithCancel(context.Background())
	pingctx, stop := context.WithTimeout(ctx, 5*time.Second)
	err = client.Ping(pingctx).Err()
	stop()
	if err != nil {
		cancel()
		_ = client.Close()
		return err
	}
	sub := client.Subscribe(ctx, "litloop:realtime")
	if _, err = sub.Receive(ctx); err != nil {
		cancel()
		_ = sub.Close()
		_ = client.Close()
		return err
	}
	hub.bus = &redisBus{client: client, subscription: sub, cancel: cancel}
	go func() {
		for msg := range sub.Channel() {
			var event struct {
				Room	string		`json:"room"`
				Data	json.RawMessage	`json:"data"`
			}
			if json.Unmarshal([]byte(msg.Payload), &event) == nil {
				hub.broadcastLocal(event.Room, event.Data)
			}
		}
	}()
	return nil
}
func (hub *Hub) publish(room string, value any) {
	if hub.bus == nil {
		hub.broadcastLocal(room, value)
		return
	}
	raw, err := json.Marshal(map[string]any{"room": room, "data": value})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err = hub.bus.client.Publish(ctx, "litloop:realtime", raw).Err(); err != nil {
		slog.Error("realtime publish", "error", err)
		hub.broadcastLocal(room, value)
	}
}
func (hub *Hub) Close() {
	if hub.bus != nil {
		hub.bus.Once.Do(func() { hub.bus.cancel(); _ = hub.bus.subscription.Close(); _ = hub.bus.client.Close() })
	}
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	for _, room := range hub.rooms {
		for peer := range room {
			peer.Close()
		}
	}
}
