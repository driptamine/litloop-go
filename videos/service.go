package videos

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) VideoAccess(q *Request, id int64) models.Video {
	v := Get[models.Video](s.DB, id)
	if v.Visibility != "public" && (q.User == nil || Val(v.UserID) != q.User.ID) {
		var n int64
		Check(s.DB.Model(&models.PostVideo{}).Where("video_id = ?", id).Count(&n).Error)
		if n == 0 {
			Fail(403, "Private video")
		}
	}
	return v
}
