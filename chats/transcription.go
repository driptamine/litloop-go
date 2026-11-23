package chats

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"
	"time"

	"gorm.io/gorm"
)

func (handler *Handler) Transcription(req *Request) any {
	voice := Get[models.VoiceMessage](handler.DB.Where("user_id = ?", req.User.ID), req.ID("voice"))
	if req.R.Method == "GET" {
		return handler.VoiceJSON(voice)
	}
	if handler.Config.TranscriptionURL == "" {
		Fail(503, "TRANSCRIPTION_URL is not configured")
	}
	if voice.TranscriptionStatus == "pending" || voice.TranscriptionStatus == "processing" {
		return handler.VoiceJSON(voice)
	}
	req.Tx(func(tx *gorm.DB) error {
		Check(tx.Model(&voice).Update("transcription_status", "pending").Error)
		return tx.Create(&models.Task{Kind: "transcribe", Payload: models.AsJSON(map[string]any{"voice_id": voice.ID, "url": handler.MediaURL(voice.R2Key, voice.GCSKey, voice.S3Key)}), Status: "pending", RunAt: time.Now()}).Error
	})
	req.Status = 202
	return map[string]any{"status": "pending", "voice_id": voice.ID}
}
func (handler *Handler) TranscriptionCallback(req *Request) any {
	if handler.Config.TranscriptionSecret == "" || !SecureEqual(strings.TrimPrefix(req.R.Header.Get("Authorization"), "Bearer "), handler.Config.TranscriptionSecret) {
		Fail(403, "Invalid callback token")
	}
	body := req.Body()
	voice := Get[models.VoiceMessage](handler.DB, Num(body["voice_id"]))
	status := Str(body["status"])
	if status != "completed" && status != "failed" && status != "processing" {
		Fail(400, "Invalid transcription status")
	}
	Check(handler.DB.Model(&voice).Updates(map[string]any{"transcription": Str(body["transcription"]), "transcription_language": Str(body["language"]), "transcription_status": status, "transcribed_at": time.Now()}).Error)
	return map[string]any{"status": "ok"}
}
