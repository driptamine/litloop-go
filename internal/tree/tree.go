// Package tree maintains Django MPTT indexes while handlers use parent IDs.
package tree

import (
	"fmt"
	"gorm.io/gorm"
)

type node struct {
	ID       int64
	ParentID *int64
}

func Rebuild(tx *gorm.DB, table string) error {
	order := "add_date, id"
	lock := int64(2101)
	switch table {
	case "notes_page":
		order = "title, id"
		lock = 2102
	case "comments_comment":
	default:
		return fmt.Errorf("unsupported tree table %q", table)
	}
	if tx.Dialector.Name() == "postgres" {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", lock).Error; err != nil {
			return err
		}
	}
	rows := []node{}
	if err := tx.Table(table).Select("id, parent_id").Order(order).Find(&rows).Error; err != nil {
		return err
	}
	children := map[int64][]int64{}
	known := map[int64]bool{}
	for _, n := range rows {
		known[n.ID] = true
	}
	for _, n := range rows {
		parent := int64(0)
		if n.ParentID != nil {
			parent = *n.ParentID
			if !known[parent] {
				return fmt.Errorf("orphan tree node %d", n.ID)
			}
		}
		children[parent] = append(children[parent], n.ID)
	}
	visited := map[int64]bool{}
	var visit func(int64, int64, int64, *int64) error
	visit = func(id, treeID, level int64, counter *int64) error {
		if visited[id] {
			return fmt.Errorf("tree cycle at %d", id)
		}
		visited[id] = true
		left := *counter
		*counter++
		for _, child := range children[id] {
			if err := visit(child, treeID, level+1, counter); err != nil {
				return err
			}
		}
		right := *counter
		*counter++
		return tx.Table(table).Where("id = ?", id).Updates(map[string]any{"lft": left, "rght": right, "tree_id": treeID, "level": level}).Error
	}
	for i, id := range children[0] {
		counter := int64(1)
		if err := visit(id, int64(i+1), 0, &counter); err != nil {
			return err
		}
	}
	if len(visited) != len(rows) {
		return fmt.Errorf("tree contains a cycle without a root")
	}
	return nil
}
