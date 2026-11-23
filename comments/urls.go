package comments

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/comments", false, server.Comments)
	server.Route("POST", "/comments/create", true, server.CommentCreate)
	server.Route("GET", "/comments/{uid}", false, server.Comments)
	server.Route("DELETE", "/comments/{uid}/delete", true, server.Comments)
}
