package playlists

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/playlist", false, s.Playlist)
	s.Route("POST", "/playlist", true, s.Playlist)
	s.Route("GET", "/playlist/{key}", false, s.Playlist)
	s.Route("GET", "/playlist/{key}/tracks", false, s.Playlist)
}
