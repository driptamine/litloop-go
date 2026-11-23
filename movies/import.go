package movies

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (handler *Handler) ImportMovies(req *Request) any {
	rows := CSVRows(req)
	ratings := strings.Contains(req.R.URL.Path, "ratings")
	created, updated := 0, 0
	req.Tx(func(tx *gorm.DB) error {
		for _, row := range rows {
			id := row["Const"]
			if id == "" {
				id = row["imdb_id"]
			}
			if id == "" {
				continue
			}
			movie := models.ImdbMovie{ImdbID: id}
			err := tx.First(&movie, "imdb_id = ?", id).Error
			if err == gorm.ErrRecordNotFound {
				movie.Title = row["Title"]
				movie.OriginalTitle = row["Original Title"]
				movie.URL = row["URL"]
				movie.TitleType = row["Title Type"]
				movie.ImdbRating = ParseFloat(row["IMDb Rating"])
				movie.RuntimeMinutes = ParseInt(row["Runtime (mins)"])
				movie.Year = ParseInt(row["Year"])
				movie.Genres = row["Genres"]
				movie.NumVotes = ParseInt(row["Num Votes"])
				movie.ReleaseDate = ParseDate(row["Release Date"])
				movie.Directors = row["Directors"]
				movie.CreatedAt = ParseDate(row["Created"])
				movie.ModifiedAt = ParseDate(row["Modified"])
				movie.Position = ParseInt(row["Position"])
				Check(tx.Create(&movie).Error)
			} else {
				Check(err)
			}
			if ratings {
				rating := ParseFloat(row["Your Rating"])
				if rating == nil {
					continue
				}
				if *rating < 1 || *rating > 10 {
					Fail(400, "Rating must be 1–10")
				}
				var movieRating models.ImdbMovieRating
				err = tx.First(&movieRating, "user_id = ? AND imdb_movie_id = ?", req.User.ID, movie.ID).Error
				if err == gorm.ErrRecordNotFound {
					created++
				} else {
					Check(err)
					updated++
				}
				movieRating.UserID = req.User.ID
				movieRating.ImdbMovieID = movie.ID
				movieRating.Rating = *rating
				movieRating.DateRated = ParseDate(row["Date Rated"])
				Check(tx.Save(&movieRating).Error)
			} else {
				var watchlistItem models.WatchlistItem
				err = tx.First(&watchlistItem, "user_id = ? AND movie_id = ?", req.User.ID, movie.ID).Error
				if err == gorm.ErrRecordNotFound {
					created++
				} else {
					Check(err)
					updated++
				}
				watchlistItem.UserID = req.User.ID
				watchlistItem.MovieID = movie.ID
				watchlistItem.UserRating = ParseFloat(row["Your Rating"])
				watchlistItem.Notes = row["Description"]
				Check(tx.Save(&watchlistItem).Error)
			}
		}
		return nil
	})
	return map[string]any{"status": "success", "imported": created, "updated": updated, "ratings_created": created, "ratings_updated": updated}
}
