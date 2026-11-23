package app

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

// queryCounter is a silent gorm logger that counts executed statements.
type queryCounter struct{ n *atomic.Int64 }

func (c queryCounter) LogMode(logger.LogLevel) logger.Interface { return c }
func (c queryCounter) Info(context.Context, string, ...any)     {}
func (c queryCounter) Warn(context.Context, string, ...any)     {}
func (c queryCounter) Error(context.Context, string, ...any)    {}
func (c queryCounter) Trace(_ context.Context, _ time.Time, _ func() (string, int64), _ error) {
	c.n.Add(1)
}

// A chat with history must serialize with a bounded number of queries,
// otherwise detail loads time out (the app aborts them after 12s).
func TestChatDetailQueryCountStaysBounded(t *testing.T) {
	f := setup(t)
	bob, bobID := f.second()
	chat := f.req("GET", fmt.Sprintf("/chats/direct/%d", bobID), f.Token, nil, 200)
	chatID := int64(chat["id"].(float64))

	const total = 25
	for i := 0; i < total; i++ {
		f.req("POST", fmt.Sprintf("/chats/%d/send", chatID), f.Token,
			map[string]any{"text": fmt.Sprintf("message %d", i), "attachments": []any{}}, 200)
	}

	var count atomic.Int64
	f.s.DB.Logger = queryCounter{n: &count}

	detail := f.req("GET", fmt.Sprintf("/chats/direct/%d", bobID), f.Token, nil, 200)
	messages := detail["messages"].([]any)
	if len(messages) != total {
		t.Fatalf("expected %d messages, got %d", total, len(messages))
	}
	if first := messages[0].(map[string]any)["text"]; first != "message 0" {
		t.Fatalf("expected oldest message first, got %v", first)
	}
	if n := count.Load(); n > 15 {
		t.Fatalf("chat detail issued %d queries for %d messages, want at most 15", n, total)
	}

	count.Store(0)
	history := f.req("GET", fmt.Sprintf("/chats/%d/messages", chatID), bob, nil, 200)
	if got := len(history["messages"].([]any)); got != total {
		t.Fatalf("expected %d messages, got %d", total, got)
	}
	if n := count.Load(); n > 10 {
		t.Fatalf("message history issued %d queries for %d messages, want at most 10", n, total)
	}
}
