package queries

import "litloop/backend/internal/httpapi"

type Handler struct{ *httpapi.Server }

func New(server *httpapi.Server) *Handler	{ return &Handler{Server: server} }
