package notes

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm/clause"
)

func (s *Handler) PageTags(q *Request) any {
	if q.Param("tag") != "" {
		t := Get[models.PageTag](s.DB, q.ID("tag"))
		s.OwnedPage(q, t.PageID)
		Check(s.DB.Delete(&t).Error)
		return map[string]any{"message": "Tag deleted"}
	}
	p := s.OwnedPage(q, q.ID("id"))
	if q.R.Method == "GET" {
		return All[models.PageTag](s.DB.Where("page_id = ?", p.ID).Order("name"))
	}
	name := strings.ToLower(strings.TrimSpace(Str(q.Body()["name"])))
	if name == "" || len([]rune(name)) > 50 {
		Fail(400, "name must contain 1–50 characters")
	}
	t := models.PageTag{PageID: p.ID, Name: name}
	Check(s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&t).Error)
	Check(s.DB.First(&t, "page_id = ? AND name = ?", p.ID, name).Error)
	q.Status = 201
	return t
}
