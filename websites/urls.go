package websites

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/websites/bing/search", false, s.Search)
	s.Route("GET", "/websites/bing/images/search", false, s.Search)
	s.Route("GET,POST", "/websites/brave/search", false, s.Search)
	s.Route("GET", "/websites/brave/images/search", false, s.Search)
	s.Route("GET", "/websites/google/custom/search", false, s.Search)
	s.Route("GET", "/websites/all", false, s.Search)
	s.Route("GET", "/websites/search", false, s.Search)
	s.Route("GET,POST", "/websites/brave/scrape", false, s.Search)
}
