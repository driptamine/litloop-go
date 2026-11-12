package notes

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) PageJSON(p models.Page, detail bool) map[string]any {
	o := Object(p)
	tags := []string{}
	tagIDs := []int64{}
	for _, t := range All[models.PageTag](s.DB.Where("page_id = ?", p.ID).Order("name")) {
		tags = append(tags, t.Name)
		tagIDs = append(tagIDs, t.ID)
	}
	o["tags"] = tags
	o["tag_ids"] = tagIDs
	if detail {
		blocks := []any{}
		for _, b := range All[models.Block](s.DB.Where("page_id = ?", p.ID).Order("\"order\", id")) {
			blocks = append(blocks, s.BlockJSON(b))
		}
		o["blocks"] = blocks
	}
	return o
}
func (s *Handler) BlockJSON(b models.Block) map[string]any {
	o := Object(b)
	if b.Type == "table" {
		var t models.BlockTable
		if s.DB.First(&t, "block_id = ?", b.ID).Error == nil {
			o["table_data"] = map[string]any{"columns": t.Columns, "rows": t.Rows}
		}
	}
	return o
}
