package videos

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) VideoAccess(req *Request, id int64) models.Video {
	video := Get[models.Video](handler.DB, id)
	if video.Visibility != "public" && (req.User == nil || Val(video.UserID) != req.User.ID) {
		var count int64
		Check(handler.DB.Model(&models.PostVideo{}).Where("video_id = ?", id).Count(&count).Error)
		if count == 0 {
			Fail(403, "Private video")
		}
	}
	return video
}
