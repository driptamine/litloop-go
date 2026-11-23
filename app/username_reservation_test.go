package app

import (
	"testing"

	"litloop/backend/internal/models"
)

// Username reservation: an admin reserves handles for a user, others cannot
// take them, the assignee can, and releasing frees the name.
func TestUsernameReservationRoundTrip(tester *testing.T) {
	fixture := setup(tester)
	bobToken, _ := fixture.second()
	if err := fixture.s.DB.Model(&models.User{}).Where("id = ?", fixture.user).Update("is_superuser", true).Error; err != nil {
		tester.Fatal(err)
	}

	fixture.req("POST", "/admin/usernames", bobToken, map[string]any{
		"usernames": "neo", "assign_to": "bob",
	}, 403)

	saved := fixture.req("POST", "/admin/usernames", fixture.Token, map[string]any{
		"usernames": "neo, trinity\nmorpheus, neo", "assign_to": "bob",
	}, 200)
	if len(saved["reserved"].([]any)) != 3 {
		tester.Fatalf("expected 3 reservations, got %v", saved)
	}
	if len(saved["skipped"].([]any)) != 1 {
		tester.Fatalf("expected 1 duplicate skip, got %v", saved)
	}

	listed := fixture.req("GET", "/admin/usernames", fixture.Token, nil, 200)
	if got, _ := listed["usernames"].([]any); len(got) != 3 {
		tester.Fatalf("expected 3 listed, got %v", listed)
	}

	fixture.req("PUT", "/users/me", bobToken, map[string]any{"username": "neo"}, 200)
	fixture.req("PUT", "/users/me", fixture.Token, map[string]any{"username": "trinity"}, 409)

	signed := fixture.req("POST", "/users/signup", "", map[string]any{
		"email": "fan@example.com", "username": "morpheus", "password": "password123",
	}, 201)
	if signed["username"] == "morpheus" {
		tester.Fatalf("reserved name was handed out at signup: %v", signed)
	}

	fixture.req("DELETE", "/admin/usernames/morpheus", fixture.Token, nil, 200)
	fixture.req("PUT", "/users/me", fixture.Token, map[string]any{"username": "morpheus"}, 200)
	me := fixture.req("GET", "/users/me", fixture.Token, nil, 200)
	if me["username"] != "morpheus" {
		tester.Fatalf("released name was not taken: %v", me)
	}

	fixture.req("POST", "/admin/usernames", fixture.Token, map[string]any{
		"usernames": "ghost", "assign_to": "nobody-here",
	}, 404)
}
