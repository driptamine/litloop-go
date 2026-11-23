package websites

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/websites/bing/search", false, server.Search)
	server.Route("GET", "/websites/bing/images/search", false, server.Search)
	server.Route("GET,POST", "/websites/brave/search", false, server.Search)
	server.Route("GET", "/websites/brave/images/search", false, server.Search)
	server.Route("GET", "/websites/google/custom/search", false, server.Search)
	server.Route("GET", "/websites/all", false, server.Search)
	server.Route("GET", "/websites/search", false, server.Search)
	server.Route("GET,POST", "/websites/brave/scrape", false, server.Search)
}
