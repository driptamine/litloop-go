package app

import (
	"testing"
)

func TestUpdateProfile(tester *testing.T) {
	fixture := setup(tester)
	fixture.second()

	updated := fixture.req("PUT", "/users/me", fixture.Token, map[string]any{"username": "alice2"}, 200)
	if updated["username"] != "alice2" {
		tester.Fatalf("expected renamed user, got %v", updated)
	}
	fixture.req("PUT", "/users/me", fixture.Token, map[string]any{"username": "bob"}, 409)
	fixture.req("PUT", "/users/me", fixture.Token, map[string]any{"username": "  "}, 400)
	fixture.req("PUT", "/users/me", fixture.Token, map[string]any{"email": "not-an-email"}, 400)
	fixture.req("PUT", "/users/me", fixture.Token, map[string]any{}, 400)

	fixture.req("POST", "/users/me/password", fixture.Token, map[string]any{"old_password": "wrong", "new_password": "newpassword123"}, 401)
	fixture.req("POST", "/users/me/password", fixture.Token, map[string]any{"old_password": "password123", "new_password": "short"}, 400)
	fixture.req("POST", "/users/me/password", fixture.Token, map[string]any{"old_password": "password123", "new_password": "newpassword123"}, 200)

	signed := fixture.req("POST", "/users/signin", "", map[string]any{"email_or_username": "alice2", "password": "newpassword123"}, 200)
	if signed["access"] == nil || signed["access"] == "" {
		tester.Fatal("expected access token after password change")
	}
}
