package chats

import (
	"crypto/rand"
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	Websocket "github.com/gorilla/websocket"
)

func (handler *Handler) Websocket(req *Request) any {
	Notifications := req.Param("id") == ""
	room := ""
	var notificationUser int64
	var presenceSession string
	if Notifications {
		presenceSession = rand.Text()
	}
	if !Notifications {
		handler.ChatAccess(req, req.ID("id"))
		room = fmt.Sprint(req.ID("id"))
	} else if req.User != nil {
		notificationUser = req.User.ID
		room = "user:" + fmt.Sprint(req.User.ID)
	}
	up := Websocket.Upgrader{CheckOrigin: func(httpReq *http.Request) bool {
		origin := httpReq.Header.Get("Origin")
		if origin == "" {
			return true
		}
		for _, allowedOrigin := range handler.Config.Origins {
			if allowedOrigin == origin {
				return true
			}
		}
		return false
	}}
	Conn, err := up.Upgrade(req.W, req.R, nil)
	if err != nil {
		req.Status = 0
		return nil
	}
	req.Status = 0
	peer := &Peer{Conn: Conn, Send: make(chan any, 64), Done: make(chan struct{})}
	defer peer.Close()
	defer func() {
		if room != "" {
			handler.Hub.Remove(room, peer)
		}
		if notificationUser != 0 {
			handler.updatePresence(notificationUser, presenceSession, false, 0)
		}
	}()
	if room != "" {
		handler.Hub.Add(room, peer)
	}
	if notificationUser != 0 {
		handler.updatePresence(notificationUser, presenceSession, true, 0)
	}
	Conn.SetReadLimit(1 << 20)
	_ = Conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	Conn.SetPongHandler(func(string) error {
		if err := Conn.SetReadDeadline(time.Now().Add(70 * time.Second)); err != nil {
			return err
		}
		if notificationUser != 0 {
			handler.updatePresence(notificationUser, presenceSession, true, 0)
			handler.sendPresence(peer, notificationUser)
		}
		return nil
	})
	go func() {
		tick := time.NewTicker(25 * time.Second)
		defer tick.Stop()
		defer peer.Close()
		for {
			select {
			case <-peer.Done:
				return
			case message := <-peer.Send:
				_ = Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if Conn.WriteJSON(message) != nil {
					return
				}
			case <-tick.C:
				if Conn.WriteControl(Websocket.PingMessage, nil, time.Now().Add(10*time.Second)) != nil {
					return
				}
			}
		}
	}()
	for {
		var payload map[string]any
		if Conn.ReadJSON(&payload) != nil {
			break
		}
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					msg := "Message failed"
					if err, ok := recovered.(Error); ok {
						msg = err.Message
					}
					select {
					case peer.Send <- map[string]any{"type": "error", "message": msg}:
					default:
					}
				}
			}()
			if Notifications {
				switch Str(payload["type"]) {
				case "auth":
					claims := handler.Claims(Str(payload["token"]), "access")
					if claims == nil {
						Fail(401, "Invalid token")
					}
					user := Get[models.User](handler.DB, claims["user_id"])
					if !user.IsActive {
						Fail(401, "Inactive user")
					}
					if notificationUser != user.ID {
						if room != "" {
							handler.Hub.Remove(room, peer)
							handler.updatePresence(notificationUser, presenceSession, false, 0)
						}
						notificationUser = user.ID
						room = "user:" + strconv.FormatInt(user.ID, 10)
						handler.Hub.Add(room, peer)
						handler.updatePresence(user.ID, presenceSession, true, 0)
					}
					peer.Send <- map[string]any{"type": "auth_ok"}
					peer.Send <- map[string]any{"type": "unread_update", "unread_count": handler.UnreadMessages(user.ID)}
					handler.sendPresence(peer, user.ID)
				case "presence":
					if notificationUser != 0 {
						handler.sendPresence(peer, notificationUser)
					}
				}
				return
			}
			if !handler.Member(req.ID("id"), req.User.ID) {
				Fail(403, "Chat membership required")
			}
			typ := Str(payload["type"])
			if typ == "" || typ == "chat_message" || typ == "send_message" {
				handler.SaveMessage(req.ID("id"), *req.User, payload)
			} else if typ == "typing" {
				handler.Hub.Broadcast(room, map[string]any{"type": "typing", "user_id": req.User.ID, "username": req.User.Username, "is_typing": payload["is_typing"]})
			} else if strings.HasPrefix(typ, "voip_") {
				if handler.VoIP(req, payload) {
					handler.broadcastCall(req, payload)
				}
			}
		}()
	}
	return nil
}
