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
	for _, relation := range models.Relations {
		if relation.Target != table {
			continue
		}
		if relation.OnDelete == "SET NULL" {
			if err := tx.Table(relation.Table).Where(relation.Column+" = ?", id).Update(relation.Column, nil).Error; err != nil {
				return err
			}
		} else {
			var children []int64
			if err := tx.Table(relation.Table).Where(relation.Column+" = ?", id).Pluck("id", &children).Error; err != nil {
				return err
			}
			for _, child := range children {
				if err := deleteTree(tx, relation.Table, child, seen); err != nil {
					return err
				}
			}
		}
	}
	return tx.Table(table).Where("id = ?", id).Delete(&map[string]any{}).Error
}
