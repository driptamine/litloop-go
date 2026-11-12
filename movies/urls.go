package movies

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/movies/list", false, s.Movies)
	s.Route("GET", "/movies/feed", false, s.Movies)
	s.Route("GET", "/movies/imdbmovies/list", false, s.Movies)
	s.Route("GET", "/movies/imdbmovies/{imdb}", false, s.Movies)
	s.Route("GET", "/movies/{id:[0-9]+}", false, s.Movies)
	s.Route("POST", "/movies/add", true, s.MovieAdd)
	s.Route("POST", "/movies/impressions/batch", true, s.Impressions)
	s.Route("POST", "/movies/{id:[0-9]+}/like", true, s.MovieLike)
	s.Route("GET", "/movies/watchlist", true, s.Watchlist)
	s.Route("GET", "/movies/watchlist/ammy_shuffle", true, s.Watchlist)
	s.Route("GET", "/movies/{username}/watchlist", false, s.Watchlist)
	s.Route("GET", "/movies/{username}/ratings", false, s.Watchlist)
	s.Route("POST", "/movies/import_watchlist_csv", true, s.ImportMovies)
	s.Route("POST", "/movies/import_ratings_csv", true, s.ImportMovies)
	s.Route("POST", "/movies/views/up", true, s.View)
}
