package views

import (
	"litloop/backend/internal/engagement"
	. "litloop/backend/internal/httpapi"

	"gorm.io/gorm"
)

func (handler *Handler) View(req *Request) any {

	response := map[string]any{}
	if req.Param("id") == "" {
		response = req.Body()
	}
	kind := "post"
	id := int64(0)
	if req.Param("id") != "" {
		kind = "album"
		id = req.ID("id")
	} else {
		if response["movie_id"] != nil {
			kind = "movie"
			id = Num(response["movie_id"])
		} else {
			id = Num(response["post_id"])
		}
	}
	req.Tx(func(tx *gorm.DB) error {
		var err error
		switch kind {
		case "post":
			err = RecordPost(tx, id, req.User.ID)
		case "album":
			err = RecordAlbum(tx, id, req.User.ID)
		default:
			err = RecordMovie(tx, id, req.User.ID)
		}
		if err != nil {
			return err
		}
		return engagement.Enqueue(tx, handler.Config, kind, id, req.User.ID, "view")
	})
	req.Status = 202
	return map[string]any{"status": "accepted"}
}
