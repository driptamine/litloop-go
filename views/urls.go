package views

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("POST", "/views/up", true, s.View)
	s.Route("GET", "/views/memes", false, s.Memes)
}
