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

func (counter queryCounter) LogMode(logger.LogLevel) logger.Interface { return counter }
func (counter queryCounter) Info(context.Context, string, ...any)     {}
func (counter queryCounter) Warn(context.Context, string, ...any)     {}
func (counter queryCounter) Error(context.Context, string, ...any)    {}
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

// Replying quotes the parent message: the reply carries reply_to and an
// embedded reply_to_message snapshot, and a reply_to from another chat is
// rejected.
func TestMessageReplyRoundTrip(tester *testing.T) {
	fixture := setup(tester)
	bob, bobID := fixture.second()
	chat := fixture.req("GET", fmt.Sprintf("/chats/direct/%d", bobID), fixture.Token, nil, 200)
	chatID := int64(chat["id"].(float64))
	send := fmt.Sprintf("/chats/%d/send", chatID)

	parent := fixture.req("POST", send, fixture.Token, map[string]any{"text": "parent message"}, 200)
	parentID := int64(parent["message"].(map[string]any)["id"].(float64))

	reply := fixture.req("POST", send, bob, map[string]any{"text": "child reply", "reply_to": parentID}, 200)
	replied := reply["message"].(map[string]any)
	if int64(replied["reply_to"].(float64)) != parentID {
		tester.Fatalf("expected reply_to %d, got %v", parentID, replied["reply_to"])
	}
	quote, ok := replied["reply_to_message"].(map[string]any)
	if !ok {
		tester.Fatalf("expected reply_to_message snapshot, got %v", replied["reply_to_message"])
	}
	if int64(quote["id"].(float64)) != parentID || quote["text"] != "parent message" || quote["username"] != "alice" {
		tester.Fatalf("unexpected quote snapshot: %v", quote)
	}

	detail := fixture.req("GET", fmt.Sprintf("/chats/direct/%d", bobID), fixture.Token, nil, 200)
	messages := detail["messages"].([]any)
	last := messages[len(messages)-1].(map[string]any)
	if int64(last["reply_to"].(float64)) != parentID {
		tester.Fatalf("chat detail lost reply_to: %v", last)
	}

	group := fixture.req("POST", "/chats/group/create", fixture.Token, map[string]any{"name": "replytest", "participant_ids": []any{float64(bobID)}}, 201)
	groupID := int64(group["chat"].(map[string]any)["id"].(float64))
	fixture.req("POST", fmt.Sprintf("/chats/%d/send", groupID), fixture.Token, map[string]any{"text": "cross-chat", "reply_to": parentID}, 400)
	fixture.req("POST", send, bob, map[string]any{"text": "ghost", "reply_to": 999999}, 404)
}
