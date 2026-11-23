package notes

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"litloop/backend/internal/tree"
	"strings"

	"gorm.io/gorm"
)

func (handler *Handler) Pages(req *Request) any {
	if req.Param("id") != "" {
		return handler.PageJSON(handler.OwnedPage(req, req.ID("id")), true)
	}
	db := handler.DB.Where("author_id = ?", req.User.ID)
	if strings.HasSuffix(req.R.URL.Path, "/search-by-tags") {
		tags, ok := req.Body()["tags"].([]any)
		if !ok || len(tags) == 0 {
			return []any{}
		}
		unique := map[string]bool{}
		for _, tag := range tags {
			unique[strings.ToLower(strings.TrimSpace(Str(tag)))] = true
		}
		for tag := range unique {
			db = db.Where("id IN (?)", handler.DB.Model(&models.PageTag{}).Select("page_id").Where("name = ?", tag))
		}
	}
	out := []any{}
	for _, page := range All[models.Page](db.Order("title")) {
		out = append(out, handler.PageJSON(page, false))
	}
	return out
}
func (handler *Handler) PageWrite(req *Request) any {
	page := models.Page{AuthorID: Ptr(req.User.ID), IsPrivate: true}
	creating := req.Param("id") == ""
	if !creating {
		page = handler.OwnedPage(req, req.ID("id"))
		if page.IsLocked {
			Fail(409, "Page is locked")
		}
	}
	if req.R.Method == "DELETE" {
		req.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "notes_page", page.ID) })
		return map[string]any{"message": "Page deleted"}
	}
	body := req.Body()
	if value, ok := body["title"]; ok {
		page.Title = strings.TrimSpace(Str(value))
	}
	if len([]rune(page.Title)) > 200 {
		Fail(400, "title too long")
	}
	if value, ok := body["parent_id"]; ok {
		page.ParentID = nil
		if value != nil {
			parent := handler.OwnedPage(req, Num(value))
			for ancestor := &parent; ancestor != nil; {
				if ancestor.ID == page.ID {
					Fail(400, "A page cannot contain itself")
				}
				if ancestor.ParentID == nil {
					break
				}
				page := handler.OwnedPage(req, *ancestor.ParentID)
				ancestor = &page
			}
			page.ParentID = Ptr(parent.ID)
		}
	}
	if value, ok := body["is_private"]; ok {
		page.IsPrivate = Boolean(value)
	}
	req.Tx(func(tx *gorm.DB) error { Check(tx.Save(&page).Error); return tree.Rebuild(tx, "notes_page") })
	if creating {
		req.Status = 201
	}
	return handler.PageJSON(page, false)
}
