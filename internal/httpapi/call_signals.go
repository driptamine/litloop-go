package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	callSignalTTL		= 2 * time.Minute
	maxCallSignals		= 128
	maxCallSignalBytes	= 64 << 10
)

type callSignalBuffer struct {
	events []json.RawMessage
}

func callSignalsKey(callID, userID int64) string {
	return fmt.Sprintf("litloop:call-signals:%d:%d", callID, userID)
}

// SaveCallSignal retains setup signals for the recipient during reconnects.
func (hub *Hub) SaveCallSignal(callID, userID int64, event any) error {
	return hub.saveCallSignal(callSignalsKey(callID, userID), event)
}

func (hub *Hub) SaveGroupVoiceSignal(callID, userID int64, session string, event any) error {
	return hub.saveCallSignal(callSignalsKey(callID, userID)+":"+session, event)
}

func (hub *Hub) saveCallSignal(key string, event any) error {
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if len(raw) > maxCallSignalBytes {
		Fail(400, "Call signal is too large")
	}
	if hub.bus != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err = hub.bus.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.RPush(ctx, key, raw)
			pipe.LTrim(ctx, key, -maxCallSignals, -1)
			pipe.Expire(ctx, key, callSignalTTL)
			return nil
		})
		return err
	}

	hub.mu.Lock()
	defer hub.mu.Unlock()
	buffer := hub.callSignals[key]
	if buffer == nil {
		buffer = &callSignalBuffer{}
		hub.callSignals[key] = buffer
		time.AfterFunc(callSignalTTL, func() {
			hub.mu.Lock()
			delete(hub.callSignals, key)
			hub.mu.Unlock()
		})
	}
	if len(buffer.events) == maxCallSignals {
		buffer.events = buffer.events[1:]
	}
	buffer.events = append(buffer.events, raw)
	return nil
}

func (hub *Hub) CallSignals(callID, userID int64) ([]json.RawMessage, error) {
	return hub.callSignalEvents(callSignalsKey(callID, userID))
}

func (hub *Hub) GroupVoiceSignals(callID, userID int64, session string) ([]json.RawMessage, error) {
	return hub.callSignalEvents(callSignalsKey(callID, userID) + ":" + session)
}

func (hub *Hub) callSignalEvents(key string) ([]json.RawMessage, error) {
	if hub.bus != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		rows, err := hub.bus.client.LRange(ctx, key, 0, -1).Result()
		if err != nil {
			return nil, err
		}
		out := make([]json.RawMessage, len(rows))
		for index, raw := range rows {
			out[index] = json.RawMessage(raw)
		}
		return out, nil
	}
	hub.mu.RLock()
	defer hub.mu.RUnlock()
	if buffer := hub.callSignals[key]; buffer != nil {
		return append([]json.RawMessage{}, buffer.events...), nil
	}
	return []json.RawMessage{}, nil
}
