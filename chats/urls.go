package chats

import (
	. "litloop/backend/internal/httpapi"
	"strings"
)

func Register(s *Server) {
	s.Route("GET", "/chats", true, s.ChatList)
	s.Route("GET", "/chats/me", true, s.ChatList)
	s.Route("GET", "/chats/direct", true, s.ChatList)
	s.Route("GET", "/chats/group", true, s.ChatList)
	s.Route("GET,POST", "/chats/direct/{user:[0-9]+}", true, s.DirectChat)
	s.Route("GET,POST", "/chats/saved", true, s.DirectChat)
	s.Route("POST", "/chats/group/create", true, s.CreateGroup)
	s.Route("GET", "/chats/group/{id:[0-9]+}", true, func(q *Request) any {
		c := s.ChatAccess(q, q.ID("id"))
		return s.ChatJSON(
			c, q.User.ID, true,
		)
	})
	s.Route("POST", "/chats/group/{id:[0-9]+}/add", true, s.GroupMember)
	s.Route("POST", "/chats/group/{id:[0-9]+}/remove", true, s.GroupMember)
	s.Route("POST", "/chats/{id:[0-9]+}/send", true, s.SendMessage)
	s.Route("GET", "/chats/{id:[0-9]+}/messages", true, s.MessagesAfter)
	s.Route("POST", "/chats/{id:[0-9]+}/call", true, s.CallCommand)
	s.Route("GET", "/chats/calls/incoming", true, s.IncomingCall)
	s.Route("POST", "/chats/{id:[0-9]+}/read", true, s.ChatAction)
	s.Route("POST,DELETE", "/chats/{id:[0-9]+}/delete", true, s.ChatAction)
	s.Route("POST", "/chats/upload", true, s.Attachment)
	s.Route("POST", "/chats/voice/upload", true, s.Attachment)
	s.Route("POST", "/chats/voice/{voice:[0-9]+}/transcribe", true, s.Transcription)
	s.Route("GET", "/chats/voice/{voice:[0-9]+}/status", true, s.Transcription)
	s.Route("POST", "/chats/voice/transcribe-callback", false, s.TranscriptionCallback)
	s.Route("GET,POST", "/chats/gemini", true, s.Gemini)
	s.Route("GET", "/chats/ice_servers", true, func(q *Request) any {
		ice := map[string]any{"urls": strings.Split(s.Config.
			ICEURLs, ","),
		}
		if s.Config.
			ICEUsername != "" {
			ice["username"] = s.Config.
				ICEUsername

			ice["credential"] =
				s.Config.ICECredential
		}
		return map[string]any{"iceServers": []any{ice}}
	})
	s.Route("GET", "/ws/chat/{id:[0-9]+}", true, s.Websocket)
}
