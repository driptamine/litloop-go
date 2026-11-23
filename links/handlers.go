package links

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (handler *Handler) LinkSave(req *Request) any {
	var link models.Link
	req.Tx(func(tx *gorm.DB) error { link = handler.SaveLink(tx, req.User.ID, req.Body()); return nil })
	req.Status = 201
	return map[string]any{"success": true, "link_id": link.ID}
}
func (handler *Handler) Links(req *Request) any {
	db := handler.DB.Model(&models.Link{}).Where("user_id = ?", req.User.ID)
	term := req.R.URL.Query().Get("q")
	if term != "" {
		like := "%" + strings.ToLower(term) + "%"
		if strings.HasSuffix(req.R.URL.Path, "/search") {
			db = db.Where("id IN (?)", handler.DB.Model(&models.LinkTag{}).Select("link_id").Where("hashtag_id IN (?)", handler.DB.Model(&models.Hashtag{}).Select("id").Where("LOWER(name) LIKE ?", like)))
		} else {
			db = db.Where("LOWER(url) LIKE ? OR LOWER(title) LIKE ?", like, like)
		}
	}
	page := req.Page(db.Order("created_at DESC"), &[]models.Link{}, func(value any) any {
		out := []any{}
		for _, link := range *value.(*[]models.Link) {
			out = append(out, handler.LinkJSON(link))
		}
		return out
	}).(map[string]any)
	return map[string]any{"results": page["items"], "query": term, "page": page["current_page"], "page_size": req.R.URL.Query().Get("page_size"), "total_pages": page["num_pages"], "total_items": page["total"]}
}
