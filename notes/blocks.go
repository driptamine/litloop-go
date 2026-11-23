package notes

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (handler *Handler) BlockWrite(req *Request) any {
	var block models.Block
	creating := req.Param("block") == ""
	if creating {
		page := handler.OwnedPage(req, req.ID("id"))
		if page.IsLocked {
			Fail(409, "Page is locked")
		}
		block = models.Block{PageID: page.ID, Type: "text"}
	} else {
		block = Get[models.Block](handler.DB, req.ID("block"))
		page := handler.OwnedPage(req, block.PageID)
		if page.IsLocked {
			Fail(409, "Page is locked")
		}
	}
	if req.R.Method == "DELETE" {
		req.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "notes_block", block.ID) })
		return map[string]any{"message": "Block deleted"}
	}
	data := req.Body()
	tableOnly := strings.HasSuffix(req.R.URL.Path, "/table")
	if tableOnly && block.Type != "table" {
		Fail(400, "Not a table block")
	}
	if value, ok := data["type"]; ok {
		block.Type = Str(value)
	}
	if block.Type != "text" && block.Type != "table" {
		Fail(400, "Invalid block type")
	}
	if value, ok := data["content"]; ok {
		block.Content = Str(value)
	}
	if value, ok := data["order"]; ok {
		block.Order = Num(value)
		if block.Order < 0 {
			Fail(400, "order must be positive")
		}
	}
	req.Tx(func(tx *gorm.DB) error {
		Check(tx.Save(&block).Error)
		if block.Type == "table" {
			td := data
			if !tableOnly {
				if nested, ok := data["table_data"].(map[string]any); ok {
					td = nested
				} else {
					td = map[string]any{}
				}
			}
			var table models.BlockTable
			_ = tx.First(&table, "block_id = ?", block.ID).Error
			table.BlockID = block.ID
			if value, ok := td["columns"]; ok {
				table.Columns = models.AsJSON(value)
			}
			if value, ok := td["rows"]; ok {
				table.Rows = models.AsJSON(value)
			}
			return tx.Save(&table).Error
		}
		return nil
	})
	if creating {
		req.Status = 201
	}
	return handler.BlockJSON(block)
}
