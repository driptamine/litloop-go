package todos

import (
	"litloop/backend/internal/models"
)

func (handler *Handler) TodoJSON(todo models.Todo) any {
	return map[string]any{"id": todo.ID, "title": todo.Title, "completed": todo.Completed, "todo_list": todo.TodoListID, "order": todo.Order}
}
