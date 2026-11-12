package notes

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) BlockWrite(q *Request) any {
	var b models.Block
	creating := q.Param("block") == ""
	if creating {
		p := s.OwnedPage(q, q.ID("id"))
		if p.IsLocked {
			Fail(409, "Page is locked")
		}
		b = models.Block{PageID: p.ID, Type: "text"}
	} else {
		b = Get[models.Block](s.DB, q.ID("block"))
		p := s.OwnedPage(q, b.PageID)
		if p.IsLocked {
			Fail(409, "Page is locked")
		}
	}
	if q.R.Method == "DELETE" {
		q.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "notes_block", b.ID) })
		return map[string]any{"message": "Block deleted"}
	}
	data := q.Body()
	tableOnly := strings.HasSuffix(q.R.URL.Path, "/table")
	if tableOnly && b.Type != "table" {
		Fail(400, "Not a table block")
	}
	if v, ok := data["type"]; ok {
		b.Type = Str(v)
	}
	if b.Type != "text" && b.Type != "table" {
		Fail(400, "Invalid block type")
	}
	if v, ok := data["content"]; ok {
		b.Content = Str(v)
	}
	if v, ok := data["order"]; ok {
		b.Order = Num(v)
		if b.Order < 0 {
			Fail(400, "order must be positive")
		}
	}
	q.Tx(func(tx *gorm.DB) error {
		Check(tx.Save(&b).Error)
		if b.Type == "table" {
			td := data
			if !tableOnly {
				if nested, ok := data["table_data"].(map[string]any); ok {
					td = nested
				} else {
					td = map[string]any{}
				}
			}
			var t models.BlockTable
			_ = tx.First(&t, "block_id = ?", b.ID).Error
			t.BlockID = b.ID
			if v, ok := td["columns"]; ok {
				t.Columns = models.AsJSON(v)
			}
			if v, ok := td["rows"]; ok {
				t.Rows = models.AsJSON(v)
			}
			return tx.Save(&t).Error
		}
		return nil
	})
	if creating {
		q.Status = 201
	}
	return s.BlockJSON(b)
}
