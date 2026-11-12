package notifications

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"net/http"
)

func (s *Handler) Notifications(q *Request) any {
	if q.R.Method == http.MethodPost {
		list := IDs(q.Body()["ids"])
		Check(s.DB.Model(&models.Notification{}).Where("to_user_id = ? AND id IN ?", q.User.ID, list).Update("is_read", true).Error)
		return map[string]any{"status": "ok"}
	}
	return All[models.Notification](s.DB.Where("to_user_id = ?", q.User.ID).Order("id DESC").Limit(100))
}
