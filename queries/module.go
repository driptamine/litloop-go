package queries

import "litloop/backend/internal/httpapi"

type Handler struct{ *httpapi.Server }

func New(s *httpapi.Server) *Handler { return &Handler{Server: s} }
