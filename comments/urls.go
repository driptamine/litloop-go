package comments

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/comments", false, s.Comments)
	s.Route("POST", "/comments/create", true, s.CommentCreate)
	s.Route("GET", "/comments/{uid}", false, s.Comments)
	s.Route("DELETE", "/comments/{uid}/delete", true, s.Comments)
}
