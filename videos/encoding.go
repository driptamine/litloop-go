package videos

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (handler *Handler) EncodeVideo(req *Request) any {
	video := Get[models.Video](handler.DB, req.ID("id"))
	Owned(Val(video.UserID), req.User.ID)
	var task models.Task
	req.Tx(func(tx *gorm.DB) error {
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&video, video.ID).Error)
		task = models.Task{Kind: "encode_video", Payload: models.AsJSON(map[string]any{"video_id": video.ID}), Status: "pending", RunAt: time.Now()}
		return tx.Create(&task).Error
	})
	req.Status = 202
	return task
}
