package jobs

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) TaskStatus(req *Request) any {
	req.Admin()
	return Get[models.Task](handler.DB, req.ID("id"))
}
