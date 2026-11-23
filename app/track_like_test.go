package app

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
)

// Liking a track toggles the like and feeds the user's audio library.
func TestTrackLikeRoundTrip(tester *testing.T) {
	fixture := setup(tester)
	init := fixture.req("POST", "/tracks/create_presigned_url", fixture.Token, map[string]any{
		"filename": "song.mp3", "content_type": "audio/mpeg", "media_type": "track",
	}, 200)
	uploadID, _ := init["upload_id"].(string)
	putReq, err := http.NewRequest(http.MethodPut, fixture.http.URL+"/uploads/"+uploadID, bytes.NewReader([]byte("fake mp3 bytes")))
	if err != nil {
		tester.Fatal(err)
	}
	putReq.Header.Set("Authorization", "Bearer "+fixture.Token)
	putResp, err := fixture.http.Client().Do(putReq)
	if err != nil {
		tester.Fatal(err)
	}
	putResp.Body.Close()
	if putResp.StatusCode != 200 {
		tester.Fatalf("expected 200 for local upload, got %d", putResp.StatusCode)
	}
	done := fixture.req("POST", "/tracks/complete_upload", fixture.Token, map[string]any{
		"upload_id": uploadID, "key": init["key"], "media_type": "track", "name": "Like Song",
	}, 200)
	trackID := int64(done["id"].(float64))

	status := fixture.req("GET", fmt.Sprintf("/tracks/like/%d", trackID), fixture.Token, nil, 200)
	if status["liked"] != false {
		tester.Fatalf("expected unliked, got %v", status)
	}
	liked := fixture.req("POST", fmt.Sprintf("/tracks/like/%d", trackID), fixture.Token, map[string]any{}, 200)
	if liked["liked"] != true {
		tester.Fatalf("expected liked, got %v", liked)
	}
	library := fixture.req("GET", "/tracks/liked", fixture.Token, nil, 200)
	tracks, _ := library["tracks"].([]any)
	if len(tracks) != 1 {
		tester.Fatalf("expected 1 liked track, got %v", library)
	}
	unliked := fixture.req("POST", fmt.Sprintf("/tracks/like/%d", trackID), fixture.Token, map[string]any{}, 200)
	if unliked["liked"] != false {
		tester.Fatalf("expected unliked, got %v", unliked)
	}
	empty := fixture.req("GET", "/tracks/liked", fixture.Token, nil, 200)
	if got, _ := empty["tracks"].([]any); len(got) != 0 {
		tester.Fatalf("expected empty library, got %v", empty)
	}
	fixture.req("POST", "/tracks/like/999999", fixture.Token, map[string]any{}, 404)
}
