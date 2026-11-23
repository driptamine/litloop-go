package app

import (
	"testing"
)

func TestUpdateProfile(t *testing.T) {
	f := setup(t)
	f.second()

	updated := f.req("PUT", "/users/me", f.Token, map[string]any{"username": "alice2"}, 200)
	if updated["username"] != "alice2" {
		t.Fatalf("expected renamed user, got %v", updated)
	}
	f.req("PUT", "/users/me", f.Token, map[string]any{"username": "bob"}, 409)
	f.req("PUT", "/users/me", f.Token, map[string]any{"username": "  "}, 400)
	f.req("PUT", "/users/me", f.Token, map[string]any{"email": "not-an-email"}, 400)
	f.req("PUT", "/users/me", f.Token, map[string]any{}, 400)

	f.req("POST", "/users/me/password", f.Token, map[string]any{"old_password": "wrong", "new_password": "newpassword123"}, 401)
	f.req("POST", "/users/me/password", f.Token, map[string]any{"old_password": "password123", "new_password": "short"}, 400)
	f.req("POST", "/users/me/password", f.Token, map[string]any{"old_password": "password123", "new_password": "newpassword123"}, 200)

	signed := f.req("POST", "/users/signin", "", map[string]any{"email_or_username": "alice2", "password": "newpassword123"}, 200)
	if signed["access"] == nil || signed["access"] == "" {
		t.Fatal("expected access token after password change")
	}
}
