package movies

import (
	"litloop/backend/internal/engagement"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (handler *Handler) Movies(req *Request) any {
	path := req.R.URL.Path
	if req.Param("imdb") != "" {
		var movie models.ImdbMovie
		Check(handler.DB.First(&movie, "imdb_id = ?", req.Param("imdb")).Error)
		return imdbJSON(movie)
	}
	if req.Param("id") != "" {
		return Get[models.Movie](handler.DB, req.ID("id"))
	}
	if strings.HasSuffix(path, "/feed") {
		return handler.MovieFeed(req)
	}
	term := req.R.URL.Query().Get("search")
	if term == "" {
		term = req.R.URL.Query().Get("q")
	}
	if strings.Contains(path, "imdbmovies") {
		db := handler.DB.Model(&models.ImdbMovie{})
		if term != "" {
			db = db.Where("LOWER(title) LIKE ?", "%"+strings.ToLower(term)+"%")
		}
		if value := req.R.URL.Query().Get("imdb_id"); value != "" {
			db = db.Where("imdb_id IN ?", strings.Split(value, ","))
		}
		if value := req.R.URL.Query().Get("year"); value != "" {
			db = db.Where("year = ?", Num(value))
		}
		if value := req.R.URL.Query().Get("title_type"); value != "" {
			db = db.Where("title_type = ?", value)
		}
		order := req.R.URL.Query().Get("ordering")
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
		return req.PageSize(db.Order(column), &[]models.ImdbMovie{}, func(value any) any {
			out := []any{}
			for _, movie := range *value.(*[]models.ImdbMovie) {
				out = append(out, imdbJSON(movie))
			}
			return out
		}, 20)
	}
	db := handler.DB.Model(&models.Movie{})
	if term != "" {
		db = db.Where("LOWER(title) LIKE ?", "%"+strings.ToLower(term)+"%")
	}
	if value := req.R.URL.Query().Get("tmdb_ids"); value != "" {
		list := []int64{}
		for _, id := range strings.Split(value, ",") {
			list = append(list, Num(strings.TrimSpace(id)))
		}
		db = db.Where("tmdb_id IN ?", list)
	}
	return req.PageSize(db.Order("id"), &[]models.Movie{}, func(value any) any { return value }, 20)
}
func (handler *Handler) MovieAdd(req *Request) any {
	body := req.Body()
	var movie models.Movie
	if body["tmdb_id"] != nil {
		tmdb := Num(body["tmdb_id"])
		if tmdb <= 0 {
			Fail(400, "Invalid tmdb_id")
		}
		movie.TmdbID = Ptr(tmdb)
		var existing models.Movie
		if handler.DB.First(&existing, "tmdb_id = ?", tmdb).Error == nil {
			return existing
		}
		if handler.Config.TMDBKey != "" {
			data := handler.FetchJSON(req, "GET", "https://api.themoviedb.org/3/movie/"+Str(tmdb)+"?api_key="+handler.Config.TMDBKey, nil, nil).(map[string]any)
			movie.Title = Str(data["title"])
			movie.Description = Str(data["overview"])
			if poster := Str(data["poster_path"]); poster != "" {
				movie.Poster = "https://image.tmdb.org/t/p/w500" + poster
			}
		}
	}
	if title := Str(body["title"]); title != "" {
		movie.Title = title
	}
	if movie.Title == "" {
		Fail(400, "title is required, or configure TMDB_API_KEY")
	}
	if value, ok := body["description"]; ok {
		movie.Description = Str(value)
	}
	if value, ok := body["poster"]; ok {
		movie.Poster = Str(value)
	}
	Check(handler.DB.Create(&movie).Error)
	req.Status = 201
	return movie
}
func (handler *Handler) MovieLike(req *Request) any {
	id := req.ID("id")
	var movie models.Movie
	Check(handler.DB.First(&movie, "tmdb_id = ?", id).Error)
	liked := false
	var count int64
	req.Tx(func(tx *gorm.DB) error {
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&movie, movie.ID).Error)
		var like models.MovieLike
		err := tx.First(&like, "movie_id = ? AND user_id = ?", movie.ID, req.User.ID).Error
		if err == gorm.ErrRecordNotFound {
			Check(tx.Create(&models.MovieLike{MovieID: movie.ID, UserID: req.User.ID}).Error)
			liked = true
		} else {
			Check(err)
			Check(tx.Delete(&like).Error)
		}
		Check(tx.Model(&models.MovieLike{}).Where("movie_id = ?", movie.ID).Count(&count).Error)
		Check(tx.Model(&movie).Update("likes_count", count).Error)
		return engagement.EnqueueReaction(tx, handler.Config, "movie", movie.ID, req.User.ID, liked)
	})
	return map[string]any{"liked": liked, "likes_count": count}
}
