package movies

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"

	"gorm.io/gorm"
)

func (handler *Handler) MovieFeed(req *Request) any {
	if handler.Config.TMDBKey == "" {
		Fail(503, "TMDB_API_KEY is not configured")
	}
	page, _ := Pagination(req, 20, "page_size")
	data := handler.FetchJSON(req, "GET", "https://api.themoviedb.org/3/movie/popular?api_key="+handler.Config.TMDBKey+"&page="+Str(page), nil, nil).(map[string]any)
	list, ok := data["results"].([]any)
	if !ok {
		Fail(502, "Invalid TMDB response")
	}
	req.Tx(func(tx *gorm.DB) error {
		for _, entry := range list {
			item, ok := entry.(map[string]any)
			if !ok {
				Fail(502, "Invalid TMDB movie")
			}
			id := Num(item["id"])
			var movie models.Movie
			err := tx.First(&movie, "tmdb_id = ?", id).Error
			if err == gorm.ErrRecordNotFound {
				movie = models.Movie{TmdbID: Ptr(id), Title: Str(item["title"]), Description: Str(item["overview"])}
				if poster := Str(item["poster_path"]); poster != "" {
					movie.Poster = "https://image.tmdb.org/t/p/w500" + poster
				}
				if backdrop := Str(item["backdrop_path"]); backdrop != "" {
					movie.MovieFile = "https://image.tmdb.org/t/p/w1280" + backdrop
				}
				Check(tx.Create(&movie).Error)
			} else {
				Check(err)
			}
			item["db_id"] = movie.ID
			item["impressions_count"] = movie.ImpressionsCount
		}
		return nil
	})
	return data
}
