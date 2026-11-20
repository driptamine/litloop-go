package chats

import (
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Handler) ChatList(q *Request) any {
	db := s.DB.Where("id IN (?)", s.DB.Model(&models.ChatParticipant{}).Select("chat_id").Where("user_id = ?", q.User.ID))
	if strings.HasSuffix(q.R.URL.Path, "/direct") {
		db = db.Where("chat_type = ? AND is_saved_messages = ?", "direct", false)
	} else if strings.HasSuffix(q.R.URL.Path, "/group") {
		db = db.Where("chat_type = ?", "groupchat")
	}
	out := []any{}
	for _, c := range All[models.Chat](db.Order("(SELECT MAX(created_at) FROM chats_message WHERE chat_id = chats_chat.id) DESC NULLS LAST")) {
		out = append(out, s.ChatJSON(c, q.User.ID, false))
	}
	return map[string]any{"chats": out}
}
func (s *Handler) DirectChat(q *Request) any {
	saved := strings.HasSuffix(q.R.URL.Path, "/saved")
	target := q.User.ID
	if !saved {
		target = q.ID("user")
		Get[models.User](s.DB, target)
		if target == q.User.ID {
			Fail(400, "You cannot chat with yourself")
		}
	}
	lo, hi := q.User.ID, target
	if lo > hi {
		lo, hi = hi, lo
	}
	name := fmt.Sprintf("dm-%d-%d", lo, hi)
	if saved {
		name = fmt.Sprintf("saved-%d", q.User.ID)
	}
	var c models.Chat
	q.Tx(func(tx *gorm.DB) error {
		var lock models.User
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lock, lo).Error)
		e := tx.First(&c, "name = ? AND chat_type = ?", name, "direct").Error
		if e == nil {
			return nil
		}
		if e != gorm.ErrRecordNotFound {
			return e
		}
		c = models.Chat{ChatType: "direct", Name: name, IsSavedMessages: saved}
		Check(tx.Create(&c).Error)
		Check(tx.Create(&models.ChatParticipant{ChatID: c.ID, UserID: q.User.ID}).Error)
		if !saved {
			return tx.Create(&models.ChatParticipant{ChatID: c.ID, UserID: target}).Error
		}
		return nil
	})
	return s.ChatJSON(c, q.User.ID, true)
}
func (s *Handler) CreateGroup(q *Request) any {
	b := q.Body()
	name := strings.TrimSpace(Str(b["name"]))
	if name == "" {
		Fail(400, "name is required")
	}
	c := models.Chat{ChatType: "groupchat", Name: Truncate(name, 20), Description: Str(b["description"]), ImageURL: Str(b["image_url"])}
	list := IDs(b["participant_ids"])
	q.Tx(func(tx *gorm.DB) error {
		Check(tx.Create(&c).Error)
		list = append(list, q.User.ID)
		for _, id := range list {
			Get[models.User](tx, id)
			Check(tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ChatParticipant{ChatID: c.ID, UserID: id}).Error)
		}
		return nil
	})
	q.Status = 201
	return map[string]any{"status": "created", "chat": s.ChatJSON(c, q.User.ID, false)}
}
func (s *Handler) GroupMember(q *Request) any {
	c := s.ChatAccess(q, q.ID("id"))
	if c.ChatType != "groupchat" {
		Fail(400, "Not a group chat")
	}
	id := Num(q.Body()["user_id"])
	u := Get[models.User](s.DB, id)
	status := "added"
	if strings.HasSuffix(q.R.URL.Path, "/remove") {
		Check(s.DB.Where("chat_id = ? AND user_id = ?", c.ID, id).Delete(&models.ChatParticipant{}).Error)
		status = "removed"
	} else {
		Check(s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.ChatParticipant{ChatID: c.ID, UserID: id}).Error)
	}
	return map[string]any{"status": status, "user_id": id, "username": u.Username}
}
func (s *Handler) ChatAction(q *Request) any {
	c := s.ChatAccess(q, q.ID("id"))
	if strings.HasSuffix(q.R.URL.Path, "/delete") {
		if c.IsSavedMessages {
			Fail(400, "Saved messages cannot be deleted")
		}
		if c.ChatType == "direct" {
			q.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "chats_chat", c.ID) })
		} else {
			Check(s.DB.Where("chat_id = ? AND user_id = ?", c.ID, q.User.ID).Delete(&models.ChatParticipant{}).Error)
		}
		return map[string]any{"status": "deleted", "id": c.ID}
	}
	var messages []models.Message
	Check(s.DB.Model(&messages).Clauses(clause.Returning{Columns: []clause.Column{{Name: "id"}}}).
		Where("chat_id = ? AND (user_id <> ? OR user_id IS NULL) AND is_read = ?", c.ID, q.User.ID, false).
		Update("is_read", true).Error)
	if len(messages) > 0 {
		ids := make([]int64, 0, len(messages))
		for _, m := range messages {
			ids = append(ids, m.ID)
		}
		s.Hub.Broadcast(fmt.Sprint(c.ID), map[string]any{"type": "read_receipt", "read_by": q.User.ID, "message_ids": ids})
	}
	s.Hub.Broadcast("user:"+fmt.Sprint(q.User.ID), map[string]any{"type": "unread_update", "unread_count": s.UnreadMessages(q.User.ID)})
	return map[string]any{"status": "success", "chat_id": c.ID}
}
func (s *Handler) SendMessage(q *Request) any {
	m := s.SaveMessage(q.ID("id"), *q.User, q.Body())
	return map[string]any{"status": "sent", "chat_id": q.ID("id"), "message": s.MessageJSON(m)}
}
