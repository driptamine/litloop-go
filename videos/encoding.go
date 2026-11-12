package videos

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Handler) EncodeVideo(q *Request) any {
	v := Get[models.Video](s.DB, q.ID("id"))
	Owned(Val(v.UserID), q.User.ID)
	var task models.Task
	q.Tx(func(tx *gorm.DB) error {
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&v, v.ID).Error)
		task = models.Task{Kind: "encode_video", Payload: models.AsJSON(map[string]any{"video_id": v.ID}), Status: "pending", RunAt: time.Now()}
		return tx.Create(&task).Error
	})
	q.Status = 202
	return task
}
