package movies

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"
)

func (s *Handler) Watchlist(q *Request) any {
	uid := int64(0)
	if username := q.Param("username"); username != "" {
		var u models.User
		Check(s.DB.First(&u, "username = ?", username).Error)
		uid = u.ID
	} else {
		if q.User == nil {
			Fail(401, "Authentication required")
		}
		uid = q.User.ID
	}
	return s.WatchlistFor(q, uid, strings.HasSuffix(q.R.URL.Path, "/ratings"))
}
func (s *Handler) WatchlistFor(q *Request, uid int64, ratings bool) any {
	defaultSize := 10
	if ratings {
		defaultSize = 20
	}
	page, size := Pagination(q, defaultSize, "page_size")
	shuffle := strings.HasSuffix(q.R.URL.Path, "/ammy_shuffle")
	var total int64
	out := []any{}
	if ratings {
		db := s.DB.Model(&models.ImdbMovieRating{}).Where("user_id = ?", uid)
		Check(db.Count(&total).Error)
		for _, r := range All[models.ImdbMovieRating](db.Order("date_rated DESC, updated_at DESC").Offset((page - 1) * size).Limit(size)) {
			m := Get[models.ImdbMovie](s.DB, r.ImdbMovieID)
			o := imdbJSON(m)
			o["rating"] = r.Rating
			o["user_rating"] = r.Rating
			o["date_rated"] = Date(r.DateRated)
			o["created_at"] = r.CreatedAt
			o["updated_at"] = r.UpdatedAt
			out = append(out, o)
		}
	} else {
		db := s.DB.Model(&models.WatchlistItem{}).Where("user_id = ?", uid)
		Check(db.Count(&total).Error)
		if shuffle {
			size = int(total)
			if raw := q.R.URL.Query().Get("limit"); raw != "" {
				size = int(Num(raw))
				if size < 0 {
					Fail(400, "Invalid limit")
				}
			}
			db = db.Order("RANDOM()").Limit(size)
		} else {
			db = db.Order("date_rated DESC, id DESC").Offset((page - 1) * size).Limit(size)
		}
		for _, w := range All[models.WatchlistItem](db) {
			m := Get[models.ImdbMovie](s.DB, w.MovieID)
			o := imdbJSON(m)
			o["your_rating"] = w.UserRating
			o["user_rating"] = w.UserRating
			o["date_rated"] = Date(w.DateRated)
			o["watched"] = w.Watched
			o["notes"] = w.Notes
			o["date_added"] = w.DateAdded.Format("2006-01-02")
			out = append(out, o)
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
	if name := q.Param("username"); name != "" {
		result["username"] = name
	}
	return result
}
