package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"litloop/backend/internal/models"
	"mime/multipart"
	"net/http"
	"testing"
)

func TestCommunitySubmissionCreatesPostAndPreservesModeration(t *testing.T) {
	f := setup(t)
	bob, _ := f.second()
	c := f.req("POST", "/communities/create", f.Token, map[string]any{"name": "Community"}, 201)
	cid := int64(c["id"].(float64))
	path := fmt.Sprintf("/communities/%d", cid)
	f.req("POST", path+"/join", bob, map[string]any{}, 200)
	submitted := f.req("POST", path+"/posts/request", bob, map[string]any{"title": "Submitted directly", "description": "Community content"}, 201)
	if submitted["status"] != "pending" || submitted["title"] != "Submitted directly" {
		t.Fatal(submitted)
	}
	requestID := int64(submitted["community_post_id"].(float64))
	approved := f.req("POST", fmt.Sprintf("%s/posts/%d/approve", path, requestID), f.Token, map[string]any{}, 200)
	if approved["status"] != "approved" || approved["requested_by"] == nil {
		t.Fatal(approved)
	}
	f.req("POST", fmt.Sprintf("%s/posts/%d/approve", path, requestID), f.Token, map[string]any{}, 400)
	published := f.req("GET", path+"/posts", "", nil, 200)
	if len(published["posts"].([]any)) != 1 {
		t.Fatal(published)
	}
	member := f.req("GET", path, bob, nil, 200)
	if member["user_is_member"] != true || member["member_count"] != float64(2) {
		t.Fatal(member)
	}
}
func TestTrackEditAlbumArtistsAndDate(t *testing.T) {
	f := setup(t)
	album := models.Album{Name: "Album", AlbumURI: "album-test"}
	if err := f.s.DB.Create(&album).Error; err != nil {
		t.Fatal(err)
	}
	track := models.Track{Name: "Old", TrackURI: "track-test", UserID: &f.user}
	if err := f.s.DB.Create(&track).Error; err != nil {
		t.Fatal(err)
	}
	edited := f.req("PATCH", "/tracks/track-test/upd", f.Token, map[string]any{"name": "New", "album_uri": "album-test", "artist_names": "Alice; Bob", "release_date": "2026-11-12"}, 200)
	if edited["name"] != "New" || len(edited["artists"].([]any)) != 2 || edited["release_date"] == nil {
		t.Fatal(edited)
	}
	var stored models.Track
	if err := f.s.DB.First(&stored, track.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.AlbumID == nil || *stored.AlbumID != album.ID {
		t.Fatal(stored)
	}
}
func TestMultipartPostAndArrayUserFeed(t *testing.T) {
	f := setup(t)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("title", "Uploaded photo")
	_ = writer.WriteField("description", "Multipart content")
	part, err := writer.CreateFormFile("photos", "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("\x89PNG\r\n\x1a\nimage-data"))
	_ = writer.Close()
	request, _ := http.NewRequest("POST", f.http.URL+"/posts/create/postman", &body)
	request.Header.Set("Authorization", "Bearer "+f.Token)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := f.http.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode != 201 {
		t.Fatalf("%d %s", response.StatusCode, raw)
	}
	var post map[string]any
	if err = json.Unmarshal(raw, &post); err != nil {
		t.Fatal(err)
	}
	if post["success"] != true || len(post["photos"].([]any)) != 1 {
		t.Fatal(post)
	}
	feed, err := f.http.Client().Get(f.http.URL + "/posts/user/alice")
	if err != nil {
		t.Fatal(err)
	}
	defer feed.Body.Close()
	var posts []any
	if err = json.NewDecoder(feed.Body).Decode(&posts); err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 {
		t.Fatal(posts)
	}
}
func TestMPTTIndexesAfterPageMutations(t *testing.T) {
	f := setup(t)
	parent := f.req("POST", "/notes/pages/create", f.Token, map[string]any{"title": "Parent"}, 201)
	id := int64(parent["id"].(float64))
	child := f.req("POST", "/notes/pages/create", f.Token, map[string]any{"title": "Child", "parent_id": id}, 201)
	childID := int64(child["id"].(float64))
	var p, c models.Page
	if err := f.s.DB.First(&p, id).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.s.DB.First(&c, childID).Error; err != nil {
		t.Fatal(err)
	}
	if p.Lft != 1 || p.Rght != 4 || c.Lft != 2 || c.Rght != 3 || c.Level != 1 || p.TreeID != c.TreeID {
		t.Fatal(p, c)
	}
	f.req("DELETE", fmt.Sprintf("/notes/pages/%d/delete", childID), f.Token, nil, 200)
	if err := f.s.DB.First(&p, id).Error; err != nil {
		t.Fatal(err)
	}
	if p.Rght != 2 {
		t.Fatal(p)
	}
}
