package todos

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/todos/all", true, server.Todos)
	server.Route("GET", "/todos/{id:[0-9]+}", true, server.Todos)
	server.Route("POST", "/todos/create", true, server.TodoWrite)
	server.Route("PUT,PATCH", "/todos/{id:[0-9]+}/update", true, server.TodoWrite)
	server.Route("DELETE", "/todos/{id:[0-9]+}/delete", true, server.TodoWrite)
	server.Route("POST", "/todos/reorder", true, server.Reorder)
	server.Route("POST", "/todos/background-image", true, server.Attachment)
	server.Route("GET", "/todos/lists", true, server.TodoLists)
	server.Route("POST", "/todos/lists/create", true, server.TodoListWrite)
	server.Route("PUT,PATCH", "/todos/lists/{id:[0-9]+}/update", true, server.TodoListWrite)
	server.Route("DELETE", "/todos/lists/{id:[0-9]+}/delete", true, server.TodoListWrite)
	server.Route("POST", "/todos/lists/reorder", true, server.Reorder)
}
