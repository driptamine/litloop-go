package httpapi

import (
	"fmt"
	"litloop/backend/internal/models"
	"litloop/backend/internal/tree"

	"gorm.io/gorm"
)

// Django handles cascading deletion in its ORM; mirror it for both SQL drivers.
func DeleteRecord(tx *gorm.DB, table string, id int64) error {
	if err := deleteTree(tx, table, id, map[string]bool{}); err != nil {
		return err
	}
	if table == "notes_page" || table == "comments_comment" {
		return tree.Rebuild(tx, table)
	}
	return nil
}
func deleteTree(tx *gorm.DB, table string, id int64, seen map[string]bool) error {
	key := fmt.Sprintf("%s:%d", table, id)
	if seen[key] {
		return nil
	}
	seen[key] = true
	for _, r := range models.Relations {
		if r.Target != table {
			continue
		}
		if r.OnDelete == "SET NULL" {
			if e := tx.Table(r.Table).Where(r.Column+" = ?", id).Update(r.Column, nil).Error; e != nil {
				return e
			}
		} else {
			var children []int64
			if e := tx.Table(r.Table).Where(r.Column+" = ?", id).Pluck("id", &children).Error; e != nil {
				return e
			}
			for _, child := range children {
				if e := deleteTree(tx, r.Table, child, seen); e != nil {
					return e
				}
			}
		}
	}
	return tx.Table(table).Where("id = ?", id).Delete(&map[string]any{}).Error
}
