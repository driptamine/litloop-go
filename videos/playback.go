package videos

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strconv"

	"gorm.io/gorm/clause"
)

func (handler *Handler) Playback(req *Request) any {
	var id int64
	if req.R.Method == "GET" {
		id = Num(req.R.URL.Query().Get("video_id"))
	} else {
		id = Num(req.Body()["video_id"])
	}
	handler.VideoAccess(req, id)
	var progress models.VideoWatchProgress
	_ = handler.DB.First(&progress, "video_id = ? AND user_id = ?", id, req.User.ID).Error
	progress.VideoID = id
	progress.UserID = req.User.ID
	if req.R.Method == "POST" {
		seconds, err := strconv.ParseFloat(Str(req.Body()["playback_time"]), 64)
		if err != nil || seconds < 0 {
			Fail(400, "Invalid playback_time")
		}
		progress.PlaybackTime = seconds
		Check(handler.DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "video_id"}}, DoUpdates: clause.AssignmentColumns([]string{"playback_time", "updated_at"})}).Create(&progress).Error)
	}
	return map[string]any{"video_id": id, "playback_time": progress.PlaybackTime}
}
