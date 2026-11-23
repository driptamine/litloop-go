package notifications

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"net/http"
)

func (handler *Handler) Notifications(req *Request) any {
	if req.R.Method == http.MethodPost {
		list := IDs(req.Body()["ids"])
		Check(handler.DB.Model(&models.Notification{}).Where("to_user_id = ? AND id IN ?", req.User.ID, list).Update("is_read", true).Error)
		return map[string]any{"status": "ok"}
	}
	return All[models.Notification](handler.DB.Where("to_user_id = ?", req.User.ID).Order("id DESC").Limit(100))
}
