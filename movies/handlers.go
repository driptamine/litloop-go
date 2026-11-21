package movies

import (
	"litloop/backend/internal/engagement"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Handler) Movies(q *Request) any {
	path := q.R.URL.Path
	if q.Param("imdb") != "" {
		var m models.ImdbMovie
		Check(s.DB.First(&m, "imdb_id = ?", q.Param("imdb")).Error)
		return imdbJSON(m)
	}
	if q.Param("id") != "" {
		return Get[models.Movie](s.DB, q.ID("id"))
	}
	if strings.HasSuffix(path, "/feed") {
		return s.MovieFeed(q)
	}
	term := q.R.URL.Query().Get("search")
	if term == "" {
		term = q.R.URL.Query().Get("q")
	}
	if strings.Contains(path, "imdbmovies") {
		db := s.DB.Model(&models.ImdbMovie{})
		if term != "" {
			db = db.Where("LOWER(title) LIKE ?", "%"+strings.ToLower(term)+"%")
		}
		if v := q.R.URL.Query().Get("imdb_id"); v != "" {
			db = db.Where("imdb_id IN ?", strings.Split(v, ","))
		}
		if v := q.R.URL.Query().Get("year"); v != "" {
			db = db.Where("year = ?", Num(v))
		}
		if v := q.R.URL.Query().Get("title_type"); v != "" {
			db = db.Where("title_type = ?", v)
		}
		order := q.R.URL.Query().Get("ordering")
		if order == "" {
			order = "-year"
		}
		column := strings.TrimPrefix(order, "-")
		valid := map[string]bool{"year": true, "title": true, "id": true, "imdb_rating": true, "created_at": true, "release_date": true, "num_votes": true}
		if !valid[column] {
			Fail(400, "Invalid ordering")
		}
		if strings.HasPrefix(order, "-") {
			column += " DESC"
		}
		return q.PageSize(db.Order(column), &[]models.ImdbMovie{}, func(v any) any {
			out := []any{}
			for _, m := range *v.(*[]models.ImdbMovie) {
				out = append(out, imdbJSON(m))
			}
			return out
		}, 20)
	}
	db := s.DB.Model(&models.Movie{})
	if term != "" {
		db = db.Where("LOWER(title) LIKE ?", "%"+strings.ToLower(term)+"%")
	}
	if v := q.R.URL.Query().Get("tmdb_ids"); v != "" {
		list := []int64{}
		for _, id := range strings.Split(v, ",") {
			list = append(list, Num(strings.TrimSpace(id)))
		}
		db = db.Where("tmdb_id IN ?", list)
	}
	return q.PageSize(db.Order("id"), &[]models.Movie{}, func(v any) any { return v }, 20)
}
func (s *Handler) MovieAdd(q *Request) any {
	b := q.Body()
	var m models.Movie
	if b["tmdb_id"] != nil {
		tmdb := Num(b["tmdb_id"])
		if tmdb <= 0 {
			Fail(400, "Invalid tmdb_id")
		}
		m.TmdbID = Ptr(tmdb)
		var existing models.Movie
		if s.DB.First(&existing, "tmdb_id = ?", tmdb).Error == nil {
			return existing
		}
		if s.Config.TMDBKey != "" {
			data := s.FetchJSON(q, "GET", "https://api.themoviedb.org/3/movie/"+Str(tmdb)+"?api_key="+s.Config.TMDBKey, nil, nil).(map[string]any)
			m.Title = Str(data["title"])
			m.Description = Str(data["overview"])
			if p := Str(data["poster_path"]); p != "" {
				m.Poster = "https://image.tmdb.org/t/p/w500" + p
			}
		}
	}
	if title := Str(b["title"]); title != "" {
		m.Title = title
	}
	if m.Title == "" {
		Fail(400, "title is required, or configure TMDB_API_KEY")
	}
	if v, ok := b["description"]; ok {
		m.Description = Str(v)
	}
	if v, ok := b["poster"]; ok {
		m.Poster = Str(v)
	}
	Check(s.DB.Create(&m).Error)
	q.Status = 201
	return m
}
func (s *Handler) MovieLike(q *Request) any {
	id := q.ID("id")
	var m models.Movie
	Check(s.DB.First(&m, "tmdb_id = ?", id).Error)
	liked := false
	var count int64
	q.Tx(func(tx *gorm.DB) error {
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&m, m.ID).Error)
		var l models.MovieLike
		e := tx.First(&l, "movie_id = ? AND user_id = ?", m.ID, q.User.ID).Error
		if e == gorm.ErrRecordNotFound {
			Check(tx.Create(&models.MovieLike{MovieID: m.ID, UserID: q.User.ID}).Error)
			liked = true
		} else {
			Check(e)
			Check(tx.Delete(&l).Error)
		}
		Check(tx.Model(&models.MovieLike{}).Where("movie_id = ?", m.ID).Count(&count).Error)
		Check(tx.Model(&m).Update("likes_count", count).Error)
		return engagement.EnqueueReaction(tx, s.Config, "movie", m.ID, q.User.ID, liked)
	})
	return map[string]any{"liked": liked, "likes_count": count}
}
