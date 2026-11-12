package admin

import (
	"litloop/backend/internal/database"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) Maintenance(q *Request) any {
	q.Admin()
	path := q.R.URL.Path
	if strings.Contains(path, "regenerate") {
		Check(database.Migrate(s.DB))
		return map[string]any{"status": "migrated"}
	}
	q.Tx(func(tx *gorm.DB) error {
		for _, m := range models.All() {
			table := m.(interface{ TableName() string }).TableName()
			if strings.Contains(path, "user-tables") && !strings.HasPrefix(table, "users_") {
				continue
			}
			if strings.Contains(path, "chat-tables") && !strings.HasPrefix(table, "chats_") {
				continue
			}
			var IDs []int64
			if strings.HasPrefix(table, "go_") {
				continue
			}
			Check(tx.Table(table).Pluck("id", &IDs).Error)
			for _, id := range IDs {
				Check(DeleteRecord(tx, table, id))
			}
		}
		return nil
	})
	return map[string]any{"status": "deleted"}
}
