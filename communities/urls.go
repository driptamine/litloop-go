package communities

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/communities", false, s.Communities)
	s.Route("POST", "/communities/create", true, s.CommunityWrite)
	s.Route("GET", "/communities/@{handle}", false, func(q *Request) any { return s.CommunityJSON(s.Community(q), q.User) })
	s.Route("GET", "/communities/{id:[0-9]+}", false, func(q *Request) any { return s.CommunityJSON(s.Community(q), q.User) })
	s.Route("PUT,PATCH", "/communities/{id:[0-9]+}/update", true, s.CommunityWrite)
	s.Route("DELETE", "/communities/{id:[0-9]+}/delete", true, s.CommunityWrite)
	s.Route("POST", "/communities/{id:[0-9]+}/join", true, s.CommunityJoin)
	s.Route("POST", "/communities/{id:[0-9]+}/leave", true, s.CommunityJoin)
	s.Route("POST", "/communities/{id:[0-9]+}/icon/r2", true, s.Attachment)
	s.Route("POST", "/communities/{id:[0-9]+}/banner/r2", true, s.Attachment)
	s.Route("GET", "/communities/{id:[0-9]+}/posts", false, s.CommunityPosts)
	s.Route("GET", "/communities/{id:[0-9]+}/posts/pending", true, s.CommunityPosts)
	s.Route("POST", "/communities/{id:[0-9]+}/posts/request", true, s.CommunityPosts)
	s.Route("POST", "/communities/{id:[0-9]+}/posts/{cp:[0-9]+}/approve", true, s.CommunityPosts)
	s.Route("POST", "/communities/{id:[0-9]+}/posts/{cp:[0-9]+}/reject", true, s.CommunityPosts)
	s.Route("DELETE", "/communities/{id:[0-9]+}/posts/{cp:[0-9]+}", true, s.CommunityPosts)
}
