package links

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) LinkSave(q *Request) any {
	var l models.Link
	q.Tx(func(tx *gorm.DB) error { l = s.SaveLink(tx, q.User.ID, q.Body()); return nil })
	q.Status = 201
	return map[string]any{"success": true, "link_id": l.ID}
}
func (s *Handler) Links(q *Request) any {
	db := s.DB.Model(&models.Link{}).Where("user_id = ?", q.User.ID)
	term := q.R.URL.Query().Get("q")
	if term != "" {
		like := "%" + strings.ToLower(term) + "%"
		if strings.HasSuffix(q.R.URL.Path, "/search") {
			db = db.Where("id IN (?)", s.DB.Model(&models.LinkTag{}).Select("link_id").Where("hashtag_id IN (?)", s.DB.Model(&models.Hashtag{}).Select("id").Where("LOWER(name) LIKE ?", like)))
		} else {
			db = db.Where("LOWER(url) LIKE ? OR LOWER(title) LIKE ?", like, like)
		}
	}
	page := q.Page(db.Order("created_at DESC"), &[]models.Link{}, func(v any) any {
		out := []any{}
		for _, l := range *v.(*[]models.Link) {
			out = append(out, s.LinkJSON(l))
		}
		return out
	}).(map[string]any)
	return map[string]any{"results": page["items"], "query": term, "page": page["current_page"], "page_size": q.R.URL.Query().Get("page_size"), "total_pages": page["num_pages"], "total_items": page["total"]}
}
