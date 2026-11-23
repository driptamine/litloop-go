package posts

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func Register(server *Server) {
	server.Route("GET", "/{username}/posts", false, server.Posts)
	server.Route("POST", "/posts/create", true, server.PostWrite)
	server.Route("POST", "/posts/create_no_drf", true, server.PostWrite)
	server.Route("POST", "/posts/create/create_post_with_line_breaks", true, server.PostWrite)
	server.Route("POST", "/posts/create/v4", true, server.PostWrite)
	server.Route("POST", "/posts/create/postman", true, server.PostWrite)
	server.Route("POST", "/posts/create/post_with_videos", true, server.PostWrite)
	server.Route("POST", "/posts/create_post_api", true, server.PostWrite)
	server.Route("PUT,PATCH", "/posts/update_no_drf/{id:[0-9]+}", true, server.PostWrite)
	server.Route("DELETE", "/posts/delete_no_drf/{id:[0-9]+}", true, server.PostWrite)
	server.Route("GET", "/posts/list", false, server.Posts)
	server.Route("GET", "/posts/list_no_drf", false, server.Posts)
	server.Route("GET", "/posts/feed", false, server.Posts)
	server.Route("GET", "/posts/user/{username}", false, server.Posts)
	server.Route("GET", "/posts/me", true, server.Posts)
	server.Route("GET", "/posts/{id:[0-9]+}", false, func(req *Request) any {
		return server.PostJSON(Get[models.Post](server.DB, req.ID("id")),
			req.User)
	})
	server.Route("GET", "/posts/detail_no_drf/{id:[0-9]+}", false, func(req *Request) any {
		return server.PostJSON(Get[models.Post](server.DB, req.ID("id")),
			req.User)
	})
	server.Route("PUT,PATCH,DELETE", "/posts/{id:[0-9]+}", true, server.PostWrite)
	server.Route("POST", "/posts/{id:[0-9]+}/like", true, server.PostLike)
	server.Route("POST", "/posts/impressions/batch", true, server.Impressions)
	server.Route("GET", "/posts/{id:[0-9]+}/comments", false, server.Comments)
	server.Route("POST", "/posts/{id:[0-9]+}/comments/create", true, server.CommentCreate)
	server.Route("GET", "/posts/{id:[0-9]+}/comments/{comment:[0-9]+}", false, server.Comments)
}
