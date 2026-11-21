package chats

import (
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"time"

	"gorm.io/gorm"
)

func (s *Handler) VoIP(q *Request, b map[string]any) bool {
	id := q.ID("id")
	typ := Str(b["type"])
	if typ == "voip_call_start" {
		call := models.Call{ChatID: id, CallerID: q.User.ID, CallType: "voice", Status: "initiated"}
		if Str(b["call_type"]) == "video" {
			call.CallType = "video"
		}
		Check(s.DB.Create(&call).Error)
		b["call_id"] = call.ID
		for _, p := range All[models.ChatParticipant](s.DB.Where("chat_id = ? AND user_id <> ?", id, q.User.ID)) {
			s.Hub.Broadcast("user:"+fmt.Sprint(p.UserID), map[string]any{
				"type": "incoming_call", "chat_id": id, "call_id": call.ID,
				"caller_id": q.User.ID, "caller_name": q.User.Username, "call_type": call.CallType,
			})
		}
		return true
	}
	updates := map[string]any{}
	switch typ {
	case "voip_call_accept":
		updates["status"] = "ongoing"
	case "voip_call_reject":
		updates["status"] = "rejected"
		updates["ended_at"] = time.Now()
	case "voip_call_end":
		updates["status"] = "completed"
		updates["ended_at"] = time.Now()
	default:
		return typ == "voip_webrtc_signal"
	}
	var c models.Call
	calls := s.DB.Where("chat_id = ? AND status IN ?", id, []string{"initiated", "ongoing"})
	if b["call_id"] != nil {
		calls = calls.Where("id = ?", Num(b["call_id"]))
	}
	err := calls.Order("id DESC").First(&c).Error
	if err == gorm.ErrRecordNotFound {
		return false
	}
	Check(err)
	Check(s.DB.Model(&c).Updates(updates).Error)
	b["call_id"] = c.ID
	if typ == "voip_call_reject" || typ == "voip_call_end" {
		for _, p := range All[models.ChatParticipant](s.DB.Where("chat_id = ? AND user_id <> ?", id, q.User.ID)) {
			s.Hub.Broadcast("user:"+fmt.Sprint(p.UserID), map[string]any{
				"type": "call_closed", "chat_id": id, "call_id": c.ID,
			})
		}
	}
	return true
}
