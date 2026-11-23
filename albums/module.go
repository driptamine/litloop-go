package albums

import (
	. "litloop/backend/internal/httpapi"
)

type Handler struct{ *Server }

func New(server *Server) *Handler	{ return &Handler{Server: server} }
