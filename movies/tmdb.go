package movies

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"

	"gorm.io/gorm"
)

func (s *Handler) MovieFeed(q *Request) any {
	if s.Config.TMDBKey == "" {
		Fail(503, "TMDB_API_KEY is not configured")
	}
	page, _ := Pagination(q, 20, "page_size")
	data := s.FetchJSON(q, "GET", "https://api.themoviedb.org/3/movie/popular?api_key="+s.Config.TMDBKey+"&page="+Str(page), nil, nil).(map[string]any)
	list, ok := data["results"].([]any)
	if !ok {
		Fail(502, "Invalid TMDB response")
	}
	q.Tx(func(tx *gorm.DB) error {
		for _, v := range list {
			item, ok := v.(map[string]any)
			if !ok {
				Fail(502, "Invalid TMDB movie")
			}
			id := Num(item["id"])
			var m models.Movie
			e := tx.First(&m, "tmdb_id = ?", id).Error
			if e == gorm.ErrRecordNotFound {
				m = models.Movie{TmdbID: Ptr(id), Title: Str(item["title"]), Description: Str(item["overview"])}
				if p := Str(item["poster_path"]); p != "" {
					m.Poster = "https://image.tmdb.org/t/p/w500" + p
				}
				if p := Str(item["backdrop_path"]); p != "" {
					m.MovieFile = "https://image.tmdb.org/t/p/w1280" + p
				}
				Check(tx.Create(&m).Error)
			} else {
				Check(e)
			}
			item["db_id"] = m.ID
			item["impressions_count"] = m.ImpressionsCount
		}
		return nil
	})
	return data
}
