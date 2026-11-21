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
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestChatAttachments(t *testing.T) {
	f := setup(t)
	bob, bobID := f.second()
	chat := f.req("GET", fmt.Sprintf("/chats/direct/%d", bobID), f.Token, nil, 200)
	chatID := int64(chat["id"].(float64))
	messagePath := fmt.Sprintf("/chats/%d/messages", chatID)
	sendPath := fmt.Sprintf("/chats/%d/send", chatID)
	image := []byte("\x89PNG\r\n\x1a\nattachment")
	legacyKey := "attachments/" + strings.Repeat("abcd", 12) + ".png"
	legacyFile := filepath.Join(f.s.Config.MediaDir, legacyKey)
	if err := os.MkdirAll(filepath.Dir(legacyFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyFile, image, 0600); err != nil {
		t.Fatal(err)
	}
	expires := strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10)
	legacyURL := f.http.URL + "/media/" + legacyKey + "?expires=" + expires + "&signature=" + f.s.SignMedia(legacyKey, expires)
	tampered := legacyURL + "invalid"
	legacy := models.Message{
		ChatID: &chatID, UserID: &f.user, CreatedAt: time.Now().Add(-2 * time.Hour),
		Attachments: models.AsJSON([]any{map[string]any{"url": legacyURL}, map[string]any{"url": tampered}}),
	}
	if err := f.s.DB.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	history := f.req("GET", messagePath, bob, nil, 200)
	attachments := history["messages"].([]any)[0].(map[string]any)["attachments"].([]any)
	refreshed := attachments[0].(map[string]any)["url"].(string)
	if refreshed == legacyURL || attachments[1].(map[string]any)["url"] != tampered {
		t.Fatal("expected a new URL only for the valid stored signature")
	}
	assertAttachmentBody(t, refreshed, image)
	f.req("GET", strings.TrimPrefix(legacyURL, f.http.URL), "", nil, 403)
	resent := f.req("POST", sendPath, bob, map[string]any{"attachments": []any{map[string]any{"url": legacyURL}}}, 200)
	if resent["message"].(map[string]any)["attachments"].([]any)[0].(map[string]any)["url"] != legacyURL {
		t.Fatal("an already-expired URL in a new message was renewed")
	}

	put := make(chan string, 1)
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			if body, err := io.ReadAll(r.Body); err != nil || !bytes.Equal(body, image) {
				t.Error("R2 upload did not send the full file")
			}
			if r.Header.Get("Content-Type") != "image/png" || r.Header.Get("Authorization") == "" {
				t.Error("R2 upload did not send the image type and authorization")
			}
			put <- r.URL.Path
		case http.MethodGet:
			if r.URL.Query().Get("X-Amz-Signature") == "" {
				t.Error("missing R2 signature")
			}
			_, _ = w.Write(image)
		default:
			t.Errorf("unexpected R2 method %s", r.Method)
			w.WriteHeader(405)
		}
	}))
	defer remote.Close()
	f.s.Config.R2Endpoint, f.s.Config.R2Bucket = remote.URL, "private"
	f.s.Config.R2AccessKey, f.s.Config.R2SecretKey = "test-access", "test-secret"
	f.s.Config.R2PublicURL = "https://public.example"

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.Write(image); err != nil {
		t.Fatal(err)
	}
	if err = form.Close(); err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, f.http.URL+"/chats/upload/", &body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+f.Token)
	request.Header.Set("Content-Type", form.FormDataContentType())
	response, err := f.http.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var upload map[string]any
	if err = json.NewDecoder(response.Body).Decode(&upload); err != nil || response.StatusCode != 200 {
		t.Fatalf("upload status %d, body %v, error %v", response.StatusCode, upload, err)
	}
	var putPath string
	select {
	case putPath = <-put:
	default:
		t.Fatal("upload never reached R2")
	}
	if !strings.HasPrefix(putPath, fmt.Sprintf("/private/chat-attachments/%d/", f.user)) {
		t.Fatal(putPath)
	}
	preview, err := url.Parse(upload["url"].(string))
	if err != nil || preview.Path != putPath || preview.Query().Get("X-Amz-Signature") == "" {
		t.Fatal("upload did not return a private R2 preview")
	}
	created := f.req("POST", sendPath, f.Token, map[string]any{"attachments": []any{upload}}, 200)["message"].(map[string]any)
	var stored models.Message
	if err := f.s.DB.First(&stored, int64(created["id"].(float64))).Error; err != nil {
		t.Fatal(err)
	}
	if stored := string(stored.Attachments); strings.Contains(stored, `"url"`) || !strings.Contains(stored, strings.TrimPrefix(putPath, "/private/")) {
		t.Fatal("expected only the durable R2 key in the persisted attachment")
	}
	history = f.req("GET", fmt.Sprintf("%s?after=%.0f", messagePath, created["id"].(float64)-1), bob, nil, 200)
	attachment := history["messages"].([]any)[0].(map[string]any)["attachments"].([]any)[0].(map[string]any)
	assertAttachmentBody(t, attachment["url"].(string), image)
	forged := f.req("POST", sendPath, bob, map[string]any{"attachments": []any{map[string]any{"r2_key": strings.TrimPrefix(putPath, "/private/")}}}, 200)
	if attachment := forged["message"].(map[string]any)["attachments"].([]any)[0].(map[string]any); attachment["url"] != nil {
		t.Fatal("another uploader's private object was signed")
	}
}

