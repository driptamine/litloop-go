package admin

import (
	"encoding/json"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"reflect"
	"strings"

	"gorm.io/gorm"
)

func (handler *Handler) Admin(req *Request) any {
	req.Admin()
	registry := map[string]any{}
	names := []string{}
	for _, entry := range models.All() {
		name := entry.(interface{ TableName() string }).TableName()
		registry[name] = entry
		names = append(names, name)
	}
	table := req.Param("table")
	if table == "" {
		return map[string]any{"tables": names}
	}
	model, ok := registry[table]
	if !ok {
		Fail(404, "Unknown table")
	}
	typ := reflect.TypeOf(model).Elem()
	dst := reflect.New(typ).Interface()
	id := req.Param("id")
	if req.R.Method == "GET" {
		if id != "" {
			Check(handler.DB.First(dst, "id = ?", id).Error)
			return dst
		}
		list := reflect.New(reflect.SliceOf(typ)).Interface()
		return req.Page(handler.DB.Model(model).Order("id"), list, func(value any) any { return value })
	}
	if req.R.Method == "DELETE" {
		if id == "" {
			Fail(400, "ID required")
		}
		req.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, table, Num(id)) })
		req.Status = 204
		return nil
	}
	if table == "users_user" || strings.HasPrefix(table, "go_") {
		Fail(400, "Use the dedicated API for users and internal records")
	}
	if id != "" {
		Check(handler.DB.First(dst, "id = ?", id).Error)
	}
	body := req.Body()
	delete(body, "id")
	encoded, err := json.Marshal(body)
	Check(err)
	Check(json.Unmarshal(encoded, dst))
	Check(handler.DB.Save(dst).Error)
	if id == "" {
		req.Status = 201
	}
	return dst
}
