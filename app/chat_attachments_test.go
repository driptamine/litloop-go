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

func TestChatAttachments(tester *testing.T) {
	fixture := setup(tester)
	bob, bobID := fixture.second()
	chat := fixture.req("GET", fmt.Sprintf("/chats/direct/%d", bobID), fixture.Token, nil, 200)
	chatID := int64(chat["id"].(float64))
	messagePath := fmt.Sprintf("/chats/%d/messages", chatID)
	sendPath := fmt.Sprintf("/chats/%d/send", chatID)
	image := []byte("\x89PNG\r\n\x1a\nattachment")
	legacyKey := "attachments/" + strings.Repeat("abcd", 12) + ".png"
	legacyFile := filepath.Join(fixture.s.Config.MediaDir, legacyKey)
	if err := os.MkdirAll(filepath.Dir(legacyFile), 0755); err != nil {
		tester.Fatal(err)
	}
	if err := os.WriteFile(legacyFile, image, 0600); err != nil {
		tester.Fatal(err)
	}
	expires := strconv.FormatInt(time.Now().Add(-time.Hour).Unix(), 10)
	legacyURL := fixture.http.URL + "/media/" + legacyKey + "?expires=" + expires + "&signature=" + fixture.s.SignMedia(legacyKey, expires)
	tampered := legacyURL + "invalid"
	legacy := models.Message{
		ChatID:	&chatID, UserID: &fixture.user, CreatedAt: time.Now().Add(-2 * time.Hour),
		Attachments:	models.AsJSON([]any{map[string]any{"url": legacyURL}, map[string]any{"url": tampered}}),
	}
	if err := fixture.s.DB.Create(&legacy).Error; err != nil {
		tester.Fatal(err)
	}
	history := fixture.req("GET", messagePath, bob, nil, 200)
	attachments := history["messages"].([]any)[0].(map[string]any)["attachments"].([]any)
	refreshed := attachments[0].(map[string]any)["url"].(string)
	if refreshed == legacyURL || attachments[1].(map[string]any)["url"] != tampered {
		tester.Fatal("expected a new URL only for the valid stored signature")
	}
	assertAttachmentBody(tester, refreshed, image)
	fixture.req("GET", strings.TrimPrefix(legacyURL, fixture.http.URL), "", nil, 403)
	resent := fixture.req("POST", sendPath, bob, map[string]any{"attachments": []any{map[string]any{"url": legacyURL}}}, 200)
	if resent["message"].(map[string]any)["attachments"].([]any)[0].(map[string]any)["url"] != legacyURL {
		tester.Fatal("an already-expired URL in a new message was renewed")
	}

	put := make(chan string, 1)
	remote := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, httpReq *http.Request) {
		switch httpReq.Method {
		case http.MethodPut:
			if body, err := io.ReadAll(httpReq.Body); err != nil || !bytes.Equal(body, image) {
				tester.Error("R2 upload did not send the full file")
			}
			if httpReq.Header.Get("Content-Type") != "image/png" || httpReq.Header.Get("Authorization") == "" {
				tester.Error("R2 upload did not send the image type and authorization")
			}
			put <- httpReq.URL.Path
		case http.MethodGet:
			if httpReq.URL.Query().Get("X-Amz-Signature") == "" {
				tester.Error("missing R2 signature")
			}
			_, _ = writer.Write(image)
		default:
			tester.Errorf("unexpected R2 method %s", httpReq.Method)
			writer.WriteHeader(405)
		}
	}))
	defer remote.Close()
	fixture.s.Config.R2Endpoint, fixture.s.Config.R2Bucket = remote.URL, "private"
	fixture.s.Config.R2AccessKey, fixture.s.Config.R2SecretKey = "test-access", "test-secret"
	fixture.s.Config.R2PublicURL = "https://public.example"

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	file, err := form.CreateFormFile("file", "photo.png")
	if err != nil {
		tester.Fatal(err)
	}
	if _, err = file.Write(image); err != nil {
		tester.Fatal(err)
	}
	if err = form.Close(); err != nil {
		tester.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPost, fixture.http.URL+"/chats/upload/", &body)
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
	var upload map[string]any
	if err = json.NewDecoder(response.Body).Decode(&upload); err != nil || response.StatusCode != 200 {
		tester.Fatalf("upload status %d, body %v, error %v", response.StatusCode, upload, err)
	}
	var putPath string
	select {
	case putPath = <-put:
	default:
		tester.Fatal("upload never reached R2")
	}
	if !strings.HasPrefix(putPath, fmt.Sprintf("/private/chat-attachments/%d/", fixture.user)) {
		tester.Fatal(putPath)
	}
	preview, err := url.Parse(upload["url"].(string))
	if err != nil || preview.Path != putPath || preview.Query().Get("X-Amz-Signature") == "" {
		tester.Fatal("upload did not return a private R2 preview")
	}
	created := fixture.req("POST", sendPath, fixture.Token, map[string]any{"attachments": []any{upload}}, 200)["message"].(map[string]any)
	var stored models.Message
	if err := fixture.s.DB.First(&stored, int64(created["id"].(float64))).Error; err != nil {
		tester.Fatal(err)
	}
	if stored := string(stored.Attachments); strings.Contains(stored, `"url"`) || !strings.Contains(stored, strings.TrimPrefix(putPath, "/private/")) {
		tester.Fatal("expected only the durable R2 key in the persisted attachment")
	}
	history = fixture.req("GET", fmt.Sprintf("%s?after=%.0f", messagePath, created["id"].(float64)-1), bob, nil, 200)
	attachment := history["messages"].([]any)[0].(map[string]any)["attachments"].([]any)[0].(map[string]any)
	assertAttachmentBody(tester, attachment["url"].(string), image)
	forged := fixture.req("POST", sendPath, bob, map[string]any{"attachments": []any{map[string]any{"r2_key": strings.TrimPrefix(putPath, "/private/")}}}, 200)
	if attachment := forged["message"].(map[string]any)["attachments"].([]any)[0].(map[string]any); attachment["url"] != nil {
		tester.Fatal("another uploader's private object was signed")
	}
}

