package chats

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) VoiceJSON(v models.VoiceMessage) any {
	o := Object(v)
	o["url"] = s.MediaURL(v.R2Key, v.GCSKey, v.S3Key)
	return o
}
func (s *Handler) MessageJSON(m models.Message) map[string]any {
	o := Object(m)
	o["username"] = "AI"
	o["avatar"] = nil
	o["voice_message"] = nil
	if m.UserID != nil {
		var u models.User
		if s.DB.First(&u, *m.UserID).Error == nil {
			o["username"] = u.Username
			o["avatar"] = u.Avatar
		}
	}
	var mv models.MessageVoice
	if s.DB.First(&mv, "message_id = ?", m.ID).Error == nil && mv.VoiceMessageID != nil {
		var voice models.VoiceMessage
		if s.DB.First(&voice, *mv.VoiceMessageID).Error == nil {
			o["voice_message"] = s.VoiceJSON(voice)
		}
	}
	return o
}
func (s *Handler) ChatJSON(c models.Chat, user int64, detail bool) map[string]any {
	o := Object(c)
	users := All[models.User](s.DB.Where("id IN (?)", s.DB.Model(&models.ChatParticipant{}).Select("user_id").Where("chat_id = ?", c.ID)))
	participants := []any{}
	o["other_participant"] = nil
	for _, u := range users {
		p := map[string]any{"id": u.ID, "username": u.Username, "avatar": u.Avatar}
		participants = append(participants, p)
		if u.ID != user {
			o["other_participant"] = p
			if c.ChatType == "direct" {
				o["username"] = u.Username
				o["avatar"] = u.Avatar
				o["target_user"] = u.Username
			}
		}
	}
	o["participants"] = participants
	o["participant_count"] = len(participants)
	var unread int64
	Check(s.DB.Model(&models.Message{}).Where("chat_id = ? AND (user_id <> ? OR user_id IS NULL) AND is_read = ?", c.ID, user, false).Count(&unread).Error)
	o["unread_count"] = unread
	var last models.Message
	if s.DB.Where("chat_id = ?", c.ID).Order("id DESC").First(&last).Error == nil {
		message := s.MessageJSON(last)
		message["user"] = message["username"]
		o["last_message"] = message
	}
	if detail {
		messages := []any{}
		for _, m := range All[models.Message](s.DB.Where("chat_id = ?", c.ID).Order("id DESC").Limit(200)) {
			messages = append(messages, s.MessageJSON(m))
		}
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
		o["messages"] = messages
	}
	return o
}
