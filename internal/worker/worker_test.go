package worker

import (
	"context"
	"encoding/json"
	"gorm.io/gorm"
	"litloop/backend/internal/config"
	"litloop/backend/internal/models"
	"litloop/backend/internal/testutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestPersistentTaskAndTranscription(tester *testing.T) {
	appConfig := config.Config{SQLitePath: filepath.Join(tester.TempDir(), "worker.db"), JWTSecret: "test-secret", PublicURL: "http://localhost:8000", TranscriptionSecret: "shared"}
	db := testutil.Open(tester, &appConfig)
	var err error
	user := models.User{Email: "worker@example.com", Username: "worker", Password: "!", IsActive: true}
	if err = db.Create(&user).Error; err != nil {
		tester.Fatal(err)
	}
	voice := models.VoiceMessage{UserID: &user.ID, S3Key: "local/voice.wav", TranscriptionStatus: "pending"}
	if err = db.Create(&voice).Error; err != nil {
		tester.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, httpReq *http.Request) {
		if httpReq.Header.Get("Authorization") != "Bearer shared" {
			tester.Error("missing shared secret")
		}
		var request map[string]any
		if err := json.NewDecoder(httpReq.Body).Decode(&request); err != nil {
			tester.Error(err)
		}
		if request["voice_id"] != float64(voice.ID) || request["url"] == "" {
			tester.Error(request)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"text": "Hello world", "language": "en"})
	}))
	defer upstream.Close()
	appConfig.TranscriptionURL = upstream.URL
	task := models.Task{Kind: "transcribe", Payload: models.AsJSON(map[string]any{"voice_id": voice.ID}), Status: "pending", RunAt: time.Now().Add(-time.Minute)}
	if err = db.Create(&task).Error; err != nil {
		tester.Fatal(err)
	}
	worker := New(db, appConfig)
	claimed, err := worker.claim()
	if err != nil || claimed.ID != task.ID {
		tester.Fatal(err, claimed)
	}
	if _, err = worker.claim(); err != gorm.ErrRecordNotFound {
		tester.Fatal("task claimed twice", err)
	}
	if err = worker.Process(context.Background(), *claimed); err != nil {
		tester.Fatal(err)
	}
	if err = db.First(&voice, voice.ID).Error; err != nil {
		tester.Fatal(err)
	}
	if voice.Transcription != "Hello world" || voice.TranscriptionStatus != "completed" {
		tester.Fatal(voice)
	}
	old := time.Now().Add(-time.Hour)
	if err = db.Model(&task).Updates(map[string]any{"locked_at": old}).Error; err != nil {
		tester.Fatal(err)
	}
	reclaimed, err := worker.claim()
	if err != nil || reclaimed.ID != task.ID || reclaimed.Attempts != 2 {
		tester.Fatal(err, reclaimed)
	}
}
