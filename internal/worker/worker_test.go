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

func TestPersistentTaskAndTranscription(t *testing.T) {
	c := config.Config{SQLitePath: filepath.Join(t.TempDir(), "worker.db"), JWTSecret: "test-secret", PublicURL: "http://localhost:8000", TranscriptionSecret: "shared"}
	db := testutil.Open(t, &c)
	var e error
	u := models.User{Email: "worker@example.com", Username: "worker", Password: "!", IsActive: true}
	if e = db.Create(&u).Error; e != nil {
		t.Fatal(e)
	}
	voice := models.VoiceMessage{UserID: &u.ID, S3Key: "local/voice.wav", TranscriptionStatus: "pending"}
	if e = db.Create(&voice).Error; e != nil {
		t.Fatal(e)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer shared" {
			t.Error("missing shared secret")
		}
		var request map[string]any
		if e := json.NewDecoder(r.Body).Decode(&request); e != nil {
			t.Error(e)
		}
		if request["voice_id"] != float64(voice.ID) || request["url"] == "" {
			t.Error(request)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"text": "Hello world", "language": "en"})
	}))
	defer upstream.Close()
	c.TranscriptionURL = upstream.URL
	task := models.Task{Kind: "transcribe", Payload: models.AsJSON(map[string]any{"voice_id": voice.ID}), Status: "pending", RunAt: time.Now().Add(-time.Minute)}
	if e = db.Create(&task).Error; e != nil {
		t.Fatal(e)
	}
	worker := New(db, c)
	claimed, e := worker.claim()
	if e != nil || claimed.ID != task.ID {
		t.Fatal(e, claimed)
	}
	if _, e = worker.claim(); e != gorm.ErrRecordNotFound {
		t.Fatal("task claimed twice", e)
	}
	if e = worker.Process(context.Background(), *claimed); e != nil {
		t.Fatal(e)
	}
	if e = db.First(&voice, voice.ID).Error; e != nil {
		t.Fatal(e)
	}
	if voice.Transcription != "Hello world" || voice.TranscriptionStatus != "completed" {
		t.Fatal(voice)
	}
	old := time.Now().Add(-time.Hour)
	if e = db.Model(&task).Updates(map[string]any{"locked_at": old}).Error; e != nil {
		t.Fatal(e)
	}
	reclaimed, e := worker.claim()
	if e != nil || reclaimed.ID != task.ID || reclaimed.Attempts != 2 {
		t.Fatal(e, reclaimed)
	}
}
