package posts

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func Register(s *Server) {
	s.Route("GET", "/{username}/posts", false, s.Posts)
	s.Route("POST", "/posts/create", true, s.PostWrite)
	s.Route("POST", "/posts/create_no_drf", true, s.PostWrite)
	s.Route("POST", "/posts/create/create_post_with_line_breaks", true, s.PostWrite)
	s.Route("POST", "/posts/create/v4", true, s.PostWrite)
	s.Route("POST", "/posts/create/postman", true, s.PostWrite)
	s.Route("POST", "/posts/create/post_with_videos", true, s.PostWrite)
	s.Route("POST", "/posts/create_post_api", true, s.PostWrite)
	s.Route("PUT,PATCH", "/posts/update_no_drf/{id:[0-9]+}", true, s.PostWrite)
	s.Route("DELETE", "/posts/delete_no_drf/{id:[0-9]+}", true, s.PostWrite)
	s.Route("GET", "/posts/list", false, s.Posts)
	s.Route("GET", "/posts/list_no_drf", false, s.Posts)
	s.Route("GET", "/posts/feed", false, s.Posts)
	s.Route("GET", "/posts/user/{username}", false, s.Posts)
	s.Route("GET", "/posts/me", true, s.Posts)
	s.Route("GET", "/posts/{id:[0-9]+}", false, func(q *Request) any {
		return s.PostJSON(Get[models.Post](s.DB, q.ID("id")),
			q.User)
	})
	s.Route("GET", "/posts/detail_no_drf/{id:[0-9]+}", false, func(q *Request) any {
		return s.PostJSON(Get[models.Post](s.DB, q.ID("id")),
			q.User)
	})
	s.Route("PUT,PATCH,DELETE", "/posts/{id:[0-9]+}", true, s.PostWrite)
	s.Route("POST", "/posts/{id:[0-9]+}/like", true, s.PostLike)
	s.Route("POST", "/posts/impressions/batch", true, s.Impressions)
	s.Route("GET", "/posts/{id:[0-9]+}/comments", false, s.Comments)
	s.Route("POST", "/posts/{id:[0-9]+}/comments/create", true, s.CommentCreate)
	s.Route("GET", "/posts/{id:[0-9]+}/comments/{comment:[0-9]+}", false, s.Comments)
}
