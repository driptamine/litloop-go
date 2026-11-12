package todos

import (
	"litloop/backend/internal/models"
)

func (s *Handler) TodoJSON(t models.Todo) any {
	return map[string]any{"id": t.ID, "title": t.Title, "completed": t.Completed, "todo_list": t.TodoListID, "order": t.Order}
}