func assertAttachmentBody(tester *testing.T, url string, expected []byte) {
	tester.Helper()
	response, err := http.Get(url)
	if err != nil {
		tester.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != 200 || !bytes.Equal(body, expected) {
		tester.Fatalf("attachment response status %d: %v", response.StatusCode, err)
	}
}

func TestChatLibraryR2Attachments(tester *testing.T) {
	fixture := setup(tester)
	bob, bobID := fixture.second()
	chat := fixture.req("GET", fmt.Sprintf("/chats/direct/%d", bobID), fixture.Token, nil, 200)
	chatID := int64(chat["id"].(float64))
	messagesPath := fmt.Sprintf("/chats/%d/messages", chatID)
	sendPath := fmt.Sprintf("/chats/%d/send", chatID)
	fixture.s.Config.R2Endpoint, fixture.s.Config.R2Bucket = "https://r2.example", "private"
	fixture.s.Config.R2AccessKey, fixture.s.Config.R2SecretKey = "test-access", "test-secret"

	const photoKey = "photos/old.png-1795288952802"
	const videoKey = "videos/old.mp4"
	const trackKey = "tracks/old.mp3"
	for _, media := range []any{
		&models.Photo{UserID: &fixture.user, R2Key: photoKey},
		&models.Video{UserID: &fixture.user, R2Key: videoKey},
		&models.Track{UserID: &fixture.user, R2Key: trackKey},
	} {
		if err := fixture.s.DB.Create(media).Error; err != nil {
			tester.Fatal(err)
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
	for _, sender := range []int64{fixture.user, bobID} {
		message := models.Message{ChatID: &chatID, UserID: &sender, Attachments: models.AsJSON(legacyAttachments)}
		if err := fixture.s.DB.Create(&message).Error; err != nil {
			tester.Fatal(err)
		}
	}
	newMessage := fixture.req("POST", sendPath, fixture.Token, map[string]any{"attachments": legacyAttachments}, 200)["message"].(map[string]any)
	var stored models.Message
	if err := fixture.s.DB.First(&stored, int64(newMessage["id"].(float64))).Error; err != nil {
		tester.Fatal(err)
	}
	var persisted []map[string]any
	if err := json.Unmarshal(stored.Attachments, &persisted); err != nil {
		tester.Fatal(err)
	}
	results := fixture.req("GET", messagesPath, bob, nil, 200)["messages"].([]any)
	for index, key := range []string{photoKey, videoKey, trackKey} {
		if persisted[index]["r2_key"] != key || persisted[index]["url"] != nil {
			tester.Fatal("expected a stored R2 key without the temporary URL")
		}
		for _, position := range []int{0, 2} {
			attachment := results[position].(map[string]any)["attachments"].([]any)[index].(map[string]any)
			fresh, err := url.Parse(attachment["url"].(string))
			if err != nil || fresh.Path != "/private/"+key || fresh.Query().Get("X-Amz-Signature") == "" || fresh.Query().Get("X-Amz-Signature") == "old" {
				tester.Fatal("expected a freshly signed catalog media URL")
			}
		}
		otherSender := results[1].(map[string]any)["attachments"].([]any)[index].(map[string]any)
		if otherSender["url"] != link(key) {
			tester.Fatal("another sender's media link should not be renewed")
		}
	}
	external := results[0].(map[string]any)["attachments"].([]any)[3].(map[string]any)
	if external["url"] != legacyAttachments[3].(map[string]any)["url"] {
		tester.Fatal("external URL changed")
	}
}
