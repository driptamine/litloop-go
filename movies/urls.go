package movies

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/movies/list", false, server.Movies)
	server.Route("GET", "/movies/feed", false, server.Movies)
	server.Route("GET", "/movies/imdbmovies/list", false, server.Movies)
	server.Route("GET", "/movies/imdbmovies/{imdb}", false, server.Movies)
	server.Route("GET", "/movies/{id:[0-9]+}", false, server.Movies)
	server.Route("POST", "/movies/add", true, server.MovieAdd)
	server.Route("POST", "/movies/impressions/batch", true, server.Impressions)
	server.Route("POST", "/movies/{id:[0-9]+}/like", true, server.MovieLike)
	server.Route("GET", "/movies/watchlist", true, server.Watchlist)
	server.Route("GET", "/movies/watchlist/ammy_shuffle", true, server.Watchlist)
	server.Route("GET", "/movies/{username}/watchlist", false, server.Watchlist)
	server.Route("GET", "/movies/{username}/ratings", false, server.Watchlist)
	server.Route("POST", "/movies/import_watchlist_csv", true, server.ImportMovies)
	server.Route("POST", "/movies/import_ratings_csv", true, server.ImportMovies)
	server.Route("POST", "/movies/views/up", true, server.View)
}
