package artists

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/artist", false, s.Catalog)
	s.Route("GET", "/artist/all", false, s.Catalog)
	s.Route("GET", "/artist/search", false, s.Catalog)
	s.Route("GET", "/artist/{key}", false, s.Catalog)
	s.Route("GET", "/artist/{key}/upd", false, s.Catalog)
	s.Route("GET", "/artist/{key}/albums", false, s.Catalog)
	s.Route("GET", "/artist/{key}/albums/upd", false, s.Catalog)
}
