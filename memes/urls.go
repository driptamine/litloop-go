package memes

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/memes", false, s.Memes)
	s.Route("GET", "/memes/list", false, s.Memes)
}
