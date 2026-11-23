package notes

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm/clause"
)

func (handler *Handler) PageTags(req *Request) any {
	if req.Param("tag") != "" {
		tag := Get[models.PageTag](handler.DB, req.ID("tag"))
		handler.OwnedPage(req, tag.PageID)
		Check(handler.DB.Delete(&tag).Error)
		return map[string]any{"message": "Tag deleted"}
	}
	page := handler.OwnedPage(req, req.ID("id"))
	if req.R.Method == "GET" {
		return All[models.PageTag](handler.DB.Where("page_id = ?", page.ID).Order("name"))
	}
	name := strings.ToLower(strings.TrimSpace(Str(req.Body()["name"])))
	if name == "" || len([]rune(name)) > 50 {
		Fail(400, "name must contain 1–50 characters")
	}
	tag := models.PageTag{PageID: page.ID, Name: name}
	Check(handler.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&tag).Error)
	Check(handler.DB.First(&tag, "page_id = ? AND name = ?", page.ID, name).Error)
	req.Status = 201
	return tag
}
