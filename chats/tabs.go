package chats

import (
	"encoding/json"
	"strings"

	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"

	"gorm.io/gorm"
)

func tabChatIDs(raw models.JSON) []int64 {
	ids := []int64{}
	if len(raw) != 0 {
		_ = json.Unmarshal(raw, &ids)
	}
	if ids == nil {
		return []int64{}
	}
	return ids
}

func (handler *Handler) ChatTabJSON(tab models.ChatTab) map[string]any {
	return map[string]any{"id": tab.ID, "emoji": tab.Emoji, "chat_ids": tabChatIDs(tab.ChatIDs)}
}

func (handler *Handler) ChatTabs(req *Request) any {
	tabs := All[models.ChatTab](handler.DB.Where("user_id = ?", req.User.ID).Order("position ASC, id ASC"))
	out := make([]any, 0, len(tabs))
	for _, tab := range tabs {
		out = append(out, handler.ChatTabJSON(tab))
	}
	return map[string]any{"tabs": out}
}

func (handler *Handler) ChatTabsWrite(req *Request) any {
	items, ok := req.Body()["tabs"].([]any)
	if !ok {
		Fail(400, "tabs is required")
	}
	if len(items) > 20 {
		Fail(400, "Too many tabs")
	}
	tabs := make([]models.ChatTab, 0, len(items))
	for _, raw := range items {
		entry, ok := raw.(map[string]any)
		if !ok {
			Fail(400, "Invalid tab")
		}
		emoji := strings.TrimSpace(Str(entry["emoji"]))
		if emoji == "" || len([]rune(emoji)) > 8 {
			Fail(400, "Tab emoji is required")
		}
		ids := []int64{}
		if list, ok := entry["chat_ids"].([]any); ok {
			seen := map[int64]bool{}
			for _, value := range list {
				if value == nil {
					continue
				}
				id := Num(value)
				if id <= 0 || seen[id] || len(ids) >= 1000 {
					continue
				}
				seen[id] = true
				ids = append(ids, id)
			}
		}
		if ids == nil {
			ids = []int64{}
		}
		tabs = append(tabs, models.ChatTab{Emoji: emoji, ChatIDs: models.AsJSON(ids)})
	}
	req.Tx(func(tx *gorm.DB) error {
		Check(tx.Where("user_id = ?", req.User.ID).Delete(&models.ChatTab{}).Error)
		for index := range tabs {
			tabs[index].UserID = req.User.ID
			tabs[index].Position = index
			Check(tx.Create(&tabs[index]).Error)
		}
		return nil
	})
	out := make([]any, 0, len(tabs))
	for _, tab := range tabs {
		out = append(out, handler.ChatTabJSON(tab))
	}
	return map[string]any{"tabs": out}
}
