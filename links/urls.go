package links

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("POST", "/links/save", true, server.LinkSave)
	server.Route("GET", "/links/all", true, server.Links)
	server.Route("GET", "/links/search", true, server.Links)
	server.Route("GET", "/links/search/query", true, server.Links)
	server.Route("POST", "/links/upload-csv", true, server.ImportLinks)
	server.Route("POST", "/links/upload-csv-bulk", true, server.ImportLinks)
}
