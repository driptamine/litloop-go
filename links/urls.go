package links

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("POST", "/links/save", true, s.LinkSave)
	s.Route("GET", "/links/all", true, s.Links)
	s.Route("GET", "/links/search", true, s.Links)
	s.Route("GET", "/links/search/query", true, s.Links)
	s.Route("POST", "/links/upload-csv", true, s.ImportLinks)
	s.Route("POST", "/links/upload-csv-bulk", true, s.ImportLinks)
}
