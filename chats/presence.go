package chats

import (
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"log/slog"
	"net/http"
	"regexp"
	"time"
)

const httpPresenceLease = 35 * time.Second

var sessionIDPattern = regexp.MustCompile(`^[a-zA-Z0-9-]{16,80}$`)

func lastSeen(user models.User) time.Time {
	at := user.CreatedAt
	for _, candidate := range []*time.Time{user.LastLogin, user.LastSeenAt} {
		if candidate != nil && candidate.After(at) {
			at = *candidate
		}
	}
	return at
}

func (handler *Handler) presenceContacts(user int64) ([]int64, error) {
	var contacts []int64
	err := handler.DB.Table("chats_chat_participants AS mine").
		Select("DISTINCT other.user_id").
		Joins("JOIN chats_chat_participants AS other ON other.chat_id = mine.chat_id AND other.user_id <> mine.user_id").
		Joins("JOIN chats_chat AS chat ON chat.id = mine.chat_id").
		Where("mine.user_id = ? AND chat.chat_type = ? AND chat.is_saved_messages = ?", user, "direct", false).
		Scan(&contacts).Error
	return contacts, err
}

func (handler *Handler) updatePresence(user int64, session string, connected bool, lease time.Duration) {
	online, changed := handler.Hub.SetPresence(user, session, connected, lease)
	now := time.Now().UTC()
	if connected || changed {
		cutoff := now
		if !changed {
			cutoff = now.Add(-10 * time.Second)
		}
		err := handler.DB.Model(&models.User{}).
			Where("id = ? AND (last_seen_at IS NULL OR last_seen_at < ?)", user, cutoff).
			UpdateColumn("last_seen_at", now).Error
		if err != nil {
			slog.Error("record chat last seen", "error", err)
		}
	}
	if !changed {
		return
	}
	contacts, err := handler.presenceContacts(user)
	if err != nil {
		slog.Error("chat presence contacts", "error", err)
		return
	}
	for _, contact := range contacts {
		handler.Hub.Broadcast(fmt.Sprint("user:", contact), map[string]any{
			"type":	"presence_update", "user_id": user, "online": online, "last_seen_at": now,
		})
	}
}

func (handler *Handler) presenceSnapshot(user int64) (map[string]any, error) {
	contacts, err := handler.presenceContacts(user)
	if err != nil {
		return nil, err
	}
	timestamps := make(map[int64]time.Time, len(contacts))
	if len(contacts) != 0 {
		var users []models.User
		err = handler.DB.Select("id", "created_at", "last_login", "last_seen_at").Where("id IN ?", contacts).Find(&users).Error
		if err != nil {
			return nil, err
		}
		for _, contact := range users {
			if at := lastSeen(contact); !at.IsZero() {
				timestamps[contact.ID] = at
			}
		}
	}
	return map[string]any{
		"type":	"presence_snapshot", "users": handler.Hub.Presence(contacts), "last_seen_at": timestamps,
	}, nil
}

func (handler *Handler) Presence(req *Request) any {
	req.W.Header().Set("Cache-Control", "no-store")
	switch req.R.Method {
	case http.MethodPost, http.MethodDelete:
		connected := req.R.Method == http.MethodPost
		id := req.R.URL.Query().Get("session_id")
		if connected {
			id = Str(req.Body()["session_id"])
		}
		if !sessionIDPattern.MatchString(id) {
			Fail(400, "Invalid presence session")
		}
		handler.updatePresence(req.User.ID, "http:"+id, connected, httpPresenceLease)
		if !connected {
			return map[string]any{"status": "ok"}
		}
	}
	event, err := handler.presenceSnapshot(req.User.ID)
	Check(err)
	return event
}

func (handler *Handler) sendPresence(peer *Peer, user int64) {
	event, err := handler.presenceSnapshot(user)
	if err != nil {
		slog.Error("chat presence snapshot", "error", err)
		return
	}
	select {
	case peer.Send <- event:
	case <-peer.Done:
	default:
		peer.Close()
	}
}
