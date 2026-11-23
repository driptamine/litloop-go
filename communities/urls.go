package communities

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/communities", false, server.Communities)
	server.Route("POST", "/communities/create", true, server.CommunityWrite)
	server.Route("GET", "/communities/@{handle}", false, func(req *Request) any { return server.CommunityJSON(server.Community(req), req.User) })
	server.Route("GET", "/communities/{id:[0-9]+}", false, func(req *Request) any { return server.CommunityJSON(server.Community(req), req.User) })
	server.Route("PUT,PATCH", "/communities/{id:[0-9]+}/update", true, server.CommunityWrite)
	server.Route("DELETE", "/communities/{id:[0-9]+}/delete", true, server.CommunityWrite)
	server.Route("POST", "/communities/{id:[0-9]+}/join", true, server.CommunityJoin)
	server.Route("POST", "/communities/{id:[0-9]+}/leave", true, server.CommunityJoin)
	server.Route("POST", "/communities/{id:[0-9]+}/icon/r2", true, server.Attachment)
	server.Route("POST", "/communities/{id:[0-9]+}/banner/r2", true, server.Attachment)
	server.Route("GET", "/communities/{id:[0-9]+}/posts", false, server.CommunityPosts)
	server.Route("GET", "/communities/{id:[0-9]+}/posts/pending", true, server.CommunityPosts)
	server.Route("POST", "/communities/{id:[0-9]+}/posts/request", true, server.CommunityPosts)
	server.Route("POST", "/communities/{id:[0-9]+}/posts/{cp:[0-9]+}/approve", true, server.CommunityPosts)
	server.Route("POST", "/communities/{id:[0-9]+}/posts/{cp:[0-9]+}/reject", true, server.CommunityPosts)
	server.Route("DELETE", "/communities/{id:[0-9]+}/posts/{cp:[0-9]+}", true, server.CommunityPosts)
}
