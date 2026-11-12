package notes

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/notes/pages", true, s.Pages)
	s.Route("POST", "/notes/pages/create", true, s.PageWrite)
	s.Route("GET", "/notes/pages/{id:[0-9]+}", true, s.Pages)
	s.Route("PUT,PATCH", "/notes/pages/{id:[0-9]+}/update", true, s.PageWrite)
	s.Route("DELETE", "/notes/pages/{id:[0-9]+}/delete", true, s.PageWrite)
	s.Route("POST", "/notes/pages/{id:[0-9]+}/blocks/create", true, s.BlockWrite)
	s.Route("PUT,PATCH", "/notes/blocks/{block:[0-9]+}/update", true, s.BlockWrite)
	s.Route("DELETE", "/notes/blocks/{block:[0-9]+}/delete", true, s.BlockWrite)
	s.Route("PUT", "/notes/blocks/{block:[0-9]+}/table", true, s.BlockWrite)
	s.Route("GET", "/notes/pages/{id:[0-9]+}/tags", true, s.PageTags)
	s.Route("POST", "/notes/pages/{id:[0-9]+}/tags/add", true, s.PageTags)
	s.Route("DELETE", "/notes/tags/{tag:[0-9]+}/delete", true, s.PageTags)
	s.Route("POST", "/notes/search-by-tags", true, s.Pages)
}
