package app

import (
	"testing"
)

// Chat tabs persist per user: PUT replaces the whole folder list, GET
// returns it back in order. Bad payloads are rejected.
func TestChatTabsRoundTrip(tester *testing.T) {
	fixture := setup(tester)

	empty := fixture.req("GET", "/chats/tabs", fixture.Token, nil, 200)
	if tabs, _ := empty["tabs"].([]any); len(tabs) != 0 {
		tester.Fatalf("expected no tabs, got %v", empty)
	}

	saved := fixture.req("PUT", "/chats/tabs", fixture.Token, map[string]any{
		"tabs": []any{
			map[string]any{"emoji": "❤️", "chat_ids": []any{float64(131), float64(47), float64(131)}},
			map[string]any{"emoji": "🔥", "chat_ids": []any{}},
		},
	}, 200)
	tabs, _ := saved["tabs"].([]any)
	if len(tabs) != 2 {
		tester.Fatalf("expected 2 tabs, got %v", saved)
	}
	first := tabs[0].(map[string]any)
	if first["emoji"] != "❤️" {
		tester.Fatalf("emoji not stored: %v", first)
	}
	ids, _ := first["chat_ids"].([]any)
	if len(ids) != 2 || int(ids[0].(float64)) != 131 || int(ids[1].(float64)) != 47 {
		tester.Fatalf("chat ids not deduped in order: %v", first)
	}

	loaded := fixture.req("GET", "/chats/tabs", fixture.Token, nil, 200)
	if got, _ := loaded["tabs"].([]any); len(got) != 2 {
		tester.Fatalf("tabs not persisted: %v", loaded)
	}

	fixture.req("PUT", "/chats/tabs", fixture.Token, map[string]any{"tabs": "nope"}, 400)
	fixture.req("PUT", "/chats/tabs", fixture.Token, map[string]any{
		"tabs": []any{map[string]any{"emoji": "", "chat_ids": []any{}}},
	}, 400)
}
