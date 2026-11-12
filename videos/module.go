package videos

import (
	. "litloop/backend/internal/httpapi"
)

type Handler struct{ *Server }

func New(s *Server) *Handler { return &Handler{Server: s} }
