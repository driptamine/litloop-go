package notes

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/notes/pages", true, server.Pages)
	server.Route("POST", "/notes/pages/create", true, server.PageWrite)
	server.Route("GET", "/notes/pages/{id:[0-9]+}", true, server.Pages)
	server.Route("PUT,PATCH", "/notes/pages/{id:[0-9]+}/update", true, server.PageWrite)
	server.Route("DELETE", "/notes/pages/{id:[0-9]+}/delete", true, server.PageWrite)
	server.Route("POST", "/notes/pages/{id:[0-9]+}/blocks/create", true, server.BlockWrite)
	server.Route("PUT,PATCH", "/notes/blocks/{block:[0-9]+}/update", true, server.BlockWrite)
	server.Route("DELETE", "/notes/blocks/{block:[0-9]+}/delete", true, server.BlockWrite)
	server.Route("PUT", "/notes/blocks/{block:[0-9]+}/table", true, server.BlockWrite)
	server.Route("GET", "/notes/pages/{id:[0-9]+}/tags", true, server.PageTags)
	server.Route("POST", "/notes/pages/{id:[0-9]+}/tags/add", true, server.PageTags)
	server.Route("DELETE", "/notes/tags/{tag:[0-9]+}/delete", true, server.PageTags)
	server.Route("POST", "/notes/search-by-tags", true, server.Pages)
}
