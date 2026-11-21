package chats

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"net/http"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	groupVoiceType  = "groupvoice"
	groupVoiceLease = 90 * time.Second
)

type voiceMember struct {
	UserID    int64  `json:"user_id"`
	SessionID string `json:"session_id"`
	Username  string `json:"username"`
}

type voiceRoom struct {
	ID      int64         `json:"id"`
	Members []voiceMember `json:"members"`
}

func activeVoiceRoom(db *gorm.DB, chatID int64, cutoff time.Time) (models.Call, error) {
	var call models.Call
	err := db.Where("chat_id = ? AND call_type = ? AND status = ?", chatID, groupVoiceType, "ongoing").
		Where(`EXISTS (SELECT 1 FROM chats_call_participant AS voice
			JOIN chats_chat_participants AS membership ON membership.user_id = voice.user_id
			WHERE voice.call_id = chats_call.id AND membership.chat_id = chats_call.chat_id AND voice.seen_at > ?)`, cutoff).
		Order("id DESC").First(&call).Error
	return call, err
}

func voiceMembers(db *gorm.DB, call models.Call) []voiceMember {
	members := []voiceMember{}
	Check(db.Table("chats_call_participant AS voice").
		Select("voice.user_id, voice.session_id, users_user.username").
		Joins("JOIN chats_chat_participants AS membership ON membership.user_id = voice.user_id AND membership.chat_id = ?", call.ChatID).
		Joins("JOIN users_user ON users_user.id = voice.user_id").
		Where("voice.call_id = ? AND voice.seen_at > ?", call.ID, time.Now().Add(-groupVoiceLease)).
		Order("voice.user_id").Scan(&members).Error)
	return members
}

func (s *Handler) joinGroupVoice(q *Request, chatID int64, session string) any {
	var call models.Call
	q.Tx(func(tx *gorm.DB) error {
		var chat models.Chat
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&chat, chatID).Error)
		now := time.Now()
		var err error
		call, err = activeVoiceRoom(tx, chatID, now.Add(-groupVoiceLease))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Check(tx.Model(&models.Call{}).Where("chat_id = ? AND call_type = ? AND status = ?", chatID, groupVoiceType, "ongoing").
				Updates(map[string]any{"status": "completed", "ended_at": now}).Error)
			call = models.Call{ChatID: chatID, CallerID: q.User.ID, CallType: groupVoiceType, Status: "ongoing"}
			Check(tx.Create(&call).Error)
		} else {
			Check(err)
		}
		member := models.CallParticipant{CallID: call.ID, UserID: q.User.ID, SessionID: session, SeenAt: now}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "call_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"session_id", "seen_at"}),
		}).Create(&member).Error
	})
	return map[string]any{"room": voiceRoom{ID: call.ID, Members: voiceMembers(s.DB, call)}}
}

func (s *Handler) leaveGroupVoice(q *Request, call models.Call, session string) {
	q.Tx(func(tx *gorm.DB) error {
		var chat models.Chat
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&chat, call.ChatID).Error)
		Check(tx.Where("call_id = ? AND user_id = ? AND session_id = ?", call.ID, q.User.ID, session).
			Delete(&models.CallParticipant{}).Error)
		if len(voiceMembers(tx, call)) == 0 {
			return tx.Model(&call).Updates(map[string]any{"status": "completed", "ended_at": time.Now()}).Error
		}
		return nil
	})
}

func (s *Handler) GroupVoice(q *Request) any {
	q.W.Header().Set("Cache-Control", "no-store")
	chat := s.ChatAccess(q, q.ID("id"))
	if chat.ChatType != "groupchat" {
		Fail(400, "Group voice is available in group chats")
	}
	if q.R.Method == http.MethodGet {
		call, err := activeVoiceRoom(s.DB, chat.ID, time.Now().Add(-groupVoiceLease))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[string]any{"room": nil}
		}
		Check(err)
		return map[string]any{"room": voiceRoom{ID: call.ID, Members: voiceMembers(s.DB, call)}}
	}

	b := q.Body()
	session := Str(b["session_id"])
	if !sessionIDPattern.MatchString(session) {
		Fail(400, "Invalid voice session")
	}
	action := Str(b["action"])
	if action == "join" {
		return s.joinGroupVoice(q, chat.ID, session)
	}

	var call models.Call
	err := s.DB.Where("id = ? AND chat_id = ? AND call_type = ? AND status = ?",
		Num(b["room_id"]), chat.ID, groupVoiceType, "ongoing").First(&call).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if action == "leave" {
			return map[string]any{"status": "ok"}
		}
		Fail(409, "The voice room is closed")
	}
	Check(err)

	member := s.DB.Model(&models.CallParticipant{}).Where(
		"call_id = ? AND user_id = ? AND session_id = ? AND seen_at > ?",
		call.ID, q.User.ID, session, time.Now().Add(-groupVoiceLease),
	)
	switch action {
	case "leave":
		s.leaveGroupVoice(q, call, session)
		return map[string]any{"status": "ok"}
	case "sync":
		result := member.UpdateColumn("seen_at", time.Now())
		Check(result.Error)
		if result.RowsAffected != 1 {
			Fail(409, "This voice session is no longer active")
		}
		signals, err := s.Hub.GroupVoiceSignals(call.ID, q.User.ID, session)
		Check(err)
		if after := Str(b["after"]); after != "" {
			for i, raw := range signals {
				var frame struct {
					ID string `json:"signal_id"`
				}
				if json.Unmarshal(raw, &frame) == nil && frame.ID == after {
					signals = signals[i+1:]
					break
				}
			}
		}
		return map[string]any{"room": voiceRoom{ID: call.ID, Members: voiceMembers(s.DB, call)}, "signals": signals}
	case "signal":
		var count int64
		Check(member.Count(&count).Error)
		if count == 0 {
			Fail(409, "This voice session is no longer active")
		}
		targetUser := Num(b["target_user_id"])
		targetSession := Str(b["target_session_id"])
		for _, target := range voiceMembers(s.DB, call) {
			if target.UserID == targetUser && targetUser != q.User.ID && target.SessionID == targetSession {
				signal, ok := b["signal_data"].(map[string]any)
				if !ok || (Str(signal["type"]) != "offer" && Str(signal["type"]) != "answer" && Str(signal["type"]) != "candidate") {
					Fail(400, "Invalid voice signal")
				}
				Check(s.Hub.SaveGroupVoiceSignal(call.ID, targetUser, targetSession, map[string]any{
					"signal_id": rand.Text(), "from_user_id": q.User.ID, "from_session_id": session, "signal_data": signal,
				}))
				return map[string]any{"status": "ok"}
			}
		}
		Fail(409, "This voice participant has left")
	}
	Fail(400, "Unknown group voice action")
	return nil
}
