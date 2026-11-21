package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	callSignalTTL      = 2 * time.Minute
	maxCallSignals     = 128
	maxCallSignalBytes = 64 << 10
)

type callSignalBuffer struct {
	events []json.RawMessage
}

func callSignalsKey(callID, userID int64) string {
	return fmt.Sprintf("litloop:call-signals:%d:%d", callID, userID)
}

// SaveCallSignal retains setup signals for the recipient during reconnects.
func (h *Hub) SaveCallSignal(callID, userID int64, event any) error {
	return h.saveCallSignal(callSignalsKey(callID, userID), event)
}

func (h *Hub) SaveGroupVoiceSignal(callID, userID int64, session string, event any) error {
	return h.saveCallSignal(callSignalsKey(callID, userID)+":"+session, event)
}

func (h *Hub) saveCallSignal(key string, event any) error {
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if len(raw) > maxCallSignalBytes {
		Fail(400, "Call signal is too large")
	}
	if h.bus != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err = h.bus.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.RPush(ctx, key, raw)
			pipe.LTrim(ctx, key, -maxCallSignals, -1)
			pipe.Expire(ctx, key, callSignalTTL)
			return nil
		})
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	buffer := h.callSignals[key]
	if buffer == nil {
		buffer = &callSignalBuffer{}
		h.callSignals[key] = buffer
		time.AfterFunc(callSignalTTL, func() {
			h.mu.Lock()
			delete(h.callSignals, key)
			h.mu.Unlock()
		})
	}
	if len(buffer.events) == maxCallSignals {
		buffer.events = buffer.events[1:]
	}
	buffer.events = append(buffer.events, raw)
	return nil
}

func (h *Hub) CallSignals(callID, userID int64) ([]json.RawMessage, error) {
	return h.callSignalEvents(callSignalsKey(callID, userID))
}

func (h *Hub) GroupVoiceSignals(callID, userID int64, session string) ([]json.RawMessage, error) {
	return h.callSignalEvents(callSignalsKey(callID, userID) + ":" + session)
}

func (h *Hub) callSignalEvents(key string) ([]json.RawMessage, error) {
	if h.bus != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		rows, err := h.bus.client.LRange(ctx, key, 0, -1).Result()
		if err != nil {
			return nil, err
		}
		out := make([]json.RawMessage, len(rows))
		for i, raw := range rows {
			out[i] = json.RawMessage(raw)
		}
		return out, nil
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if buffer := h.callSignals[key]; buffer != nil {
		return append([]json.RawMessage{}, buffer.events...), nil
	}
	return []json.RawMessage{}, nil
}
