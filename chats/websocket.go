package chats

import (
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	Websocket "github.com/gorilla/websocket"
)

func (s *Handler) Websocket(q *Request) any {
	Notifications := q.Param("id") == ""
	room := ""
	if !Notifications {
		s.ChatAccess(q, q.ID("id"))
		room = fmt.Sprint(q.ID("id"))
	} else if q.User != nil {
		room = "user:" + fmt.Sprint(q.User.ID)
	}
	up := Websocket.Upgrader{CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		for _, o := range s.Config.Origins {
			if o == origin {
				return true
			}
		}
		return false
	}}
	Conn, e := up.Upgrade(q.W, q.R, nil)
	if e != nil {
		q.Status = 0
		return nil
	}
	q.Status = 0
	p := &Peer{Conn: Conn, Send: make(chan any, 64), Done: make(chan struct{})}
	defer p.Close()
	defer func() {
		if room != "" {
			s.Hub.Remove(room, p)
		}
	}()
	if room != "" {
		s.Hub.Add(room, p)
	}
	Conn.SetReadLimit(1 << 20)
	_ = Conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	Conn.SetPongHandler(func(string) error { return Conn.SetReadDeadline(time.Now().Add(70 * time.Second)) })
	go func() {
		tick := time.NewTicker(25 * time.Second)
		defer tick.Stop()
		defer p.Close()
		for {
			select {
			case <-p.Done:
				return
			case v := <-p.Send:
				_ = Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if Conn.WriteJSON(v) != nil {
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
		var b map[string]any
		if Conn.ReadJSON(&b) != nil {
			break
		}
		func() {
			defer func() {
				if v := recover(); v != nil {
					msg := "Message failed"
					if e, ok := v.(Error); ok {
						msg = e.Message
					}
					select {
					case p.Send <- map[string]any{"type": "error", "message": msg}:
					default:
					}
				}
			}()
			if Notifications {
				if Str(b["type"]) == "auth" {
					c := s.Claims(Str(b["token"]), "access")
					if c == nil {
						Fail(401, "Invalid token")
					}
					u := Get[models.User](s.DB, c["user_id"])
					if !u.IsActive {
						Fail(401, "Inactive user")
					}
					if room != "" {
						s.Hub.Remove(room, p)
					}
					room = "user:" + strconv.FormatInt(u.ID, 10)
					s.Hub.Add(room, p)
					p.Send <- map[string]any{"type": "auth_ok"}
				}
				return
			}
			if !s.Member(q.ID("id"), q.User.ID) {
				Fail(403, "Chat membership required")
			}
			typ := Str(b["type"])
			if typ == "" || typ == "chat_message" {
				s.SaveMessage(q.ID("id"), *q.User, b)
			} else if typ == "typing" {
				s.Hub.Broadcast(room, map[string]any{"type": "typing", "user_id": q.User.ID, "username": q.User.Username, "is_typing": b["is_typing"]})
			} else if strings.HasPrefix(typ, "voip_") {
				s.VoIP(q, b)
				b["from_user_id"] = q.User.ID
				b["username"] = q.User.Username
				s.Hub.Broadcast(room, b)
			}
		}()
	}
	return nil
}
