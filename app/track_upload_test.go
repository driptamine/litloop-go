package app

import (
	"bytes"
	"fmt"
	"litloop/backend/internal/models"
	"net/http"
	"testing"
)

// Completing a local track upload with extracted metadata stores the name,
// duration and artist link, and echoes them in the response resource.
func TestTrackUploadStoresMetadata(tester *testing.T) {
	fixture := setup(tester)
	init := fixture.req("POST", "/tracks/create_presigned_url", fixture.Token, map[string]any{
		"filename": "song.mp3", "content_type": "audio/mpeg", "media_type": "track",
	}, 200)
	uploadID, _ := init["upload_id"].(string)
	if uploadID == "" {
		tester.Fatalf("expected upload_id, got %v", init)
	}
	put, err := http.NewRequest(http.MethodPut, fixture.http.URL+"/uploads/"+uploadID, bytes.NewReader([]byte("fake mp3 bytes")))
	if err != nil {
		tester.Fatal(err)
	}
	put.Header.Set("Authorization", "Bearer "+fixture.Token)
	putResponse, err := fixture.http.Client().Do(put)
	if err != nil {
		tester.Fatal(err)
	}
	putResponse.Body.Close()
	if putResponse.StatusCode != 200 {
		tester.Fatalf("expected 200 for local upload, got %d", putResponse.StatusCode)
	}
	done := fixture.req("POST", "/tracks/complete_upload", fixture.Token, map[string]any{
		"upload_id": uploadID, "key": init["key"], "media_type": "track",
		"name": "Test Song", "artist": "Test Artist", "duration": 187.5,
	}, 200)
	trackID := int64(done["id"].(float64))
	var track models.Track
	if err := fixture.s.DB.First(&track, trackID).Error; err != nil {
		tester.Fatal(err)
	}
	if track.Name != "Test Song" || track.Duration != 187.5 {
		tester.Fatalf("metadata not stored: %+v", track)
	}
	var links []models.TrackArtist
	if err := fixture.s.DB.Where("track_id = ?", trackID).Find(&links).Error; err != nil || len(links) != 1 {
		tester.Fatalf("expected one artist link, got %v %v", links, err)
	}
	var artist models.Artist
	if err := fixture.s.DB.First(&artist, links[0].ArtistID).Error; err != nil || artist.Name != "Test Artist" {
		tester.Fatalf("artist not linked: %v %v", artist, err)
	}
	resource, _ := done["resource"].(map[string]any)
	if fmt.Sprint(resource["name"]) != "Test Song" {
		tester.Fatalf("resource echo lost the name: %v", resource)
	}
}
