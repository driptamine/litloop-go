package todos

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (handler *Handler) Todos(req *Request) any {
	db := handler.DB.Where("user_id = ?", req.User.ID)
	if req.Param("id") != "" {
		return handler.TodoJSON(Get[models.Todo](db, req.ID("id")))
	}
	if id := req.R.URL.Query().Get("list_id"); id != "" {
		db = db.Where("todo_list_id = ?", id)
	}
	out := []any{}
	for _, todo := range All[models.Todo](db.Order("\"order\", id")) {
		out = append(out, handler.TodoJSON(todo))
	}
	var profile models.Profile
	_ = handler.DB.First(&profile, "user_id = ?", req.User.ID).Error
	return map[string]any{"todos": out, "background_image_url": profile.BackgroundImage}
}
func (handler *Handler) TodoWrite(req *Request) any {
	todo := models.Todo{UserID: Ptr(req.User.ID)}
	creating := req.Param("id") == ""
	if !creating {
		todo = Get[models.Todo](handler.DB.Where("user_id = ?", req.User.ID), req.ID("id"))
	}
	if req.R.Method == "DELETE" {
		Check(handler.DB.Delete(&todo).Error)
		return map[string]any{"message": "Todo deleted"}
	}
	body := req.Body()
	if value, ok := body["title"]; ok {
		todo.Title = Str(value)
	}
	if len([]rune(todo.Title)) > 255 {
		Fail(400, "title too long")
	}
	if value, ok := body["completed"]; ok {
		todo.Completed = Boolean(value)
	}
	if value, ok := body["order"]; ok {
		todo.Order = Num(value)
	}
	if value, ok := body["todo_list"]; ok {
		todo.TodoListID = nil
		if value != nil {
			list := Get[models.TodoList](handler.DB.Where("user_id = ?", req.User.ID), Num(value))
			todo.TodoListID = Ptr(list.ID)
		}
	}
	Check(handler.DB.Save(&todo).Error)
	if creating {
		req.Status = 201
	}
	return handler.TodoJSON(todo)
}
func (handler *Handler) TodoLists(req *Request) any {
	out := All[models.TodoList](handler.DB.Where("user_id = ?", req.User.ID).Order("\"order\", id"))
	return out
}
func (handler *Handler) TodoListWrite(req *Request) any {
	list := models.TodoList{UserID: req.User.ID}
	creating := req.Param("id") == ""
	if !creating {
		list = Get[models.TodoList](handler.DB.Where("user_id = ?", req.User.ID), req.ID("id"))
	}
	if req.R.Method == "DELETE" {
		req.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "todos_todolist", list.ID) })
		return map[string]any{"message": "List deleted"}
	}
	body := req.Body()
	if value, ok := body["name"]; ok {
		list.Name = strings.TrimSpace(Str(value))
	}
	if list.Name == "" || len([]rune(list.Name)) > 255 {
		Fail(400, "name must contain 1–255 characters")
	}
	if value, ok := body["order"]; ok {
		list.Order = Num(value)
	}
	Check(handler.DB.Save(&list).Error)
	if creating {
		req.Status = 201
	}
	return list
}
func (handler *Handler) Reorder(req *Request) any {
	items, ok := req.Body()["items"].([]any)
	if !ok || len(items) > 1000 {
		Fail(400, "Expected at most 1000 items")
	}
	table := "todos_todo"
	if strings.Contains(req.R.URL.Path, "/lists/") {
		table = "todos_todolist"
	}
	req.Tx(func(tx *gorm.DB) error {
		for _, item := range items {
			entry, ok := item.(map[string]any)
			if !ok {
				Fail(400, "Invalid item")
			}
			query := tx.Table(table).Where("id = ? AND user_id = ?", Num(entry["id"]), req.User.ID).Update("order", Num(entry["order"]))
			Check(query.Error)
			if query.RowsAffected == 0 {
				Fail(404, "Item not found")
			}
		}
		return nil
	})
	return map[string]any{"message": "Reordered"}
}
