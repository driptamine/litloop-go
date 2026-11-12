package links

import (
	. "litloop/backend/internal/httpapi"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) ImportLinks(q *Request) any {
	rows := CSVRows(q)
	count := 0
	q.Tx(func(tx *gorm.DB) error {
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
			for _, t := range strings.FieldsFunc(tags, func(r rune) bool { return r == ',' || r == ';' || r == ' ' || r == '|' }) {
				tagList = append(tagList, t)
			}
			s.SaveLink(tx, q.User.ID, map[string]any{"url": uri, "title": row["title"], "tags": tagList, "status": row["status"], "time_added": row["time_added"]})
			count++
		}
		return nil
	})
	return map[string]any{"success": true, "imported": count}
}
