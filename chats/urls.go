package chats

import (
	. "litloop/backend/internal/httpapi"
	"strings"
)

func Register(server *Server) {
	server.Route("GET", "/chats", true, server.ChatList)
	server.Route("GET", "/chats/me", true, server.ChatList)
	server.Route("GET,POST,DELETE", "/chats/presence", true, server.Presence)
	server.Route("GET", "/chats/direct", true, server.ChatList)
	server.Route("GET", "/chats/group", true, server.ChatList)
	server.Route("GET,POST", "/chats/direct/{user:[0-9]+}", true, server.DirectChat)
	server.Route("GET,POST", "/chats/saved", true, server.DirectChat)
	server.Route("POST", "/chats/group/create", true, server.CreateGroup)
	server.Route("GET", "/chats/group/{id:[0-9]+}", true, func(req *Request) any {
		chat := server.ChatAccess(req, req.ID("id"))
		return server.ChatJSON(
			chat, req.User.ID, true,
		)
	})
	server.Route("POST", "/chats/group/{id:[0-9]+}/add", true, server.GroupMember)
	server.Route("POST", "/chats/group/{id:[0-9]+}/remove", true, server.GroupMember)
	server.Route("POST", "/chats/{id:[0-9]+}/send", true, server.SendMessage)
	server.Route("GET", "/chats/{id:[0-9]+}/messages", true, server.MessagesAfter)
	server.Route("GET", "/chats/tabs", true, server.ChatTabs)
	server.Route("PUT", "/chats/tabs", true, server.ChatTabsWrite)
	server.Route("POST", "/chats/{id:[0-9]+}/call", true, server.CallCommand)
	server.Route("GET", "/chats/{id:[0-9]+}/call", true, server.CallState)
	server.Route("GET,POST", "/chats/{id:[0-9]+}/voice", true, server.GroupVoice)
	server.Route("GET", "/chats/calls/incoming", true, server.IncomingCall)
	server.Route("POST", "/chats/{id:[0-9]+}/read", true, server.ChatAction)
	server.Route("POST,DELETE", "/chats/{id:[0-9]+}/delete", true, server.ChatAction)
	server.Route("POST", "/chats/upload", true, server.Attachment)
	server.Route("POST", "/chats/voice/upload", true, server.Attachment)
	server.Route("POST", "/chats/voice/{voice:[0-9]+}/transcribe", true, server.Transcription)
	server.Route("GET", "/chats/voice/{voice:[0-9]+}/status", true, server.Transcription)
	server.Route("POST", "/chats/voice/transcribe-callback", false, server.TranscriptionCallback)
	server.Route("GET,POST", "/chats/gemini", true, server.Gemini)
	server.Route("GET", "/chats/ice_servers", true, func(req *Request) any {
		ice := map[string]any{"urls": strings.Split(server.Config.
			ICEURLs, ","),
		}
		if server.Config.
			ICEUsername != "" {
			ice["username"] = server.Config.
				ICEUsername

			ice["credential"] =
				server.Config.ICECredential
		}
		return map[string]any{"iceServers": []any{ice}}
	})
	server.Route("GET", "/ws/chat/{id:[0-9]+}", true, server.Websocket)
}
