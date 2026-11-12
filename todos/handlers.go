package todos

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) Todos(q *Request) any {
	db := s.DB.Where("user_id = ?", q.User.ID)
	if q.Param("id") != "" {
		return s.TodoJSON(Get[models.Todo](db, q.ID("id")))
	}
	if id := q.R.URL.Query().Get("list_id"); id != "" {
		db = db.Where("todo_list_id = ?", id)
	}
	out := []any{}
	for _, t := range All[models.Todo](db.Order("\"order\", id")) {
		out = append(out, s.TodoJSON(t))
	}
	var p models.Profile
	_ = s.DB.First(&p, "user_id = ?", q.User.ID).Error
	return map[string]any{"todos": out, "background_image_url": p.BackgroundImage}
}
func (s *Handler) TodoWrite(q *Request) any {
	t := models.Todo{UserID: Ptr(q.User.ID)}
	creating := q.Param("id") == ""
	if !creating {
		t = Get[models.Todo](s.DB.Where("user_id = ?", q.User.ID), q.ID("id"))
	}
	if q.R.Method == "DELETE" {
		Check(s.DB.Delete(&t).Error)
		return map[string]any{"message": "Todo deleted"}
	}
	b := q.Body()
	if v, ok := b["title"]; ok {
		t.Title = Str(v)
	}
	if len([]rune(t.Title)) > 255 {
		Fail(400, "title too long")
	}
	if v, ok := b["completed"]; ok {
		t.Completed = Boolean(v)
	}
	if v, ok := b["order"]; ok {
		t.Order = Num(v)
	}
	if v, ok := b["todo_list"]; ok {
		t.TodoListID = nil
		if v != nil {
			list := Get[models.TodoList](s.DB.Where("user_id = ?", q.User.ID), Num(v))
			t.TodoListID = Ptr(list.ID)
		}
	}
	Check(s.DB.Save(&t).Error)
	if creating {
		q.Status = 201
	}
	return s.TodoJSON(t)
}
func (s *Handler) TodoLists(q *Request) any {
	out := All[models.TodoList](s.DB.Where("user_id = ?", q.User.ID).Order("\"order\", id"))
	return out
}
func (s *Handler) TodoListWrite(q *Request) any {
	list := models.TodoList{UserID: q.User.ID}
	creating := q.Param("id") == ""
	if !creating {
		list = Get[models.TodoList](s.DB.Where("user_id = ?", q.User.ID), q.ID("id"))
	}
	if q.R.Method == "DELETE" {
		q.Tx(func(tx *gorm.DB) error { return DeleteRecord(tx, "todos_todolist", list.ID) })
		return map[string]any{"message": "List deleted"}
	}
	b := q.Body()
	if v, ok := b["name"]; ok {
		list.Name = strings.TrimSpace(Str(v))
	}
	if list.Name == "" || len([]rune(list.Name)) > 255 {
		Fail(400, "name must contain 1–255 characters")
	}
	if v, ok := b["order"]; ok {
		list.Order = Num(v)
	}
	Check(s.DB.Save(&list).Error)
	if creating {
		q.Status = 201
	}
	return list
}
func (s *Handler) Reorder(q *Request) any {
	items, ok := q.Body()["items"].([]any)
	if !ok || len(items) > 1000 {
		Fail(400, "Expected at most 1000 items")
	}
	table := "todos_todo"
	if strings.Contains(q.R.URL.Path, "/lists/") {
		table = "todos_todolist"
	}
	q.Tx(func(tx *gorm.DB) error {
		for _, v := range items {
			o, ok := v.(map[string]any)
			if !ok {
				Fail(400, "Invalid item")
			}
			r := tx.Table(table).Where("id = ? AND user_id = ?", Num(o["id"]), q.User.ID).Update("order", Num(o["order"]))
			Check(r.Error)
			if r.RowsAffected == 0 {
				Fail(404, "Item not found")
			}
		}
		return nil
	})
	return map[string]any{"message": "Reordered"}
}
