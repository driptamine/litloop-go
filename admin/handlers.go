package admin

import (
	"encoding/json"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) Admin(q *Request) any {
	q.Admin()
	registry := map[string]any{}
	names := []string{}
	for _, m := range models.All() {
		name := m.(interface{ TableName() string }).TableName()
		registry[name] = m
		names = append(names, name)
	}
	table := q.Param("table")
	if table == "" {
		return map[string]any{"tables": names}
	}
	model, ok := registry[table]
	if !ok {
		Fail(404, "Unknown table")
	}
	typ := reflect.TypeOf(model).Elem()
	dst := reflect.New(typ).Interface()
	id := q.Param("id")
	if q.R.Method == "GET" {
		if id != "" {
			Check(s.DB.First(dst, "id = ?", id).Error)
			return dst
		}
		list := reflect.New(reflect.SliceOf(typ)).Interface()
		return q.Page(s.DB.Model(model).Order("id"), list, func(v any) any { return v })
	}
	if q.R.Method == "DELETE" {
		if id == "" {
			Fail(400, "ID required")
		}
		q.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, table, Num(id)) })
		q.Status = 204
		return nil
	}
	if table == "users_user" || strings.HasPrefix(table, "go_") {
		Fail(400, "Use the dedicated API for users and internal records")
	}
	if id != "" {
		Check(s.DB.First(dst, "id = ?", id).Error)
	}
	b := q.Body()
	delete(b, "id")
	encoded, e := json.Marshal(b)
	Check(e)
	Check(json.Unmarshal(encoded, dst))
	Check(s.DB.Save(dst).Error)
	if id == "" {
		q.Status = 201
	}
	return dst
}
