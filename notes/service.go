package notes

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) OwnedPage(req *Request, id int64) models.Page {
	return Get[models.Page](handler.DB.Where("author_id = ?", req.User.ID), id)
}
