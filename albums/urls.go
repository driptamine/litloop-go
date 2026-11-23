package albums

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/album", false, server.Catalog)
	server.Route("GET", "/album/all", false, server.Catalog)
	server.Route("GET", "/album/search", false, server.Catalog)
	server.Route("GET", "/album/feed", false, server.Catalog)
	server.Route("GET", "/album/{key}/upd", false, server.Catalog)
	server.Route("GET", "/album/{key}", false, server.Catalog)
	server.Route("POST,PUT,DELETE", "/album/{key}/fike", true, server.AlbumLike)
	server.Route("POST", "/album/{id:[0-9]+}/view", true, server.View)
}
