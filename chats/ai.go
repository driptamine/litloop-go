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

func (handler *Handler) Gemini(req *Request) any {
	if handler.Config.GeminiKey == "" {
		Fail(503, "GEMINI_API_KEY is not configured")
	}
	text := req.R.URL.Query().Get("message")
	if text == "" {
		text = req.R.URL.Query().Get("q")
	}
	if text == "" {
		text = req.R.URL.Query().Get("text")
	}
	if req.R.Method == "POST" {
		text = Str(req.Body()["message"])
	}
	if text == "" {
		Fail(400, "message is required")
	}
	payload, _ := json.Marshal(map[string]any{"contents": []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": text}}}}})
	data := handler.FetchJSON(req, "POST", "https://generativelanguage.googleapis.com/v1beta/models/"+url.PathEscape(handler.Config.GeminiModel)+":generateContent", payload, map[string]string{"Content-Type": "application/json", "x-goog-api-key": handler.Config.GeminiKey}).(map[string]any)
	answer := ""
	if candidates, ok := data["candidates"].([]any); ok {
		for _, candidate := range candidates {
			candidateMap, _ := candidate.(map[string]any)
			content, _ := candidateMap["content"].(map[string]any)
			parts, _ := content["parts"].([]any)
			for _, part := range parts {
				partMap, _ := part.(map[string]any)
				answer += Str(partMap["text"])
			}
		}
	}
	var chat models.Chat
	req.Tx(func(tx *gorm.DB) error {
		var lock models.User
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lock, req.User.ID).Error)
		name := fmt.Sprintf("ai-%d", req.User.ID)
		err := tx.First(&chat, "name = ? AND chat_type = ?", name, "direct").Error
		if err == gorm.ErrRecordNotFound {
			chat = models.Chat{Name: name, ChatType: "direct", Description: "AI Assistant Conversation"}
			Check(tx.Create(&chat).Error)
			Check(tx.Create(&models.ChatParticipant{ChatID: chat.ID, UserID: req.User.ID}).Error)
		} else {
			Check(err)
		}
		Check(tx.Create(&models.Message{ChatID: Ptr(chat.ID), UserID: Ptr(req.User.ID), Text: text}).Error)
		return tx.Create(&models.Message{ChatID: Ptr(chat.ID), Text: answer}).Error
	})
	return map[string]any{"response": answer, "chat_id": chat.ID}
}
