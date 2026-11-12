package movies

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) ImportMovies(q *Request) any {
	rows := CSVRows(q)
	ratings := strings.Contains(q.R.URL.Path, "ratings")
	created, updated := 0, 0
	q.Tx(func(tx *gorm.DB) error {
		for _, r := range rows {
			id := r["Const"]
			if id == "" {
				id = r["imdb_id"]
			}
			if id == "" {
				continue
			}
			m := models.ImdbMovie{ImdbID: id}
			e := tx.First(&m, "imdb_id = ?", id).Error
			if e == gorm.ErrRecordNotFound {
				m.Title = r["Title"]
				m.OriginalTitle = r["Original Title"]
				m.URL = r["URL"]
				m.TitleType = r["Title Type"]
				m.ImdbRating = ParseFloat(r["IMDb Rating"])
				m.RuntimeMinutes = ParseInt(r["Runtime (mins)"])
				m.Year = ParseInt(r["Year"])
				m.Genres = r["Genres"]
				m.NumVotes = ParseInt(r["Num Votes"])
				m.ReleaseDate = ParseDate(r["Release Date"])
				m.Directors = r["Directors"]
				m.CreatedAt = ParseDate(r["Created"])
				m.ModifiedAt = ParseDate(r["Modified"])
				m.Position = ParseInt(r["Position"])
				Check(tx.Create(&m).Error)
			} else {
				Check(e)
			}
			if ratings {
				rating := ParseFloat(r["Your Rating"])
				if rating == nil {
					continue
				}
				if *rating < 1 || *rating > 10 {
					Fail(400, "Rating must be 1–10")
				}
				var v models.ImdbMovieRating
				e = tx.First(&v, "user_id = ? AND imdb_movie_id = ?", q.User.ID, m.ID).Error
				if e == gorm.ErrRecordNotFound {
					created++
				} else {
					Check(e)
					updated++
				}
				v.UserID = q.User.ID
				v.ImdbMovieID = m.ID
				v.Rating = *rating
				v.DateRated = ParseDate(r["Date Rated"])
				Check(tx.Save(&v).Error)
			} else {
				var w models.WatchlistItem
				e = tx.First(&w, "user_id = ? AND movie_id = ?", q.User.ID, m.ID).Error
				if e == gorm.ErrRecordNotFound {
					created++
				} else {
					Check(e)
					updated++
				}
				w.UserID = q.User.ID
				w.MovieID = m.ID
				w.UserRating = ParseFloat(r["Your Rating"])
				w.Notes = r["Description"]
				Check(tx.Save(&w).Error)
			}
		}
		return nil
	})
	return map[string]any{"status": "success", "imported": created, "updated": updated, "ratings_created": created, "ratings_updated": updated}
}
