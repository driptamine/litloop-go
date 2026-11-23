package chats

import (
	"crypto/rand"
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
		"type":	"incoming_call", "chat_id": call.ChatID, "call_id": call.ID,
		"caller_id":	call.CallerID, "caller_name": caller,
		"call_type":	call.CallType, "expires_at": call.StartedAt.Add(ringDuration),
	}
}

func (handler *Handler) IncomingCall(req *Request) any {
	chats := handler.DB.Model(&models.ChatParticipant{}).Select("chat_id").Where("user_id = ?", req.User.ID)
	var call models.Call
	err := handler.DB.Where("chat_id IN (?) AND caller_id <> ? AND status = ? AND started_at > ?",
		chats, req.User.ID, "initiated", time.Now().Add(-ringDuration)).
		Order("id DESC").First(&call).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return map[string]any{"incoming_call": nil}
	}
	Check(err)
	caller := Get[models.User](handler.DB, call.CallerID)
	return map[string]any{"incoming_call": incomingCallEvent(call, caller.Username)}
}

func (handler *Handler) CallCommand(req *Request) any {
	chat := handler.ChatAccess(req, req.ID("id"))
	if chat.IsSavedMessages {
		Fail(400, "This conversation does not support calls")
	}
	body := req.Body()
	if Str(body["type"]) != "voip_call_start" && body["call_id"] == nil {
		Fail(400, "Call ID is required")
	}
	if !handler.VoIP(req, body) {
		Fail(409, "Call is no longer available")
	}
	handler.broadcastCall(req, body)
	return map[string]any{"call_id": body["call_id"], "expires_at": body["expires_at"]}
}

func (handler *Handler) CallState(req *Request) any {
	req.W.Header().Set("Cache-Control", "no-store")
	chat := handler.ChatAccess(req, req.ID("id"))
	id := Num(req.R.URL.Query().Get("call_id"))
	var call models.Call
	err := handler.DB.Where("id = ? AND chat_id = ? AND (caller_id = ? OR receiver_id = ?)",
		id, chat.ID, req.User.ID, req.User.ID).First(&call).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		Fail(404, "Call is no longer available")
	}
	Check(err)
	status := call.Status
	if status == "initiated" && !time.Now().Before(call.StartedAt.Add(ringDuration)) {
		status = "expired"
	}
	result := map[string]any{"call_id": call.ID, "chat_id": call.ChatID, "status": status}
	if status == "ongoing" && req.R.URL.Query().Get("signals") == "1" {
		signals, err := handler.Hub.CallSignals(call.ID, req.User.ID)
		Check(err)
		result["signals"] = signals
	}
	return result
}

func (handler *Handler) broadcastCall(req *Request, body map[string]any) {
	body["from_user_id"] = req.User.ID
	body["chat_id"] = req.ID("id")
	body["username"] = req.User.Username
	handler.Hub.Broadcast(fmt.Sprint(req.ID("id")), body)
}

func (handler *Handler) notifyCallMembers(call models.Call, except int64, event map[string]any) {
	for _, participant := range All[models.ChatParticipant](handler.DB.Where("chat_id = ? AND user_id <> ?", call.ChatID, except)) {
		handler.Hub.Broadcast(fmt.Sprint("user:", participant.UserID), event)
	}
}

func (handler *Handler) startCall(req *Request, body map[string]any) {
	id := req.ID("id")
	call := models.Call{ChatID: id, CallerID: req.User.ID, CallType: "voice", Status: "initiated"}
	if Str(body["call_type"]) == "video" {
		call.CallType = "video"
	}
	req.Tx(func(tx *gorm.DB) error {
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
	event := incomingCallEvent(call, req.User.Username)
	body["call_id"] = call.ID
	body["expires_at"] = event["expires_at"]
	handler.notifyCallMembers(call, req.User.ID, event)
}

func (handler *Handler) VoIP(req *Request, body map[string]any) bool {
	typ := Str(body["type"])
	switch typ {
	case "voip_call_start":
		handler.startCall(req, body)
		return true
	case "voip_call_accept", "voip_call_reject", "voip_call_end", "voip_webrtc_signal":
	default:
		return false
	}

	calls := handler.DB.Where("chat_id = ? AND status IN ? AND call_type <> ?", req.ID("id"), []string{"initiated", "ongoing"}, groupVoiceType)
	if body["call_id"] != nil {
		calls = calls.Where("id = ?", Num(body["call_id"]))
	}
	var call models.Call
	err := calls.Order("id DESC").First(&call).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false
	}
	Check(err)

	actor := req.User.ID
	body["call_id"] = call.ID
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
		signal, ok := body["signal_data"].(map[string]any)
		if !ok {
			return false
		}
		switch Str(signal["type"]) {
		case "offer", "answer", "candidate":
		default:
			return false
		}
		signalID := rand.Text()
		body["signal_id"] = signalID
		event := map[string]any{
			"type":	typ, "chat_id": call.ChatID, "call_id": call.ID,
			"from_user_id":	actor, "signal_id": signalID, "signal_data": signal,
		}
		Check(handler.Hub.SaveCallSignal(call.ID, target, event))
		handler.Hub.Broadcast(fmt.Sprint("user:", target), event)
		return true
	}

	now := time.Now()
	updates := map[string]any{}
	switch typ {
	case "voip_call_accept", "voip_call_reject":
		if typ == "voip_call_accept" && call.Status == "ongoing" && call.ReceiverID != nil && actor == *call.ReceiverID {
			return true
		}
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
	result := handler.DB.Model(&models.Call{}).Where("id = ? AND status = ?", call.ID, call.Status).Updates(updates)
	Check(result.Error)
	if result.RowsAffected != 1 {
		return false
	}

	event := map[string]any{"chat_id": call.ChatID, "call_id": call.ID, "from_user_id": actor}
	if typ == "voip_call_accept" {
		event["type"] = "call_accepted"
		handler.Hub.Broadcast(fmt.Sprint("user:", call.CallerID), event)
	} else {
		event["type"] = "call_closed"
		handler.notifyCallMembers(call, actor, event)
	}
	return true
}
