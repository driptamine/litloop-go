package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"litloop/backend/internal/models"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestR2MediaUploadsAndPlayback(tester *testing.T) {
	fixture := setup(tester)
	fixture.s.Config.RequireR2Media = true
	remote := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, httpReq *http.Request) {
		if !strings.HasPrefix(httpReq.URL.Path, "/private/") {
			tester.Errorf("unexpected bucket path %s", httpReq.URL.Path)
			writer.WriteHeader(404)
			return
		}
		switch httpReq.Method {
		case http.MethodPut:
			_, _ = io.Copy(io.Discard, httpReq.Body)
		case http.MethodPost:
			_, _ = io.WriteString(writer, `<InitiateMultipartUploadResult><UploadId>test-upload</UploadId></InitiateMultipartUploadResult>`)
		case http.MethodGet:
			if strings.HasSuffix(httpReq.URL.Path, ".m3u8") {
				_, _ = io.WriteString(writer, "#EXTM3U\n#EXTINF:5,\nsegment-00000.ts\n")
			}
		default:
			writer.WriteHeader(405)
		}
	}))
	defer remote.Close()
	fixture.s.Config.R2Endpoint, fixture.s.Config.R2Bucket = remote.URL, "private"
	fixture.s.Config.R2AccessKey, fixture.s.Config.R2SecretKey = "test-access", "test-secret"

	media := []byte("\x89PNG\r\n\x1a\nattachment")
	avatar := multipartTestUpload(tester, fixture, "/users/me/avatar/", "avatar", "portrait.png", media, 200)
	var asset models.Asset
	if err := fixture.s.DB.First(&asset).Error; err != nil || asset.Provider != "r2" || !strings.HasSuffix(avatar["avatar"].(string), "/assets/"+asset.ID) {
		tester.Fatalf("avatar did not store a stable R2 asset: %v", err)
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	redirect, err := client.Get(avatar["avatar"].(string))
	if err != nil {
		tester.Fatal(err)
	}
	redirect.Body.Close()
	if redirect.StatusCode != 307 || !strings.HasPrefix(redirect.Header.Get("Location"), remote.URL+"/private/") {
		tester.Fatal("avatar did not redirect to private R2")
	}

	multipartTestUpload(tester, fixture, "/chats/voice/upload/", "file", "voice.webm", []byte("voice data"), 201)
	var voice models.VoiceMessage
	if err := fixture.s.DB.First(&voice).Error; err != nil || voice.R2Key == "" || voice.S3Key != "" {
		tester.Fatalf("voice upload did not use R2: %v", err)
	}
	album := fixture.req("POST", "/photos/album/create", fixture.Token, map[string]any{"title": "Album"}, 201)
	multipartTestUpload(tester, fixture, fmt.Sprintf("/photos/album/%.0f/upload", album["id"].(float64)), "file", "album.png", media, 201)
	multipartTestUpload(tester, fixture, "/posts/create", "photos", "post.png", media, 201)
	var photos []models.Photo
	if err := fixture.s.DB.Find(&photos).Error; err != nil || len(photos) != 2 {
		tester.Fatalf("expected an album and a post photo: %v", err)
	}
	for _, photo := range photos {
		if photo.R2Key == "" || photo.S3Key != "" {
			tester.Fatal("expected an R2 photo")
		}
	}
	for _, endpoint := range []string{"/videos/create_presigned_url", "/photos/gcs/create_presigned_url"} {
		init := fixture.req("POST", endpoint, fixture.Token, map[string]any{"filename": "media.bin"}, 200)
		var upload models.Upload
		if err := fixture.s.DB.First(&upload, "id = ?", init["upload_id"]).Error; err != nil || upload.Provider != "r2" {
			tester.Fatalf("expected R2 multipart upload: %v", err)
		}
	}
	fixture.req("POST", "/videos/gcs/native/initiate", fixture.Token, map[string]any{"filename": "media.bin"}, 410)

	video := models.Video{UserID: &fixture.user, HlsS3: "r2/hls/01/output/index.m3u8", Thumbnail: "r2/hls/01/output/thumbnail.jpg"}
	if err := fixture.s.DB.Create(&video).Error; err != nil {
		tester.Fatal(err)
	}
	detail := fixture.req("GET", fmt.Sprintf("/videos/%d", video.ID), fixture.Token, nil, 200)
	response, err := http.Get(detail["hls_url"].(string))
	if err != nil {
		tester.Fatal(err)
	}
	defer response.Body.Close()
	manifest, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != 200 {
		tester.Fatalf("expected an R2 manifest: %d, %v", response.StatusCode, err)
	}
	lines := strings.Split(strings.TrimSpace(string(manifest)), "\n")
	segment, err := url.Parse(lines[len(lines)-1])
	if err != nil || segment.Path != "/private/hls/01/output/segment-00000.ts" || segment.Query().Get("X-Amz-Signature") == "" {
		tester.Fatal("expected a signed R2 segment URL")
	}
	if entries, err := os.ReadDir(fixture.s.Config.MediaDir); err != nil || len(entries) != 0 {
		tester.Fatalf("R2 uploads wrote to the media directory: %v", err)
	}
}

func multipartTestUpload(tester *testing.T, fixture *testFixture, endpoint, field, name string, data []byte, status int) map[string]any {
	tester.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile(field, name)
	if err != nil {
		tester.Fatal(err)
	}
	if _, err := file.Write(data); err != nil {
		tester.Fatal(err)
	}
	if err := form.Close(); err != nil {
		tester.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, fixture.http.URL+endpoint, &body)
	if err != nil {
		tester.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+fixture.Token)
	request.Header.Set("Content-Type", form.FormDataContentType())
	response, err := fixture.http.Client().Do(request)
	if err != nil {
		tester.Fatal(err)
	}
	defer response.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(response.Body).Decode(&out); err != nil || response.StatusCode != status {
		tester.Fatalf("%s returned %d %v (%v)", endpoint, response.StatusCode, out, err)
	}
	return out
}
