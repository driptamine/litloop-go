package chats

import (
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (handler *Handler) ChatList(req *Request) any {
	db := handler.DB.Where("id IN (?)", handler.DB.Model(&models.ChatParticipant{}).Select("chat_id").Where("user_id = ?", req.User.ID))
	if strings.HasSuffix(req.R.URL.Path, "/direct") {
		db = db.Where("chat_type = ? AND is_saved_messages = ?", "direct", false)
	} else if strings.HasSuffix(req.R.URL.Path, "/group") {
		db = db.Where("chat_type = ?", "groupchat")
	}
	chats := All[models.Chat](db.Order("(SELECT MAX(created_at) FROM chats_message WHERE chat_id = chats_chat.id) DESC NULLS LAST"))
	preloads := handler.loadChatPreload(chats, req.User.ID, false)
	out := []any{}
	for _, chat := range chats {
		out = append(out, handler.chatJSONWith(chat, req.User.ID, false, preloads[chat.ID]))
	}
	return map[string]any{"chats": out}
}

func (handler *Handler) MessagesAfter(req *Request) any {
	chat := handler.ChatAccess(req, req.ID("id"))
	var after int64
	if value := req.R.URL.Query().Get("after"); value != "" {
		after = Num(value)
	}
	const pageSize = 200
	rows := All[models.Message](handler.DB.Where("chat_id = ? AND id > ?", chat.ID, after).Order("id ASC").Limit(pageSize + 1))
	hasMore := len(rows) > pageSize
	if hasMore {
		rows = rows[:pageSize]
	}
	return map[string]any{"messages": handler.messagesJSON(rows), "has_more": hasMore}
}
func (handler *Handler) DirectChat(req *Request) any {
	saved := strings.HasSuffix(req.R.URL.Path, "/saved")
	target := req.User.ID
	if !saved {
		target = req.ID("user")
		Get[models.User](handler.DB, target)
		if target == req.User.ID {
			Fail(400, "You cannot chat with yourself")
		}
	}
	lo, hi := req.User.ID, target
	if lo > hi {
		lo, hi = hi, lo
	}
	name := fmt.Sprintf("dm-%d-%d", lo, hi)
	if saved {
		name = fmt.Sprintf("saved-%d", req.User.ID)
	}
	var chat models.Chat
	req.Tx(func(tx *gorm.DB) error {
		var lock models.User
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lock, lo).Error)
		err := tx.First(&chat, "name = ? AND chat_type = ?", name, "direct").Error
		if err == nil {
			return nil
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		chat = models.Chat{ChatType: "direct", Name: name, IsSavedMessages: saved}
		Check(tx.Create(&chat).Error)
		Check(tx.Create(&models.ChatParticipant{ChatID: chat.ID, UserID: req.User.ID}).Error)
		if !saved {
			return tx.Create(&models.ChatParticipant{ChatID: chat.ID, UserID: target}).Error
		}
		return nil
	})
	return handler.ChatJSON(chat, req.User.ID, true)
}
func (handler *Handler) CreateGroup(req *Request) any {
	body := req.Body()
	name := strings.TrimSpace(Str(body["name"]))
	if name == "" {
		Fail(400, "name is required")
	}
	chat := models.Chat{ChatType: "groupchat", Name: Truncate(name, 20), Description: Str(body["description"]), ImageURL: Str(body["image_url"])}
	list := IDs(body["participant_ids"])
	req.Tx(func(tx *gorm.DB) error {
		Check(tx.Create(&chat).Error)
		list = append(list, req.User.ID)
		for _, id := range list {
			Get[models.User](tx, id)
			Check(tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ChatParticipant{ChatID: chat.ID, UserID: id}).Error)
		}
		return nil
	})
	req.Status = 201
	return map[string]any{"status": "created", "chat": handler.ChatJSON(chat, req.User.ID, false)}
}
func (handler *Handler) GroupMember(req *Request) any {
	chat := handler.ChatAccess(req, req.ID("id"))
	if chat.ChatType != "groupchat" {
		Fail(400, "Not a group chat")
	}
	id := Num(req.Body()["user_id"])
	user := Get[models.User](handler.DB, id)
	status := "added"
	if strings.HasSuffix(req.R.URL.Path, "/remove") {
		Check(handler.DB.Where("chat_id = ? AND user_id = ?", chat.ID, id).Delete(&models.ChatParticipant{}).Error)
		status = "removed"
	} else {
		Check(handler.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ChatParticipant{ChatID: chat.ID, UserID: id}).Error)
	}
	return map[string]any{"status": status, "user_id": id, "username": user.Username}
}
func (handler *Handler) ChatAction(req *Request) any {
	chat := handler.ChatAccess(req, req.ID("id"))
	if strings.HasSuffix(req.R.URL.Path, "/delete") {
		if chat.IsSavedMessages {
			Fail(400, "Saved messages cannot be deleted")
		}
		if chat.ChatType == "direct" {
			req.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "chats_chat", chat.ID) })
		} else {
			Check(handler.DB.Where("chat_id = ? AND user_id = ?", chat.ID, req.User.ID).Delete(&models.ChatParticipant{}).Error)
		}
		return map[string]any{"status": "deleted", "id": chat.ID}
	}
	var messages []models.Message
	Check(handler.DB.Model(&messages).Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).
		Where("chat_id = ? AND (user_id <> ? OR user_id IS NULL) AND is_read = ?", chat.ID, req.User.ID, false).
		Update("is_read", true).Error)
	if len(messages) > 0 {
		ids := make([]int64, 0, len(messages))
		for _, messageID := range messages {
			ids = append(ids, messageID.ID)
		}
		handler.Hub.Broadcast(fmt.Sprint(chat.ID), map[string]any{"type": "read_receipt", "read_by": req.User.ID, "message_ids": ids})
	}
	handler.Hub.Broadcast("user:"+fmt.Sprint(req.User.ID), map[string]any{
		"type":	"unread_update", "chat_id": chat.ID, "unread_count": handler.UnreadMessages(req.User.ID),
	})
	return map[string]any{"status": "success", "chat_id": chat.ID}
}
func (handler *Handler) SendMessage(req *Request) any {
	message := handler.SaveMessage(req.ID("id"), *req.User, req.Body())
	return map[string]any{"status": "sent", "chat_id": req.ID("id"), "message": handler.MessageJSON(message)}
}
