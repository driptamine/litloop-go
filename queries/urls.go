package queries

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/queries/bing/search", false, server.Query)
	server.Route("GET", "/queries/brave/search", false, server.Query)
	server.Route("GET", "/queries/brave/images/search", false, server.Query)
	server.Route("GET", "/queries/google/firefox/search", false, server.Query)
	server.Route("GET", "/queries/google/search-old", false, server.Query)
	server.Route("GET", "/queries/google/all", false, server.Query)
	server.Route("GET", "/queries/google/all/search", false, server.Query)
	server.Route("GET", "/queries/google/search", false, server.Query)
}
