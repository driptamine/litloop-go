package chats

import (
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"time"
)

func (s *Handler) VoIP(q *Request, b map[string]any) {
	id := q.ID("id")
	typ := Str(b["type"])
	if typ == "voip_call_start" {
		call := models.Call{ChatID: id, CallerID: q.User.ID, CallType: "voice", Status: "initiated"}
		if Str(b["call_type"]) == "video" {
			call.CallType = "video"
		}
		Check(s.DB.Create(&call).Error)
		for _, p := range All[models.ChatParticipant](s.DB.Where("chat_id = ? AND user_id <> ?", id, q.User.ID)) {
			s.Hub.Broadcast("user:"+fmt.Sprint(p.UserID), map[string]any{"type": "incoming_call", "chat_id": id, "caller_id": q.User.ID, "caller_name": q.User.Username, "call_type": call.CallType})
		}
		return
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
		return
	}
	var c models.Call
	if s.DB.Where("chat_id = ? AND status IN ?", id, []string{"initiated", "ongoing"}).Order("id DESC").First(&c).Error == nil {
		Check(s.DB.Model(&c).Updates(updates).Error)
	}
}
