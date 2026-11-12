package notes

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"litloop/backend/internal/tree"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) Pages(q *Request) any {
	if q.Param("id") != "" {
		return s.PageJSON(s.OwnedPage(q, q.ID("id")), true)
	}
	db := s.DB.Where("author_id = ?", q.User.ID)
	if strings.HasSuffix(q.R.URL.Path, "/search-by-tags") {
		tags, ok := q.Body()["tags"].([]any)
		if !ok || len(tags) == 0 {
			return []any{}
		}
		unique := map[string]bool{}
		for _, t := range tags {
			unique[strings.ToLower(strings.TrimSpace(Str(t)))] = true
		}
		for t := range unique {
			db = db.Where("id IN (?)", s.DB.Model(&models.PageTag{}).Select("page_id").Where("name = ?", t))
		}
	}
	out := []any{}
	for _, p := range All[models.Page](db.Order("title")) {
		out = append(out, s.PageJSON(p, false))
	}
	return out
}
func (s *Handler) PageWrite(q *Request) any {
	p := models.Page{AuthorID: Ptr(q.User.ID), IsPrivate: true}
	creating := q.Param("id") == ""
	if !creating {
		p = s.OwnedPage(q, q.ID("id"))
		if p.IsLocked {
			Fail(409, "Page is locked")
		}
	}
	if q.R.Method == "DELETE" {
		q.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "notes_page", p.ID) })
		return map[string]any{"message": "Page deleted"}
	}
	b := q.Body()
	if v, ok := b["title"]; ok {
		p.Title = strings.TrimSpace(Str(v))
	}
	if len([]rune(p.Title)) > 200 {
		Fail(400, "title too long")
	}
	if v, ok := b["parent_id"]; ok {
		p.ParentID = nil
		if v != nil {
			parent := s.OwnedPage(q, Num(v))
			for ancestor := &parent; ancestor != nil; {
				if ancestor.ID == p.ID {
					Fail(400, "A page cannot contain itself")
				}
				if ancestor.ParentID == nil {
					break
				}
				x := s.OwnedPage(q, *ancestor.ParentID)
				ancestor = &x
			}
			p.ParentID = Ptr(parent.ID)
		}
	}
	if v, ok := b["is_private"]; ok {
		p.IsPrivate = Boolean(v)
	}
	q.Tx(func(tx *gorm.DB) error { Check(tx.Save(&p).Error); return tree.Rebuild(tx, "notes_page") })
	if creating {
		q.Status = 201
	}
	return s.PageJSON(p, false)
}
