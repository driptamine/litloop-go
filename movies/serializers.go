package movies

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func imdbJSON(movie models.ImdbMovie) map[string]any {
	out := Object(movie)
	out["release_date"] = Date(movie.ReleaseDate)
	out["created_at"] = Date(movie.CreatedAt)
	out["modified_at"] = Date(movie.ModifiedAt)
	return out
}