func assertAttachmentBody(t *testing.T, url string, expected []byte) {
	t.Helper()
	response, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != 200 || !bytes.Equal(body, expected) {
		t.Fatalf("attachment response status %d: %v", response.StatusCode, err)
	}
}

func TestChatLibraryR2Attachments(t *testing.T) {
	f := setup(t)
	bob, bobID := f.second()
	chat := f.req("GET", fmt.Sprintf("/chats/direct/%d", bobID), f.Token, nil, 200)
	chatID := int64(chat["id"].(float64))
	messagesPath := fmt.Sprintf("/chats/%d/messages", chatID)
	sendPath := fmt.Sprintf("/chats/%d/send", chatID)
	f.s.Config.R2Endpoint, f.s.Config.R2Bucket = "https://r2.example", "private"
	f.s.Config.R2AccessKey, f.s.Config.R2SecretKey = "test-access", "test-secret"

	const photoKey = "photos/old.png-1795288952802"
	const videoKey = "videos/old.mp4"
	const trackKey = "tracks/old.mp3"
	for _, media := range []any{
		&models.Photo{UserID: &f.user, R2Key: photoKey},
		&models.Video{UserID: &f.user, R2Key: videoKey},
		&models.Track{UserID: &f.user, R2Key: trackKey},
	} {
		if err := f.s.DB.Create(media).Error; err != nil {
			t.Fatal(err)
		}
	}
	link := func(key string) string {
		return "https://r2.example/private/" + key + "?X-Amz-Date=20210101T000000Z&X-Amz-Signature=old"
	}
	legacyAttachments := []any{
		map[string]any{"url": link(photoKey), "type": "photo"},
		map[string]any{"url": link(videoKey), "type": "video"},
		map[string]any{"url": link(trackKey), "type": "track"},
		map[string]any{"url": strings.Replace(link(photoKey), "r2.example", "unrelated.example", 1), "type": "photo"},
	}
	for _, sender := range []int64{f.user, bobID} {
		message := models.Message{ChatID: &chatID, UserID: &sender, Attachments: models.AsJSON(legacyAttachments)}
		if err := f.s.DB.Create(&message).Error; err != nil {
			t.Fatal(err)
		}
	}
	newMessage := f.req("POST", sendPath, f.Token, map[string]any{"attachments": legacyAttachments}, 200)["message"].(map[string]any)
	var stored models.Message
	if err := f.s.DB.First(&stored, int64(newMessage["id"].(float64))).Error; err != nil {
		t.Fatal(err)
	}
	var persisted []map[string]any
	if err := json.Unmarshal(stored.Attachments, &persisted); err != nil {
		t.Fatal(err)
	}
	results := f.req("GET", messagesPath, bob, nil, 200)["messages"].([]any)
	for i, key := range []string{photoKey, videoKey, trackKey} {
		if persisted[i]["r2_key"] != key || persisted[i]["url"] != nil {
			t.Fatal("expected a stored R2 key without the temporary URL")
		}
		for _, index := range []int{0, 2} {
			attachment := results[index].(map[string]any)["attachments"].([]any)[i].(map[string]any)
			fresh, err := url.Parse(attachment["url"].(string))
			if err != nil || fresh.Path != "/private/"+key || fresh.Query().Get("X-Amz-Signature") == "" || fresh.Query().Get("X-Amz-Signature") == "old" {
				t.Fatal("expected a freshly signed catalog media URL")
			}
		}
		otherSender := results[1].(map[string]any)["attachments"].([]any)[i].(map[string]any)
		if otherSender["url"] != link(key) {
			t.Fatal("another sender's media link should not be renewed")
		}
	}
	external := results[0].(map[string]any)["attachments"].([]any)[3].(map[string]any)
	if external["url"] != legacyAttachments[3].(map[string]any)["url"] {
		t.Fatal("external URL changed")
	}
}
