package users

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("POST", "/users/signup", false, s.Signup)
	s.Route("POST", "/users/signin", false, s.Signin)
	s.Route("POST", "/users/callback", false, s.Signin)
	s.Route("POST", "/users/token", false, s.OAuth)
	s.Route("POST", "/users/token-exchange", false, s.OAuth)
	s.Route("GET", "/users/me", true, func(q *Request) any { return q.User })
	s.Route("PUT", "/users/me", true, s.UpdateProfile)
	s.Route("POST", "/users/me/password", true, s.ChangePassword)
	s.Route("POST", "/users/me/avatar", true, s.Attachment)
	s.Route("POST", "/users/me/avatar/gcs", true, s.Attachment)
	s.Route("POST", "/users/me/avatar/r2", true, s.Attachment)
	s.Route("GET", "/users/list", false, s.UserList)
	s.Route("GET", "/users/search", false, s.UserList)
	s.Route("PUT", "/users/{id:[0-9]+}/role", true, s.Role)
	s.Route("POST", "/users/password-reset", false, s.ForgotPassword)
	s.Route("POST", "/users/password-reset-confirm/{uid}/{token}", false, s.ResetPassword)
	s.Route("GET", "/users/{key}", false, s.UserDetail)
	s.Route("GET", "/users/{username}/posts", false, func(q *Request) any { return s.UserMediaKind(q, "posts") })
	s.Route("GET", "/users/{username}/photos", true, func(q *Request) any { return s.UserMediaKind(q, "photos") })
	s.Route("GET", "/users/{username}/albums", true, func(q *Request) any { return s.UserMediaKind(q, "albums") })
	s.Route("GET", "/users/{username}/videos", true, func(q *Request) any { return s.UserMediaKind(q, "videos") })
	s.Route("GET", "/users/{username}/tracks", false, func(q *Request) any { return s.UserMediaKind(q, "tracks") })
	s.Route("GET", "/users/{username}/watchlist", true, func(q *Request) any { return s.UserMediaKind(q, "watchlist") })
}
