package views

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (handler *Handler) Impressions(req *Request) any {
	body := req.Body()
	kind := "post"
	table := "posts_post"
	key := "post_ids"
	if strings.HasPrefix(req.R.URL.Path, "/movies/") {
		kind = "movie"
		table = "movies_movie"
		key = "movie_ids"
	}
	list := IDs(body[key])
	if len(list) == 0 || len(list) > 1000 {
		Fail(400, "Expected 1–1000 IDs")
	}
	req.Tx(func(tx *gorm.DB) error {
		for _, id := range list {
			Check(tx.Table(table).Where("id = ?", id).UpdateColumn("impressions_count", gorm.Expr("impressions_count + 1")).Error)
			if kind == "post" {
				Check(tx.Create(&models.PostImpression{PostID: id, UserID: Ptr(req.User.ID)}).Error)
			} else {
				Check(tx.Create(&models.MovieImpression{MovieID: id, UserID: Ptr(req.User.ID)}).Error)
			}
		}
		return nil
	})
	return map[string]any{"status": "ok", "enqueued": len(list)}
}
