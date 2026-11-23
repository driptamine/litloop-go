package views

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("POST", "/views/up", true, server.View)
	server.Route("GET", "/views/memes", false, server.Memes)
}
