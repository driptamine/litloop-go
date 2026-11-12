package jobs

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) TaskStatus(q *Request) any { q.Admin(); return Get[models.Task](s.DB, q.ID("id")) }
