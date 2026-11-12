package notes

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) OwnedPage(q *Request, id int64) models.Page {
	return Get[models.Page](s.DB.Where("author_id = ?", q.User.ID), id)
}
