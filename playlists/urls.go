package playlists

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/playlist", false, server.Playlist)
	server.Route("POST", "/playlist", true, server.Playlist)
	server.Route("GET", "/playlist/{key}", false, server.Playlist)
	server.Route("GET", "/playlist/{key}/tracks", false, server.Playlist)
}
