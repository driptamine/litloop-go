package videos

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strconv"

	"gorm.io/gorm/clause"
)

func (s *Handler) Playback(q *Request) any {
	var id int64
	if q.R.Method == "GET" {
		id = Num(q.R.URL.Query().Get("video_id"))
	} else {
		id = Num(q.Body()["video_id"])
	}
	s.VideoAccess(q, id)
	var p models.VideoWatchProgress
	_ = s.DB.First(&p, "video_id = ? AND user_id = ?", id, q.User.ID).Error
	p.VideoID = id
	p.UserID = q.User.ID
	if q.R.Method == "POST" {
		seconds, e := strconv.ParseFloat(Str(q.Body()["playback_time"]), 64)
		if e != nil || seconds < 0 {
			Fail(400, "Invalid playback_time")
		}
		p.PlaybackTime = seconds
		Check(s.DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "video_id"}}, DoUpdates: clause.AssignmentColumns([]string{"playback_time", "updated_at"})}).Create(&p).Error)
	}
	return map[string]any{"video_id": id, "playback_time": p.PlaybackTime}
}
