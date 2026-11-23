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

func TestCommunitySubmissionCreatesPostAndPreservesModeration(tester *testing.T) {
	fixture := setup(tester)
	bob, _ := fixture.second()
	response := fixture.req("POST", "/communities/create", fixture.Token, map[string]any{"name": "Community"}, 201)
	cid := int64(response["id"].(float64))
	path := fmt.Sprintf("/communities/%d", cid)
	fixture.req("POST", path+"/join", bob, map[string]any{}, 200)
	submitted := fixture.req("POST", path+"/posts/request", bob, map[string]any{"title": "Submitted directly", "description": "Community content"}, 201)
	if submitted["status"] != "pending" || submitted["title"] != "Submitted directly" {
		tester.Fatal(submitted)
	}
	requestID := int64(submitted["community_post_id"].(float64))
	approved := fixture.req("POST", fmt.Sprintf("%s/posts/%d/approve", path, requestID), fixture.Token, map[string]any{}, 200)
	if approved["status"] != "approved" || approved["requested_by"] == nil {
		tester.Fatal(approved)
	}
	fixture.req("POST", fmt.Sprintf("%s/posts/%d/approve", path, requestID), fixture.Token, map[string]any{}, 400)
	published := fixture.req("GET", path+"/posts", "", nil, 200)
	if len(published["posts"].([]any)) != 1 {
		tester.Fatal(published)
	}
	member := fixture.req("GET", path, bob, nil, 200)
	if member["user_is_member"] != true || member["member_count"] != float64(2) {
		tester.Fatal(member)
	}
}
func TestTrackEditAlbumArtistsAndDate(tester *testing.T) {
	fixture := setup(tester)
	album := models.Album{Name: "Album", AlbumURI: "album-test"}
	if err := fixture.s.DB.Create(&album).Error; err != nil {
		tester.Fatal(err)
	}
	track := models.Track{Name: "Old", TrackURI: "track-test", UserID: &fixture.user}
	if err := fixture.s.DB.Create(&track).Error; err != nil {
		tester.Fatal(err)
	}
	edited := fixture.req("PATCH", "/tracks/track-test/upd", fixture.Token, map[string]any{"name": "New", "album_uri": "album-test", "artist_names": "Alice; Bob", "release_date": "2026-11-12"}, 200)
	if edited["name"] != "New" || len(edited["artists"].([]any)) != 2 || edited["release_date"] == nil {
		tester.Fatal(edited)
	}
	var stored models.Track
	if err := fixture.s.DB.First(&stored, track.ID).Error; err != nil {
		tester.Fatal(err)
	}
	if stored.AlbumID == nil || *stored.AlbumID != album.ID {
		tester.Fatal(stored)
	}
}
func TestMultipartPostAndArrayUserFeed(tester *testing.T) {
	fixture := setup(tester)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("title", "Uploaded photo")
	_ = writer.WriteField("description", "Multipart content")
	part, err := writer.CreateFormFile("photos", "photo.png")
	if err != nil {
		tester.Fatal(err)
	}
	_, _ = part.Write([]byte("\x89PNG\r\n\x1a\nimage-data"))
	_ = writer.Close()
	request, _ := http.NewRequest("POST", fixture.http.URL+"/posts/create/postman", &body)
	request.Header.Set("Authorization", "Bearer "+fixture.Token)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := fixture.http.Client().Do(request)
	if err != nil {
		tester.Fatal(err)
	}
	defer response.Body.Close()
	raw, _ := io.ReadAll(response.Body)
	if response.StatusCode != 201 {
		tester.Fatalf("%d %s", response.StatusCode, raw)
	}
	var post map[string]any
	if err = json.Unmarshal(raw, &post); err != nil {
		tester.Fatal(err)
	}
	if post["success"] != true || len(post["photos"].([]any)) != 1 {
		tester.Fatal(post)
	}
	feed, err := fixture.http.Client().Get(fixture.http.URL + "/posts/user/alice")
	if err != nil {
		tester.Fatal(err)
	}
	defer feed.Body.Close()
	var posts []any
	if err = json.NewDecoder(feed.Body).Decode(&posts); err != nil {
		tester.Fatal(err)
	}
	if len(posts) != 1 {
		tester.Fatal(posts)
	}
}
func TestMPTTIndexesAfterPageMutations(tester *testing.T) {
	fixture := setup(tester)
	parent := fixture.req("POST", "/notes/pages/create", fixture.Token, map[string]any{"title": "Parent"}, 201)
	id := int64(parent["id"].(float64))
	child := fixture.req("POST", "/notes/pages/create", fixture.Token, map[string]any{"title": "Child", "parent_id": id}, 201)
	childID := int64(child["id"].(float64))
	var parentPage, childPage models.Page
	if err := fixture.s.DB.First(&parentPage, id).Error; err != nil {
		tester.Fatal(err)
	}
	if err := fixture.s.DB.First(&childPage, childID).Error; err != nil {
		tester.Fatal(err)
	}
	if parentPage.Lft != 1 || parentPage.Rght != 4 || childPage.Lft != 2 || childPage.Rght != 3 || childPage.Level != 1 || childPage.TreeID != parentPage.TreeID {
		tester.Fatal(parentPage, childPage)
	}
	fixture.req("DELETE", fmt.Sprintf("/notes/pages/%d/delete", childID), fixture.Token, nil, 200)
	if err := fixture.s.DB.First(&parentPage, id).Error; err != nil {
		tester.Fatal(err)
	}
	if parentPage.Rght != 2 {
		tester.Fatal(parentPage)
	}
}
