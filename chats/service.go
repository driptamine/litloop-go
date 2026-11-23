package chats

import (
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (handler *Handler) Member(chat, user int64) bool {
	var count int64
	Check(handler.DB.Model(&models.ChatParticipant{}).Where("chat_id = ? AND user_id = ?", chat, user).Count(&count).Error)
	return count > 0
}
func (handler *Handler) ChatAccess(req *Request, id int64) models.Chat {
	chat := Get[models.Chat](handler.DB, id)
	if req.User == nil || !handler.Member(id, req.User.ID) {
		Fail(403, "Chat membership required")
	}
	return chat
}

func (handler *Handler) UnreadMessages(userID int64) int64 {
	chats := handler.DB.Model(&models.Chat{}).
		Select("chats_chat.id").
		Joins("JOIN chats_chat_participants ON chats_chat_participants.chat_id = chats_chat.id").
		Where("chats_chat_participants.user_id = ? AND chats_chat.is_saved_messages = ?", userID, false)
	var count int64
	Check(handler.DB.Model(&models.Message{}).
		Where("chat_id IN (?) AND (user_id <> ? OR user_id IS NULL) AND is_read = ?", chats, userID, false).
		Count(&count).Error)
	return count
}

func (handler *Handler) SaveMessage(chat int64, user models.User, body map[string]any) models.Message {
	if !handler.Member(chat, user.ID) {
		Fail(403, "Chat membership required")
	}
	text := Str(body["text"])
	if len(text) > 100000 {
		Fail(400, "Message too long")
	}
	attachments, ok := body["attachments"].([]any)
	if !ok && body["attachments"] != nil {
		Fail(400, "attachments must be an array")
	}
	if attachments == nil {
		attachments = []any{}
	}
	handler.storeAttachmentKeys(attachments, user.ID)
	voiceID := int64(0)
	if body["voice_message_id"] != nil {
		voiceID = Num(body["voice_message_id"])
		Get[models.VoiceMessage](handler.DB.Where("user_id = ?", user.ID), voiceID)
	}
	if strings.TrimSpace(text) == "" && len(attachments) == 0 && voiceID == 0 {
		Fail(400, "Message is empty")
	}
	chatMessage := models.Message{ChatID: Ptr(chat), UserID: Ptr(user.ID), Text: text, Attachments: models.AsJSON(attachments)}
	var recipients []models.ChatParticipant
	Check(handler.DB.Transaction(func(tx *gorm.DB) error {
		// Keep the message ID cursor in commit order within this conversation.
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&models.Chat{}, chat).Error; err != nil {
			return err
		}
		Check(tx.Create(&chatMessage).Error)
		if voiceID > 0 {
			Check(tx.Create(&models.MessageVoice{MessageID: Ptr(chatMessage.ID), VoiceMessageID: Ptr(voiceID)}).Error)
		}
		recipients = All[models.ChatParticipant](tx.Where("chat_id = ? AND user_id <> ?", chat, user.ID))
		for _, recipient := range recipients {
			Check(tx.Create(&models.Notification{FromUserID: user.ID, ToUserID: recipient.UserID, TargetObjectID: Ptr(chatMessage.ID)}).Error)
		}
		return nil
	}))
	message := handler.MessageJSON(chatMessage)
	handler.Hub.Broadcast(fmt.Sprint(chat), map[string]any{"type": "new_message", "message": message})
	for _, recipient := range recipients {
		handler.Hub.Broadcast("user:"+fmt.Sprint(recipient.UserID), map[string]any{
			"type":	"new_message", "chat_id": chat, "chat_user_id": user.ID, "message_id": chatMessage.ID,
			"sender":	user.Username, "preview": Truncate(text, 50), "text": Truncate(text, 50),
			"created_at":	chatMessage.CreatedAt, "unread_count": handler.UnreadMessages(recipient.UserID), "message": message,
		})
	}
	handler.Hub.Broadcast("user:"+fmt.Sprint(user.ID), map[string]any{
		"type":	"chat_updated", "chat_id": chat, "message": message,
	})
	return chatMessage
}
