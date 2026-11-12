package chats

import (
	"encoding/json"
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"net/url"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Handler) Gemini(q *Request) any {
	if s.Config.GeminiKey == "" {
		Fail(503, "GEMINI_API_KEY is not configured")
	}
	text := q.R.URL.Query().Get("message")
	if text == "" {
		text = q.R.URL.Query().Get("q")
	}
	if text == "" {
		text = q.R.URL.Query().Get("text")
	}
	if q.R.Method == "POST" {
		text = Str(q.Body()["message"])
	}
	if text == "" {
		Fail(400, "message is required")
	}
	payload, _ := json.Marshal(map[string]any{"contents": []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": text}}}}})
	data := s.FetchJSON(q, "POST", "https://generativelanguage.googleapis.com/v1beta/models/"+url.PathEscape(s.Config.GeminiModel)+":generateContent", payload, map[string]string{"Content-Type": "application/json", "x-goog-api-key": s.Config.GeminiKey}).(map[string]any)
	answer := ""
	if candidates, ok := data["candidates"].([]any); ok {
		for _, candidate := range candidates {
			c, _ := candidate.(map[string]any)
			content, _ := c["content"].(map[string]any)
			parts, _ := content["parts"].([]any)
			for _, part := range parts {
				p, _ := part.(map[string]any)
				answer += Str(p["text"])
			}
		}
	}
	var chat models.Chat
	q.Tx(func(tx *gorm.DB) error {
		var lock models.User
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lock, q.User.ID).Error)
		name := fmt.Sprintf("ai-%d", q.User.ID)
		e := tx.First(&chat, "name = ? AND chat_type = ?", name, "direct").Error
		if e == gorm.ErrRecordNotFound {
			chat = models.Chat{Name: name, ChatType: "direct", Description: "AI Assistant Conversation"}
			Check(tx.Create(&chat).Error)
			Check(tx.Create(&models.ChatParticipant{ChatID: chat.ID, UserID: q.User.ID}).Error)
		} else {
			Check(e)
		}
		Check(tx.Create(&models.Message{ChatID: Ptr(chat.ID), UserID: Ptr(q.User.ID), Text: text}).Error)
		return tx.Create(&models.Message{ChatID: Ptr(chat.ID), Text: answer}).Error
	})
	return map[string]any{"response": answer, "chat_id": chat.ID}
}
