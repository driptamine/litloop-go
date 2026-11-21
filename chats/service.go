package chats

import (
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (s *Handler) UnreadMessages(userID int64) int64 {
	chats := s.DB.Model(&models.Chat{}).
		Select("chats_chat.id").
		Joins("JOIN chats_chat_participants ON chats_chat_participants.chat_id = chats_chat.id").
		Where("chats_chat_participants.user_id = ? AND chats_chat.is_saved_messages = ?", userID, false)
	var count int64
	Check(s.DB.Model(&models.Message{}).
		Where("chat_id IN (?) AND (user_id <> ? OR user_id IS NULL) AND is_read = ?", chats, userID, false).
		Count(&count).Error)
	return count
}

func (s *Handler) SaveMessage(chat int64, u models.User, b map[string]any) models.Message {
	if !s.Member(chat, u.ID) {
		Fail(403, "Chat membership required")
	}
	text := Str(b["text"])
	if len(text) > 100000 {
		Fail(400, "Message too long")
	}
	attachments, ok := b["attachments"].([]any)
	if !ok && b["attachments"] != nil {
		Fail(400, "attachments must be an array")
	}
	if attachments == nil {
		attachments = []any{}
	}
	s.storeAttachmentKeys(attachments, u.ID)
	voiceID := int64(0)
	if b["voice_message_id"] != nil {
		voiceID = Num(b["voice_message_id"])
		Get[models.VoiceMessage](s.DB.Where("user_id = ?", u.ID), voiceID)
	}
	if strings.TrimSpace(text) == "" && len(attachments) == 0 && voiceID == 0 {
		Fail(400, "Message is empty")
	}
	m := models.Message{ChatID: Ptr(chat), UserID: Ptr(u.ID), Text: text, Attachments: models.AsJSON(attachments)}
	var recipients []models.ChatParticipant
	Check(s.DB.Transaction(func(tx *gorm.DB) error {
		// Keep the message ID cursor in commit order within this conversation.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&models.Chat{}, chat).Error; err != nil {
			return err
		}
		Check(tx.Create(&m).Error)
		if voiceID > 0 {
			Check(tx.Create(&models.MessageVoice{MessageID: Ptr(m.ID), VoiceMessageID: Ptr(voiceID)}).Error)
		}
		recipients = All[models.ChatParticipant](tx.Where("chat_id = ? AND user_id <> ?", chat, u.ID))
		for _, p := range recipients {
			Check(tx.Create(&models.Notification{FromUserID: u.ID, ToUserID: p.UserID, TargetObjectID: Ptr(m.ID)}).Error)
		}
		return nil
	}))
	message := s.MessageJSON(m)
	s.Hub.Broadcast(fmt.Sprint(chat), map[string]any{"type": "new_message", "message": message})
	for _, p := range recipients {
		s.Hub.Broadcast("user:"+fmt.Sprint(p.UserID), map[string]any{
			"type": "new_message", "chat_id": chat, "chat_user_id": u.ID, "message_id": m.ID,
			"sender": u.Username, "preview": Truncate(text, 50), "text": Truncate(text, 50),
			"created_at": m.CreatedAt, "unread_count": s.UnreadMessages(p.UserID), "message": message,
		})
	}
	s.Hub.Broadcast("user:"+fmt.Sprint(u.ID), map[string]any{
		"type": "chat_updated", "chat_id": chat, "message": message,
	})
	return m
}
