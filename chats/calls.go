package chats

import (
	"errors"
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const ringDuration = 45 * time.Second

func incomingCallEvent(call models.Call, caller string) map[string]any {
	return map[string]any{
		"type": "incoming_call", "chat_id": call.ChatID, "call_id": call.ID,
		"caller_id": call.CallerID, "caller_name": caller,
		"call_type": call.CallType, "expires_at": call.StartedAt.Add(ringDuration),
	}
}

func (s *Handler) IncomingCall(q *Request) any {
	chats := s.DB.Model(&models.ChatParticipant{}).Select("chat_id").Where("user_id = ?", q.User.ID)
	var call models.Call
	err := s.DB.Where("chat_id IN (?) AND caller_id <> ? AND status = ? AND started_at > ?",
		chats, q.User.ID, "initiated", time.Now().Add(-ringDuration)).
		Order("id DESC").First(&call).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return map[string]any{"incoming_call": nil}
	}
	Check(err)
	caller := Get[models.User](s.DB, call.CallerID)
	return map[string]any{"incoming_call": incomingCallEvent(call, caller.Username)}
}

func (s *Handler) CallCommand(q *Request) any {
	chat := s.ChatAccess(q, q.ID("id"))
	if chat.IsSavedMessages {
		Fail(400, "This conversation does not support calls")
	}
	b := q.Body()
	if Str(b["type"]) != "voip_call_start" && b["call_id"] == nil {
		Fail(400, "Call ID is required")
	}
	if !s.VoIP(q, b) {
		Fail(409, "Call is no longer available")
	}
	s.broadcastCall(q, b)
	return map[string]any{"call_id": b["call_id"], "expires_at": b["expires_at"]}
}

func (s *Handler) broadcastCall(q *Request, b map[string]any) {
	b["from_user_id"] = q.User.ID
	b["chat_id"] = q.ID("id")
	b["username"] = q.User.Username
	s.Hub.Broadcast(fmt.Sprint(q.ID("id")), b)
}

func (s *Handler) notifyCallMembers(call models.Call, except int64, event map[string]any) {
	for _, participant := range All[models.ChatParticipant](s.DB.Where("chat_id = ? AND user_id <> ?", call.ChatID, except)) {
		s.Hub.Broadcast(fmt.Sprint("user:", participant.UserID), event)
	}
}

func (s *Handler) startCall(q *Request, b map[string]any) {
	id := q.ID("id")
	call := models.Call{ChatID: id, CallerID: q.User.ID, CallType: "voice", Status: "initiated"}
	if Str(b["call_type"]) == "video" {
		call.CallType = "video"
	}
	q.Tx(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&models.Chat{}, id).Error; err != nil {
			return err
		}
		var ringing int64
		Check(tx.Model(&models.Call{}).Where("chat_id = ? AND status = ? AND started_at > ?",
			id, "initiated", time.Now().Add(-ringDuration)).Count(&ringing).Error)
		if ringing != 0 {
			Fail(409, "A call is already ringing in this chat")
		}
		return tx.Create(&call).Error
	})
	event := incomingCallEvent(call, q.User.Username)
	b["call_id"] = call.ID
	b["expires_at"] = event["expires_at"]
	s.notifyCallMembers(call, q.User.ID, event)
}

func (s *Handler) VoIP(q *Request, b map[string]any) bool {
	typ := Str(b["type"])
	switch typ {
	case "voip_call_start":
		s.startCall(q, b)
		return true
	case "voip_call_accept", "voip_call_reject", "voip_call_end", "voip_webrtc_signal":
	default:
		return false
	}

	calls := s.DB.Where("chat_id = ? AND status IN ?", q.ID("id"), []string{"initiated", "ongoing"})
	if b["call_id"] != nil {
		calls = calls.Where("id = ?", Num(b["call_id"]))
	}
	var call models.Call
	err := calls.Order("id DESC").First(&call).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false
	}
	Check(err)

	actor := q.User.ID
	b["call_id"] = call.ID
	if typ == "voip_webrtc_signal" {
		if call.Status != "ongoing" || call.ReceiverID == nil {
			return false
		}
		target := call.CallerID
		if actor == call.CallerID {
			target = *call.ReceiverID
		} else if actor != *call.ReceiverID {
			return false
		}
		signal, ok := b["signal_data"].(map[string]any)
		if !ok {
			return false
		}
		switch Str(signal["type"]) {
		case "offer", "answer", "candidate":
		default:
			return false
		}
		s.Hub.Broadcast(fmt.Sprint("user:", target), map[string]any{
			"type": typ, "chat_id": call.ChatID, "call_id": call.ID,
			"from_user_id": actor, "signal_data": signal,
		})
		return true
	}

	now := time.Now()
	updates := map[string]any{}
	switch typ {
	case "voip_call_accept", "voip_call_reject":
		if call.Status != "initiated" || actor == call.CallerID || !now.Before(call.StartedAt.Add(ringDuration)) {
			return false
		}
		if typ == "voip_call_accept" {
			updates["status"] = "ongoing"
			updates["receiver_id"] = actor
		} else {
			updates["status"] = "rejected"
			updates["ended_at"] = now
		}
	case "voip_call_end":
		if actor != call.CallerID && (call.ReceiverID == nil || actor != *call.ReceiverID) {
			return false
		}
		updates["status"] = "completed"
		updates["ended_at"] = now
	}
	result := s.DB.Model(&models.Call{}).Where("id = ? AND status = ?", call.ID, call.Status).Updates(updates)
	Check(result.Error)
	if result.RowsAffected != 1 {
		return false
	}

	event := map[string]any{"chat_id": call.ChatID, "call_id": call.ID, "from_user_id": actor}
	if typ == "voip_call_accept" {
		event["type"] = "call_accepted"
		s.Hub.Broadcast(fmt.Sprint("user:", call.CallerID), event)
	} else {
		event["type"] = "call_closed"
		s.notifyCallMembers(call, actor, event)
	}
	return true
}
