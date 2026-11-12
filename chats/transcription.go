package chats

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"
	"time"

	"gorm.io/gorm"
)

func (s *Handler) Transcription(q *Request) any {
	v := Get[models.VoiceMessage](s.DB.Where("user_id = ?", q.User.ID), q.ID("voice"))
	if q.R.Method == "GET" {
		return s.VoiceJSON(v)
	}
	if s.Config.TranscriptionURL == "" {
		Fail(503, "TRANSCRIPTION_URL is not configured")
	}
	if v.TranscriptionStatus == "pending" || v.TranscriptionStatus == "processing" {
		return s.VoiceJSON(v)
	}
	q.Tx(func(tx *gorm.DB) error {
		Check(tx.Model(&v).Update("transcription_status", "pending").Error)
		return tx.Create(&models.Task{Kind: "transcribe", Payload: models.AsJSON(map[string]any{"voice_id": v.ID, "url": s.MediaURL(v.R2Key, v.GCSKey, v.S3Key)}), Status: "pending", RunAt: time.Now()}).Error
	})
	q.Status = 202
	return map[string]any{"status": "pending", "voice_id": v.ID}
}
func (s *Handler) TranscriptionCallback(q *Request) any {
	if s.Config.TranscriptionSecret == "" || !SecureEqual(strings.TrimPrefix(q.R.Header.Get("Authorization"), "Bearer "), s.Config.TranscriptionSecret) {
		Fail(403, "Invalid callback token")
	}
	b := q.Body()
	v := Get[models.VoiceMessage](s.DB, Num(b["voice_id"]))
	status := Str(b["status"])
	if status != "completed" && status != "failed" && status != "processing" {
		Fail(400, "Invalid transcription status")
	}
	Check(s.DB.Model(&v).Updates(map[string]any{"transcription": Str(b["transcription"]), "transcription_language": Str(b["language"]), "transcription_status": status, "transcribed_at": time.Now()}).Error)
	return map[string]any{"status": "ok"}
}
