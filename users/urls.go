package users

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("POST", "/users/signup", false, server.Signup)
	server.Route("POST", "/users/signin", false, server.Signin)
	server.Route("POST", "/users/callback", false, server.Signin)
	server.Route("POST", "/users/token", false, server.OAuth)
	server.Route("POST", "/users/token-exchange", false, server.OAuth)
	server.Route("GET", "/users/me", true, func(req *Request) any { return req.User })
	server.Route("PUT", "/users/me", true, server.UpdateProfile)
	server.Route("POST", "/users/me/password", true, server.ChangePassword)
	server.Route("POST", "/users/me/avatar", true, server.Attachment)
	server.Route("POST", "/users/me/avatar/gcs", true, server.Attachment)
	server.Route("POST", "/users/me/avatar/r2", true, server.Attachment)
	server.Route("GET", "/users/list", false, server.UserList)
	server.Route("GET", "/users/search", false, server.UserList)
	server.Route("PUT", "/users/{id:[0-9]+}/role", true, server.Role)
	server.Route("POST", "/users/password-reset", false, server.ForgotPassword)
	server.Route("POST", "/users/password-reset-confirm/{uid}/{token}", false, server.ResetPassword)
	server.Route("GET", "/users/{key}", false, server.UserDetail)
	server.Route("GET", "/users/{username}/posts", false, func(req *Request) any { return server.UserMediaKind(req, "posts") })
	server.Route("GET", "/users/{username}/photos", true, func(req *Request) any { return server.UserMediaKind(req, "photos") })
	server.Route("GET", "/users/{username}/albums", true, func(req *Request) any { return server.UserMediaKind(req, "albums") })
	server.Route("GET", "/users/{username}/videos", true, func(req *Request) any { return server.UserMediaKind(req, "videos") })
	server.Route("GET", "/users/{username}/tracks", false, func(req *Request) any { return server.UserMediaKind(req, "tracks") })
	server.Route("GET", "/users/{username}/watchlist", true, func(req *Request) any { return server.UserMediaKind(req, "watchlist") })
}
