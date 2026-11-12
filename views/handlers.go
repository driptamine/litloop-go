package views

import (
	. "litloop/backend/internal/httpapi"

	"gorm.io/gorm"
)

func (s *Handler) View(q *Request) any {

	b := map[string]any{}
	if q.Param("id") == "" {
		b = q.Body()
	}
	kind := "post"
	id := int64(0)
	if q.Param("id") != "" {
		kind = "album"
		id = q.ID("id")
	} else {
		if b["movie_id"] != nil {
			kind = "movie"
			id = Num(b["movie_id"])
		} else {
			id = Num(b["post_id"])
		}
	}
	q.Tx(func(tx *gorm.DB) error {
		switch kind {
		case "post":
			return RecordPost(tx, id, q.User.ID)
		case "album":
			return RecordAlbum(tx, id, q.User.ID)
		default:
			return RecordMovie(tx, id, q.User.ID)
		}
	})
	q.Status = 202
	return map[string]any{"status": "accepted"}
}
