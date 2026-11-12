package queries

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/queries/bing/search", false, s.Query)
	s.Route("GET", "/queries/brave/search", false, s.Query)
	s.Route("GET", "/queries/brave/images/search", false, s.Query)
	s.Route("GET", "/queries/google/firefox/search", false, s.Query)
	s.Route("GET", "/queries/google/search-old", false, s.Query)
	s.Route("GET", "/queries/google/all", false, s.Query)
	s.Route("GET", "/queries/google/all/search", false, s.Query)
	s.Route("GET", "/queries/google/search", false, s.Query)
}
