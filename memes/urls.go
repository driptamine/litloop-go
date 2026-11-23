package memes

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/memes", false, server.Memes)
	server.Route("GET", "/memes/list", false, server.Memes)
}
