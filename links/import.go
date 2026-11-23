package links

import (
	. "litloop/backend/internal/httpapi"
	"strings"

	"gorm.io/gorm"
)

func (handler *Handler) ImportLinks(req *Request) any {
	rows := CSVRows(req)
	count := 0
	req.Tx(func(tx *gorm.DB) error {
		for _, row := range rows {
			uri := row["url"]
			if uri == "" {
				uri = row["URL"]
			}
			if uri == "" {
				continue
			}
			tags := row["tags"]
			if tags == "" {
				tags = row["hashtags"]
			}
			tagList := []any{}
			for _, tag := range strings.FieldsFunc(tags, func(ch rune) bool { return ch == ',' || ch == ';' || ch == ' ' || ch == '|' }) {
				tagList = append(tagList, tag)
			}
			handler.SaveLink(tx, req.User.ID, map[string]any{"url": uri, "title": row["title"], "tags": tagList, "status": row["status"], "time_added": row["time_added"]})
			count++
		}
		return nil
	})
	return map[string]any{"success": true, "imported": count}
}
