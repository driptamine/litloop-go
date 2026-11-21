package chats

import (
	"fmt"
	. "litloop/backend/internal/httpapi"
	"log/slog"
)

func (s *Handler) presenceContacts(user int64) ([]int64, error) {
	var contacts []int64
	err := s.DB.Table("chats_chat_participants AS mine").
		Select("DISTINCT other.user_id").
		Joins("JOIN chats_chat_participants AS other ON other.chat_id = mine.chat_id AND other.user_id <> mine.user_id").
		Joins("JOIN chats_chat AS chat ON chat.id = mine.chat_id").
		Where("mine.user_id = ? AND chat.chat_type = ? AND chat.is_saved_messages = ?", user, "direct", false).
		Scan(&contacts).Error
	return contacts, err
}

func (s *Handler) updatePresence(user int64, session string, connected bool) {
	online, changed := s.Hub.SetPresence(user, session, connected)
	if !changed {
		return
	}
	contacts, err := s.presenceContacts(user)
	if err != nil {
		slog.Error("chat presence contacts", "error", err)
		return
	}
	for _, contact := range contacts {
		s.Hub.Broadcast(fmt.Sprint("user:", contact), map[string]any{
			"type": "presence_update", "user_id": user, "online": online,
		})
	}
}

func (s *Handler) sendPresence(p *Peer, user int64) {
	contacts, err := s.presenceContacts(user)
	if err != nil {
		slog.Error("chat presence snapshot", "error", err)
		return
	}
	event := map[string]any{"type": "presence_snapshot", "users": s.Hub.Presence(contacts)}
	select {
	case p.Send <- event:
	case <-p.Done:
	default:
		p.Close()
	}
}
