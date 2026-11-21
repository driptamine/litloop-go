package views

import (
	"litloop/backend/internal/engagement"
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
		var err error
		switch kind {
		case "post":
			err = RecordPost(tx, id, q.User.ID)
		case "album":
			err = RecordAlbum(tx, id, q.User.ID)
		default:
			err = RecordMovie(tx, id, q.User.ID)
		}
		if err != nil {
			return err
		}
		return engagement.Enqueue(tx, s.Config, kind, id, q.User.ID, "view")
	})
	q.Status = 202
	return map[string]any{"status": "accepted"}
}
