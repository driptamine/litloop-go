package todos

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/todos/all", true, s.Todos)
	s.Route("GET", "/todos/{id:[0-9]+}", true, s.Todos)
	s.Route("POST", "/todos/create", true, s.TodoWrite)
	s.Route("PUT,PATCH", "/todos/{id:[0-9]+}/update", true, s.TodoWrite)
	s.Route("DELETE", "/todos/{id:[0-9]+}/delete", true, s.TodoWrite)
	s.Route("POST", "/todos/reorder", true, s.Reorder)
	s.Route("POST", "/todos/background-image", true, s.Attachment)
	s.Route("GET", "/todos/lists", true, s.TodoLists)
	s.Route("POST", "/todos/lists/create", true, s.TodoListWrite)
	s.Route("PUT,PATCH", "/todos/lists/{id:[0-9]+}/update", true, s.TodoListWrite)
	s.Route("DELETE", "/todos/lists/{id:[0-9]+}/delete", true, s.TodoListWrite)
	s.Route("POST", "/todos/lists/reorder", true, s.Reorder)
}
