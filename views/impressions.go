package views

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) Impressions(q *Request) any {
	b := q.Body()
	kind := "post"
	table := "posts_post"
	key := "post_ids"
	if strings.HasPrefix(q.R.URL.Path, "/movies/") {
		kind = "movie"
		table = "movies_movie"
		key = "movie_ids"
	}
	list := IDs(b[key])
	if len(list) == 0 || len(list) > 1000 {
		Fail(400, "Expected 1–1000 IDs")
	}
	q.Tx(func(tx *gorm.DB) error {
		for _, id := range list {
			Check(tx.Table(table).Where("id = ?", id).UpdateColumn("impressions_count", gorm.Expr("impressions_count + 1")).Error)
			if kind == "post" {
				Check(tx.Create(&models.PostImpression{PostID: id, UserID: Ptr(q.User.ID)}).Error)
			} else {
				Check(tx.Create(&models.MovieImpression{MovieID: id, UserID: Ptr(q.User.ID)}).Error)
			}
		}
		return nil
	})
	return map[string]any{"status": "ok", "enqueued": len(list)}
}
