package artists

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/artist", false, server.Catalog)
	server.Route("GET", "/artist/all", false, server.Catalog)
	server.Route("GET", "/artist/search", false, server.Catalog)
	server.Route("GET", "/artist/{key}", false, server.Catalog)
	server.Route("GET", "/artist/{key}/upd", false, server.Catalog)
	server.Route("GET", "/artist/{key}/albums", false, server.Catalog)
	server.Route("GET", "/artist/{key}/albums/upd", false, server.Catalog)
}
