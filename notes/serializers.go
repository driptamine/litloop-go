package notes

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) PageJSON(page models.Page, detail bool) map[string]any {
	out := Object(page)
	tags := []string{}
	tagIDs := []int64{}
	for _, tag := range All[models.PageTag](handler.DB.Where("page_id = ?", page.ID).Order("name")) {
		tags = append(tags, tag.Name)
		tagIDs = append(tagIDs, tag.ID)
	}
	out["tags"] = tags
	out["tag_ids"] = tagIDs
	if detail {
		blocks := []any{}
		for _, block := range All[models.Block](handler.DB.Where("page_id = ?", page.ID).Order("\"order\", id")) {
			blocks = append(blocks, handler.BlockJSON(block))
		}
		out["blocks"] = blocks
	}
	return out
}
func (handler *Handler) BlockJSON(block models.Block) map[string]any {
	out := Object(block)
	if block.Type == "table" {
		var table models.BlockTable
		if handler.DB.First(&table, "block_id = ?", block.ID).Error == nil {
			out["table_data"] = map[string]any{"columns": table.Columns, "rows": table.Rows}
		}
	}
	return out
}
