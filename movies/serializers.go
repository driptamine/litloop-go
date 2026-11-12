package movies

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func imdbJSON(m models.ImdbMovie) map[string]any {
	o := Object(m)
	o["release_date"] = Date(m.ReleaseDate)
	o["created_at"] = Date(m.CreatedAt)
	o["modified_at"] = Date(m.ModifiedAt)
	return o
}
