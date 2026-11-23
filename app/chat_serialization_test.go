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

func (counter queryCounter) LogMode(logger.LogLevel) logger.Interface	{ return counter }
func (counter queryCounter) Info(context.Context, string, ...any)	{}
func (counter queryCounter) Warn(context.Context, string, ...any)	{}
func (counter queryCounter) Error(context.Context, string, ...any)	{}
func (counter queryCounter) Trace(_ context.Context, _ time.Time, _ func() (string, int64), _ error) {
	counter.n.Add(1)
}

// A chat with history must serialize with a bounded number of queries,
// otherwise detail loads time out (the app aborts them after 12s).
func TestChatDetailQueryCountStaysBounded(tester *testing.T) {
	fixture := setup(tester)
	bob, bobID := fixture.second()
	chat := fixture.req("GET", fmt.Sprintf("/chats/direct/%d", bobID), fixture.Token, nil, 200)
	chatID := int64(chat["id"].(float64))

	const total = 25
	for index := 0; index < total; index++ {
		fixture.req("POST", fmt.Sprintf("/chats/%d/send", chatID), fixture.Token,
			map[string]any{"text": fmt.Sprintf("message %d", index), "attachments": []any{}}, 200)
	}

	var count atomic.Int64
	fixture.s.DB.Logger = queryCounter{n: &count}

	detail := fixture.req("GET", fmt.Sprintf("/chats/direct/%d", bobID), fixture.Token, nil, 200)
	messages := detail["messages"].([]any)
	if len(messages) != total {
		tester.Fatalf("expected %d messages, got %d", total, len(messages))
	}
	if first := messages[0].(map[string]any)["text"]; first != "message 0" {
		tester.Fatalf("expected oldest message first, got %v", first)
	}
	if loaded := count.Load(); loaded > 15 {
		tester.Fatalf("chat detail issued %d queries for %d messages, want at most 15", loaded, total)
	}

	count.Store(0)
	history := fixture.req("GET", fmt.Sprintf("/chats/%d/messages", chatID), bob, nil, 200)
	if got := len(history["messages"].([]any)); got != total {
		tester.Fatalf("expected %d messages, got %d", total, got)
	}
	if loaded := count.Load(); loaded > 10 {
		tester.Fatalf("message history issued %d queries for %d messages, want at most 10", loaded, total)
	}
}
