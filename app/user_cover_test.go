package app

import (
	"testing"

	"litloop/backend/internal/models"
)

// Uploading a profile cover stores the image URL on the user.
func TestUserCoverUpload(tester *testing.T) {
	fixture := setup(tester)
	media := []byte("\x89PNG\r\n\x1a\ncover")
	cover := multipartTestUpload(tester, fixture, "/users/me/cover", "file", "cover.png", media, 200)
	uri, _ := cover["cover_image"].(string)
	if uri == "" {
		tester.Fatalf("expected cover_image in response, got %v", cover)
	}
	var user models.User
	if err := fixture.s.DB.First(&user, fixture.user).Error; err != nil {
		tester.Fatal(err)
	}
	if user.CoverImage != uri {
		tester.Fatalf("cover not stored: %q vs %q", user.CoverImage, uri)
	}
	detail := fixture.req("GET", "/users/me", fixture.Token, nil, 200)
	if detail["cover_image"] != uri {
		tester.Fatalf("cover missing from profile: %v", detail)
	}
}
