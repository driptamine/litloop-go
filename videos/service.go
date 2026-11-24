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

func (handler *Handler) VideoUpdate(req *Request) any {
	video := Get[models.Video](handler.DB, req.ID("id"))
	Owned(Val(video.UserID), req.User.ID)
	body := req.Body()
	if title, ok := body["title"]; ok {
		video.Title = Str(title)
	}
	if description, ok := body["description"]; ok {
		video.Description = Str(description)
	}
	if visibility, ok := body["visibility"]; ok {
		v := Str(visibility)
		if v != "private" && v != "public" && v != "unlisted" {
			Fail(400, "Invalid visibility")
		}
		video.Visibility = v
	}
	if thumbnail, ok := body["thumbnail"]; ok {
		video.Thumbnail = Str(thumbnail)
	}
	Check(handler.DB.Save(&video).Error)
	return handler.MediaJSON(video)
}
