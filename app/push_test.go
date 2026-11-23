package app

import (
	"fmt"
	"testing"
)

// Subscribing stores a browser push endpoint, and a direct message to a
// subscribed user is accepted (delivery itself is best-effort and only
// logged). VAPID keys are stubbed because key generation is an ops step.
func TestPushSubscribeAndDirectMessage(tester *testing.T) {
	fixture := setup(tester)
	fixture.s.Config.VAPIDPublicKey = "test-public-key"
	fixture.s.Config.VAPIDPrivateKey = "test-private-key"

	key := fixture.req("GET", "/push/vapid-public-key", "", nil, 200)
	if key["publicKey"] != "test-public-key" {
		tester.Fatalf("expected stubbed VAPID key, got %v", key)
	}

	endpoint := "https://push.example.test/endpoint/1"
	fixture.req("POST", "/push/subscribe", fixture.Token, map[string]any{
		"endpoint": endpoint, "keys": map[string]any{"p256dh": "p256dh-test", "auth": "auth-test"},
	}, 200)
	fixture.req("POST", "/push/subscribe", fixture.Token, map[string]any{
		"endpoint": endpoint, "keys": map[string]any{"p256dh": "p256dh-test", "auth": "auth-test"},
	}, 200)

	bob, bobID := fixture.second()
	chat := fixture.req("GET", fmt.Sprintf("/chats/direct/%d", bobID), fixture.Token, nil, 200)
	chatID := int64(chat["id"].(float64))
	fixture.req("POST", fmt.Sprintf("/chats/%d/send", chatID), bob, map[string]any{"text": "push hello"}, 200)

	fixture.req("POST", "/push/unsubscribe", fixture.Token, map[string]any{"endpoint": endpoint}, 200)
	fixture.req("POST", "/push/subscribe", fixture.Token, map[string]any{"endpoint": ""}, 400)
}
