package movies

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"
)

func (handler *Handler) Watchlist(req *Request) any {
	uid := int64(0)
	if username := req.Param("username"); username != "" {
		var user models.User
		Check(handler.DB.First(&user, "username = ?", username).Error)
		uid = user.ID
	} else {
		if req.User == nil {
			Fail(401, "Authentication required")
		}
		uid = req.User.ID
	}
	return handler.WatchlistFor(req, uid, strings.HasSuffix(req.R.URL.Path, "/ratings"))
}
func (handler *Handler) WatchlistFor(req *Request, uid int64, ratings bool) any {
	defaultSize := 10
	if ratings {
		defaultSize = 20
	}
	page, size := Pagination(req, defaultSize, "page_size")
	shuffle := strings.HasSuffix(req.R.URL.Path, "/ammy_shuffle")
	var total int64
	out := []any{}
	if ratings {
		db := handler.DB.Model(&models.ImdbMovieRating{}).Where("user_id = ?", uid)
		Check(db.Count(&total).Error)
		for _, rating := range All[models.ImdbMovieRating](db.Order("date_rated DESC, updated_at DESC").Offset((page - 1) * size).Limit(size)) {
			movie := Get[models.ImdbMovie](handler.DB, rating.ImdbMovieID)
			movieJSON := imdbJSON(movie)
			movieJSON["rating"] = rating.Rating
			movieJSON["user_rating"] = rating.Rating
			movieJSON["date_rated"] = Date(rating.DateRated)
			movieJSON["created_at"] = rating.CreatedAt
			movieJSON["updated_at"] = rating.UpdatedAt
			out = append(out, movieJSON)
		}
	} else {
		db := handler.DB.Model(&models.WatchlistItem{}).Where("user_id = ?", uid)
		Check(db.Count(&total).Error)
		if shuffle {
			size = int(total)
			if raw := req.R.URL.Query().Get("limit"); raw != "" {
				size = int(Num(raw))
				if size < 0 {
					Fail(400, "Invalid limit")
				}
			}
			db = db.Order("RANDOM()").Limit(size)
		} else {
			db = db.Order("date_rated DESC, id DESC").Offset((page - 1) * size).Limit(size)
		}
		for _, watchlistItem := range All[models.WatchlistItem](db) {
			movie := Get[models.ImdbMovie](handler.DB, watchlistItem.MovieID)
			movieJSON := imdbJSON(movie)
			movieJSON["your_rating"] = watchlistItem.UserRating
			movieJSON["user_rating"] = watchlistItem.UserRating
			movieJSON["date_rated"] = Date(watchlistItem.DateRated)
			movieJSON["watched"] = watchlistItem.Watched
			movieJSON["notes"] = watchlistItem.Notes
			movieJSON["date_added"] = watchlistItem.DateAdded.Format("2006-01-02")
			out = append(out, movieJSON)
		}
	}
	if !shuffle && int64((page-1)*size) >= total && page > 1 {
		Fail(404, "Page out of range")
	}
	result := map[string]any{"count": total, "results": out, "total": total}
	if !shuffle {
		Pages := (total + int64(size) - 1) / int64(size)
		if Pages == 0 {
			Pages = 1
		}
		result["total_pages"] = Pages
		result["current_page"] = page
		result["page_size"] = size
	} else {
		result["count"] = len(out)
	}
	if ratings {
		result["ratings"] = out
	} else {
		result["watchlist"] = out
	}
	if name := req.Param("username"); name != "" {
		result["username"] = name
	}
	return result
}
