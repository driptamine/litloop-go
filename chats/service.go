package chats

import (
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) Member(chat, user int64) bool {
	var n int64
	Check(s.DB.Model(&models.ChatParticipant{}).Where("chat_id = ? AND user_id = ?", chat, user).Count(&n).Error)
	return n > 0
}
func (s *Handler) ChatAccess(q *Request, id int64) models.Chat {
	c := Get[models.Chat](s.DB, id)
	if q.User == nil || !s.Member(id, q.User.ID) {
		Fail(403, "Chat membership required")
	}
	return c
}
func (s *Handler) SaveMessage(chat int64, u models.User, b map[string]any) models.Message {
	if !s.Member(chat, u.ID) {
		Fail(403, "Chat membership required")
	}
	text := Str(b["text"])
	if len(text) > 100000 {
		Fail(400, "Message too long")
	}
	attachments := b["attachments"]
	if attachments == nil {
		attachments = []any{}
	}
	if _, ok := attachments.([]any); !ok {
		Fail(400, "attachments must be an array")
	}
	voiceID := int64(0)
	if b["voice_message_id"] != nil {
		voiceID = Num(b["voice_message_id"])
		Get[models.VoiceMessage](s.DB.Where("user_id = ?", u.ID), voiceID)
	}
	if strings.TrimSpace(text) == "" && len(attachments.([]any)) == 0 && voiceID == 0 {
		Fail(400, "Message is empty")
	}
	m := models.Message{ChatID: Ptr(chat), UserID: Ptr(u.ID), Text: text, Attachments: models.AsJSON(attachments)}
	Check(s.DB.Transaction(func(tx *gorm.DB) error {
		Check(tx.Create(&m).Error)
		if voiceID > 0 {
			Check(tx.Create(&models.MessageVoice{MessageID: Ptr(m.ID), VoiceMessageID: Ptr(voiceID)}).Error)
		}
		for _, p := range All[models.ChatParticipant](tx.Where("chat_id = ? AND user_id <> ?", chat, u.ID)) {
			Check(tx.Create(&models.Notification{FromUserID: u.ID, ToUserID: p.UserID, TargetObjectID: Ptr(m.ID)}).Error)
		}
		return nil
	}))
	message := s.MessageJSON(m)
	s.Hub.Broadcast(fmt.Sprint(chat), map[string]any{"type": "new_message", "message": message})
	for _, p := range All[models.ChatParticipant](s.DB.Where("chat_id = ? AND user_id <> ?", chat, u.ID)) {
		s.Hub.Broadcast("user:"+fmt.Sprint(p.UserID), map[string]any{"type": "new_message", "chat_id": chat, "message_id": m.ID, "sender": u.Username, "preview": Truncate(text, 50), "text": Truncate(text, 50), "created_at": m.CreatedAt})
	}
	return m
}
