package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"litloop/backend/internal/config"
	"litloop/backend/internal/engagement"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"litloop/backend/internal/testutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	Websocket "github.com/gorilla/websocket"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/pbkdf2"
)

type testFixture struct {
	t     *testing.T
	s     *Server
	http  *httptest.Server
	Token string
	user  int64
}

func setup(tester *testing.T) *testFixture {
	tester.Helper()
	appConfig := config.Config{SQLitePath: filepath.Join(tester.TempDir(), "test.db"), JWTSecret: "test-key-with-at-least-thirty-two-characters", MediaDir: tester.TempDir(), AccessTTL: time.Hour, RefreshTTL: 24 * time.Hour, Origins: []string{"http://localhost:3000"}}
	db := testutil.Open(tester, &appConfig)
	server := New(db, appConfig)
	ts := httptest.NewServer(server)
	server.Config.PublicURL = ts.URL
	tester.Cleanup(func() { ts.Close(); pool, _ := db.DB(); _ = pool.Close() })
	fixture := &testFixture{t: tester, s: server, http: ts}
	out := fixture.req("POST", "/users/signup/", "", map[string]any{"email": "alice@example.com", "username": "alice", "password": "password123"}, 201)
	fixture.Token = out["access"].(string)
	fixture.user = int64(out["user"].(map[string]any)["id"].(float64))
	return fixture
}
func (fixture *testFixture) req(method, path, Token string, body any, status int) map[string]any {
	fixture.t.Helper()
	var data io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			fixture.t.Fatal(err)
		}
		data = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, fixture.http.URL+path, data)
	if err != nil {
		fixture.t.Fatal(err)
	}
	if Token != "" {
		req.Header.Set("Authorization", "Bearer "+Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := fixture.http.Client().Do(req)
	if err != nil {
		fixture.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		fixture.t.Fatal(err)
	}
	if res.StatusCode != status {
		fixture.t.Fatalf("%s %s: expected %d, got %d: %s", method, path, status, res.StatusCode, raw)
	}
	out := map[string]any{}
	if len(raw) > 0 {
		if err = json.Unmarshal(raw, &out); err != nil {
			fixture.t.Fatalf("JSON object %s: %v", raw, err)
		}
	}
	return out
}
func (fixture *testFixture) second() (string, int64) {
	out := fixture.req("POST", "/users/signup", "", map[string]any{"email": "bob@example.com", "username": "bob", "password": "password123"}, 201)
	return out["access"].(string), int64(out["user"].(map[string]any)["id"].(float64))
}
func TestAuthenticationAndDjangoPasswords(tester *testing.T) {
	fixture := setup(tester)
	fixture.req("GET", "/users/me", "", nil, 401)
	fixture.req("POST", "/users/signin", "", map[string]any{"email_or_username": "alice", "password": "wrong"}, 401)
	signed := fixture.req("POST", "/users/signin", "", map[string]any{"email_or_username": "alice", "password": "password123"}, 200)
	fixture.req("POST", "/auth/refresh", "", map[string]any{"refresh": signed["refresh"]}, 200)
	fixture.req("POST", "/auth/refresh", "", map[string]any{"refresh": signed["access"]}, 401)
	fixture.req("PUT", fmt.Sprintf("/users/%d/role", fixture.user), fixture.Token, map[string]any{"is_superuser": true}, 403)
	key := pbkdf2.Key([]byte("legacy-password"), []byte("django-salt"), 600000, 32, sha256.New)
	encoded := "pbkdf2_sha256$600000$django-salt$" + base64.StdEncoding.EncodeToString(key)
	if err := fixture.s.DB.Model(&models.User{}).Where("id = ?", fixture.user).Update("password", encoded).Error; err != nil {
		tester.Fatal(err)
	}
	fixture.req("POST", "/users/signin", "", map[string]any{"email_or_username": "alice@example.com", "password": "legacy-password"}, 200)
	fixture.req("POST", "/auth/logout", fixture.Token, map[string]any{}, 200)
	fixture.req("GET", "/users/me", fixture.Token, nil, 401)
}
func TestPostsOwnershipReactionsAndComments(tester *testing.T) {
	fixture := setup(tester)
	bob, _ := fixture.second()
	response := fixture.req("POST", "/posts/create/", fixture.Token, map[string]any{"title": "<b>Hello</b>", "description": "world"}, 201)
	id := int64(response["id"].(float64))
	path := fmt.Sprintf("/posts/%d", id)
	if response["title"] != "Hello" {
		tester.Fatal(response)
	}
	fixture.req("PATCH", path, bob, map[string]any{"title": "stolen"}, 403)
	fixture.req("GET", path, "", nil, 200)
	fixture.req("POST", path+"/like", bob, map[string]any{}, 200)
	response = fixture.req("GET", path, bob, nil, 200)
	if response["is_liked"] != true || response["likes_count"] != float64(1) {
		tester.Fatal(response)
	}
	fixture.req("POST", path+"/like", bob, map[string]any{}, 200)
	commentResponse := fixture.req("POST", path+"/comments/create", bob, map[string]any{"text": "Comment"}, 201)
	fixture.req("GET", path+fmt.Sprintf("/comments/%d", int64(commentResponse["id"].(float64))), "", nil, 200)
	fixture.req("POST", "/posts/impressions/batch", bob, map[string]any{"post_ids": []int64{id}}, 200)
	fixture.req("DELETE", path, bob, nil, 403)
	fixture.req("DELETE", path, fixture.Token, nil, 204)
	fixture.req("GET", path, "", nil, 404)
	var count int64
	if err := fixture.s.DB.Model(&models.PostComment{}).Count(&count).Error; err != nil || count != 0 {
		tester.Fatal(err, count)
	}
}

func TestPostEngagementQueuesViewsAndLikeChanges(tester *testing.T) {
	fixture := setup(tester)
	fixture.s.Config.AstraDatabaseID = "test-database"
	fixture.s.Config.AstraToken = "test-token"
	post := fixture.req("POST", "/posts/create", fixture.Token, map[string]any{"title": "Engagement"}, 201)
	id := int64(post["id"].(float64))
	like := fmt.Sprintf("/posts/%d/like", id)
	fixture.req("POST", "/views/up", fixture.Token, map[string]any{"post_id": id}, 202)
	fixture.req("POST", like, fixture.Token, nil, 200)
	fixture.req("POST", like, fixture.Token, nil, 200)
	fixture.req("POST", "/views/up", fixture.Token, map[string]any{"post_id": id + 1}, 404)

	var tasks []models.Task
	if err := fixture.s.DB.Where("kind = ?", engagement.TaskKind).Order("id").Find(&tasks).Error; err != nil {
		tester.Fatal(err)
	}
	actions := []string{"view", "like", "unlike"}
	if len(tasks) != len(actions) {
		tester.Fatalf("got %d engagement tasks, want %d", len(tasks), len(actions))
	}
	for index, task := range tasks {
		var event engagement.Event
		if err := json.Unmarshal(task.Payload, &event); err != nil {
			tester.Fatal(err)
		}
		if event.ResourceType != "post" || event.ResourceID != id || event.ActorID != fixture.user ||
			event.Action != actions[index] || event.OccurredAt.IsZero() {
			tester.Fatalf("unexpected engagement event: %+v", event)
		}
	}
}

func TestNotesAndTodoBoundaries(tester *testing.T) {
	fixture := setup(tester)
	bob, _ := fixture.second()
	response := fixture.req("POST", "/notes/pages/create", fixture.Token, map[string]any{"title": "Notes"}, 201)
	id := int64(response["id"].(float64))
	path := fmt.Sprintf("/notes/pages/%d", id)
	fixture.req("GET", path, bob, nil, 404)
	blockResponse := fixture.req("POST", path+"/blocks/create", fixture.Token, map[string]any{"type": "table", "table_data": map[string]any{"columns": []string{"Name"}, "rows": [][]string{{"Alice"}}}}, 201)
	block := int64(blockResponse["id"].(float64))
	fixture.req("PUT", fmt.Sprintf("/notes/blocks/%d/table", block), bob, map[string]any{"rows": []any{}}, 404)
	fixture.req("PUT", fmt.Sprintf("/notes/blocks/%d/table", block), fixture.Token, map[string]any{"rows": [][]string{{"Bob"}}}, 200)
	fixture.req("POST", path+"/tags/add", fixture.Token, map[string]any{"name": "Go"}, 201)
	response = fixture.req("GET", path, fixture.Token, nil, 200)
	if len(response["blocks"].([]any)) != 1 || len(response["tags"].([]any)) != 1 {
		tester.Fatal(response)
	}
	fixture.req("PUT", path+"/update", fixture.Token, map[string]any{"parent_id": id}, 400)
	list := fixture.req("POST", "/todos/lists/create", fixture.Token, map[string]any{"name": "Work"}, 201)
	lid := int64(list["id"].(float64))
	fixture.req("POST", "/todos/create", bob, map[string]any{"title": "x", "todo_list": lid}, 404)
	todo := fixture.req("POST", "/todos/create", fixture.Token, map[string]any{"title": "Write Go", "todo_list": lid}, 201)
	tid := int64(todo["id"].(float64))
	fixture.req("PUT", fmt.Sprintf("/todos/%d/update", tid), fixture.Token, map[string]any{"completed": true, "order": 0}, 200)
	fixture.req("DELETE", fmt.Sprintf("/todos/lists/%d/delete", lid), fixture.Token, nil, 200)
	fixture.req("GET", fmt.Sprintf("/todos/%d", tid), fixture.Token, nil, 404)
	fixture.req("DELETE", path+"/delete", fixture.Token, nil, 200)
	var count int64
	fixture.s.DB.Model(&models.BlockTable{}).Count(&count)
	if count != 0 {
		tester.Fatal("orphan table", count)
	}
}
func TestChatsWebsocketAndMembership(tester *testing.T) {
	fixture := setup(tester)
	bob, bid := fixture.second()
	fixture.req("POST", "/chats/1/send", bob, map[string]any{"text": "intrusion"}, 403)
	response := fixture.req("GET", fmt.Sprintf("/chats/direct/%d", bid), fixture.Token, nil, 200)
	id := int64(response["id"].(float64))
	chats := fixture.req("GET", "/chats/me", fixture.Token, nil, 200)["chats"].([]any)
	found := false
	for _, item := range chats {
		chat := item.(map[string]any)
		if chat["id"] != response["id"] {
			continue
		}
		participant := chat["other_participant"].(map[string]any)
		if participant["id"] != float64(bid) || chat["target_user"] != nil {
			tester.Fatal(chat)
		}
		found = true
	}
	if !found {
		tester.Fatal("direct chat missing from chat list")
	}
	path := fmt.Sprintf("/ws/chat/%d/?token=%s", id, url.QueryEscape(bob))
	ws, _, err := Websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(fixture.http.URL, "http")+path, nil)
	if err != nil {
		tester.Fatal(err)
	}
	defer ws.Close()
	fixture.req("POST", fmt.Sprintf("/chats/%d/send", id), fixture.Token, map[string]any{"text": "Hello Bob"}, 200)
	_ = ws.SetReadDeadline(time.Now().Add(3 * time.Second))
	var event map[string]any
	if err = ws.ReadJSON(&event); err != nil {
		tester.Fatal(err)
	}
	if event["type"] != "new_message" {
		tester.Fatal(event)
	}
	firstMessageID := event["message"].(map[string]any)["id"]
	if err = ws.WriteJSON(map[string]any{"type": "send_message", "text": "Hi Alice"}); err != nil {
		tester.Fatal(err)
	}
	if err = ws.ReadJSON(&event); err != nil {
		tester.Fatal(err)
	}
	response = fixture.req("GET", fmt.Sprintf("/chats/direct/%d", bid), fixture.Token, nil, 200)
	if len(response["messages"].([]any)) != 2 {
		tester.Fatal(response)
	}
	missed := fixture.req("GET", fmt.Sprintf("/chats/%d/messages?after=%.0f", id, firstMessageID), bob, nil, 200)
	messages := missed["messages"].([]any)
	if len(messages) != 1 || messages[0].(map[string]any)["text"] != "Hi Alice" || missed["has_more"] != false {
		tester.Fatal(missed)
	}
	fixture.req("POST", fmt.Sprintf("/chats/%d/read", id), bob, map[string]any{}, 200)
	if err = ws.ReadJSON(&event); err != nil {
		tester.Fatal(err)
	}
	if ids, ok := event["message_ids"].([]any); event["type"] != "read_receipt" || !ok || len(ids) != 1 || ids[0] != firstMessageID {
		tester.Fatal(event)
	}
}

func TestCallNotificationsOutsideChat(tester *testing.T) {
	fixture := setup(tester)
	bob, bid := fixture.second()
	chat := fixture.req("GET", fmt.Sprintf("/chats/direct/%d", bid), fixture.Token, nil, 200)
	id := int64(chat["id"].(float64))
	connect := func(path string) *Websocket.Conn {
		tester.Helper()
		conn, _, err := Websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(fixture.http.URL, "http")+path, nil)
		if err != nil {
			tester.Fatal(err)
		}
		tester.Cleanup(func() { conn.Close() })
		return conn
	}
	read := func(conn *Websocket.Conn, eventType string) map[string]any {
		tester.Helper()
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		for {
			var event map[string]any
			if err := conn.ReadJSON(&event); err != nil {
				tester.Fatalf("waiting for %s: %v", eventType, err)
			}
			if event["type"] == eventType {
				return event
			}
		}
	}
	send := func(conn *Websocket.Conn, event map[string]any) {
		tester.Helper()
		if err := conn.WriteJSON(event); err != nil {
			tester.Fatal(err)
		}
	}

	notifications := connect("/ws/notifications/")
	send(notifications, map[string]any{"type": "auth", "token": bob})
	read(notifications, "auth_ok")
	caller := connect(fmt.Sprintf("/ws/chat/%d/?token=%s", id, url.QueryEscape(fixture.Token)))
	send(caller, map[string]any{"type": "voip_call_start", "call_type": "audio"})
	incoming := read(notifications, "incoming_call")
	if incoming["chat_id"] != chat["id"] || incoming["call_id"] == nil || incoming["call_type"] != "voice" {
		tester.Fatal(incoming)
	}
	send(caller, map[string]any{"type": "voip_call_end", "call_id": incoming["call_id"]})
	closed := read(notifications, "call_closed")
	if closed["chat_id"] != chat["id"] || closed["call_id"] != incoming["call_id"] {
		tester.Fatal(closed)
	}
}

func TestCallCommandsAndRingingRecoveryWithoutRoomSockets(tester *testing.T) {
	fixture := setup(tester)
	bob, bid := fixture.second()
	chat := fixture.req("GET", fmt.Sprintf("/chats/direct/%d", bid), fixture.Token, nil, 200)
	command := fmt.Sprintf("/chats/%.0f/call", chat["id"])
	started := fixture.req("POST", command, fixture.Token, map[string]any{"type": "voip_call_start", "call_type": "audio"}, 200)
	callID := started["call_id"]
	restored := fixture.req("GET", "/chats/calls/incoming", bob, nil, 200)["incoming_call"].(map[string]any)
	if restored["call_id"] != callID || restored["call_type"] != "voice" || restored["expires_at"] == nil {
		tester.Fatal(restored)
	}
	if own := fixture.req("GET", "/chats/calls/incoming", fixture.Token, nil, 200)["incoming_call"]; own != nil {
		tester.Fatal(own)
	}
	statePath := fmt.Sprintf("%s?call_id=%.0f&signals=1", command, callID)
	if status := fixture.req("GET", statePath, fixture.Token, nil, 200)["status"]; status != "initiated" {
		tester.Fatal(status)
	}
	fixture.req("GET", statePath, bob, nil, 404)

	connect := func(token string) *Websocket.Conn {
		tester.Helper()
		conn, _, err := Websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(fixture.http.URL, "http")+"/ws/notifications/", nil)
		if err != nil {
			tester.Fatal(err)
		}
		tester.Cleanup(func() { conn.Close() })
		if err := conn.WriteJSON(map[string]any{"type": "auth", "token": token}); err != nil {
			tester.Fatal(err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		var event map[string]any
		if err := conn.ReadJSON(&event); err != nil || event["type"] != "auth_ok" {
			tester.Fatalf("notification auth: %v %v", event, err)
		}
		return conn
	}
	expect := func(conn *Websocket.Conn, kind string) map[string]any {
		tester.Helper()
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		for {
			var event map[string]any
			if err := conn.ReadJSON(&event); err != nil {
				tester.Fatalf("waiting for %s: %v", kind, err)
			}
			if event["type"] == kind {
				if event["call_id"] != callID || event["chat_id"] != chat["id"] {
					tester.Fatal(event)
				}
				return event
			}
		}
	}

	caller := connect(fixture.Token)
	recipient := connect(bob)
	fixture.req("POST", command, bob, map[string]any{"type": "voip_call_accept", "call_id": callID}, 200)
	expect(caller, "call_accepted")
	fixture.req("POST", command, bob, map[string]any{"type": "voip_call_accept", "call_id": callID}, 200)
	if status := fixture.req("GET", statePath, fixture.Token, nil, 200)["status"]; status != "ongoing" {
		tester.Fatal(status)
	}
	fixture.req("POST", command, fixture.Token, map[string]any{
		"type": "voip_webrtc_signal", "call_id": callID,
		"signal_data": map[string]any{"type": "offer", "sdp": "test offer"},
	}, 200)
	event := expect(recipient, "voip_webrtc_signal")
	if event["signal_data"].(map[string]any)["sdp"] != "test offer" {
		tester.Fatal(event)
	}
	replayed := fixture.req("GET", statePath, bob, nil, 200)["signals"].([]any)
	if len(replayed) != 1 || replayed[0].(map[string]any)["signal_id"] != event["signal_id"] {
		tester.Fatal(replayed)
	}
	if own := fixture.req("GET", statePath, fixture.Token, nil, 200)["signals"].([]any); len(own) != 0 {
		tester.Fatal(own)
	}
	fixture.req("POST", command, bob, map[string]any{"type": "voip_call_end", "call_id": callID}, 200)
	expect(caller, "call_closed")
	if status := fixture.req("GET", statePath, fixture.Token, nil, 200)["status"]; status != "completed" {
		tester.Fatal(status)
	}
}

func TestRealtimeDeliveryAndPresenceAcrossSessions(tester *testing.T) {
	fixture := setup(tester)
	bob, bid := fixture.second()
	chat := fixture.req("GET", fmt.Sprintf("/chats/direct/%d", bid), fixture.Token, nil, 200)
	chatID := int64(chat["id"].(float64))
	read := func(conn *Websocket.Conn, eventType string) map[string]any {
		tester.Helper()
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		for {
			var event map[string]any
			if err := conn.ReadJSON(&event); err != nil {
				tester.Fatalf("waiting for %s: %v", eventType, err)
			}
			if event["type"] == eventType {
				return event
			}
		}
	}
	notifications := func(token string) (*Websocket.Conn, map[string]any) {
		tester.Helper()
		conn, _, err := Websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(fixture.http.URL, "http")+"/ws/notifications/", nil)
		if err != nil {
			tester.Fatal(err)
		}
		tester.Cleanup(func() { conn.Close() })
		if err := conn.WriteJSON(map[string]any{"type": "auth", "token": token}); err != nil {
			tester.Fatal(err)
		}
		return conn, read(conn, "presence_snapshot")
	}

	alice, snapshot := notifications(fixture.Token)
	bobKey := fmt.Sprint(bid)
	if snapshot["users"].(map[string]any)[bobKey] != false {
		tester.Fatal(snapshot)
	}
	bobFirst, _ := notifications(bob)
	if event := read(alice, "presence_update"); event["user_id"] != float64(bid) || event["online"] != true {
		tester.Fatal(event)
	}
	bobSecond, _ := notifications(bob)
	fixture.req("POST", fmt.Sprintf("/chats/%d/send", chatID), fixture.Token, map[string]any{"text": "Delivered outside the room"}, 200)
	event := read(bobSecond, "new_message")
	if event["chat_id"] != chat["id"] || event["message"].(map[string]any)["text"] != "Delivered outside the room" {
		tester.Fatal(event)
	}

	bobFirst.Close()
	if err := alice.WriteJSON(map[string]any{"type": "presence"}); err != nil {
		tester.Fatal(err)
	}
	snapshot = read(alice, "presence_snapshot")
	if snapshot["users"].(map[string]any)[bobKey] != true {
		tester.Fatal(snapshot)
	}
	bobSecond.Close()
	if event := read(alice, "presence_update"); event["user_id"] != float64(bid) || event["online"] != false {
		tester.Fatal(event)
	}
}

func TestHTTPPresenceAndLastSeen(tester *testing.T) {
	fixture := setup(tester)
	bob, bid := fixture.second()
	bobKey := fmt.Sprint(bid)
	const path = "/chats/presence"
	if users := fixture.req("GET", path, fixture.Token, nil, 200)["users"].(map[string]any); len(users) != 0 {
		tester.Fatal(users)
	}
	direct := fmt.Sprintf("/chats/direct/%d", bid)
	chat := fixture.req("GET", direct, fixture.Token, nil, 200)
	before := chat["other_participant"].(map[string]any)["last_seen_at"].(string)
	started, err := time.Parse(time.RFC3339Nano, before)
	if err != nil {
		tester.Fatal(err)
	}
	offline := fixture.req("GET", path, fixture.Token, nil, 200)
	if offline["users"].(map[string]any)[bobKey] != false || offline["last_seen_at"].(map[string]any)[bobKey] != before {
		tester.Fatal(offline)
	}

	const session = "test-http-presence-session"
	fixture.req("POST", path, bob, map[string]any{"session_id": session}, 200)
	if online := fixture.req("GET", path, fixture.Token, nil, 200)["users"].(map[string]any)[bobKey]; online != true {
		tester.Fatal(online)
	}
	fixture.req("DELETE", path+"?session_id="+session, bob, nil, 200)
	offline = fixture.req("GET", path, fixture.Token, nil, 200)
	stamp := offline["last_seen_at"].(map[string]any)[bobKey].(string)
	lastSeen, err := time.Parse(time.RFC3339Nano, stamp)
	if err != nil || !lastSeen.After(started) || offline["users"].(map[string]any)[bobKey] != false {
		tester.Fatal(offline)
	}
	chat = fixture.req("GET", direct, fixture.Token, nil, 200)
	if actual := chat["other_participant"].(map[string]any)["last_seen_at"]; actual != stamp {
		tester.Fatalf("chat last seen = %v, want %s", actual, stamp)
	}
}

func TestGroupVoiceWithThreeMembers(tester *testing.T) {
	fixture := setup(tester)
	bob, bid := fixture.second()
	third := fixture.req("POST", "/users/signup", "", map[string]any{
		"email": "group-voice-third@example.test", "password": "test-password-third",
	}, 201)
	carol := third["access"].(string)
	cid := third["user"].(map[string]any)["id"]
	created := fixture.req("POST", "/chats/group/create", fixture.Token, map[string]any{
		"name": "Test group voice", "participant_ids": []any{bid, cid},
	}, 201)
	chat := created["chat"].(map[string]any)
	path := fmt.Sprintf("/chats/%.0f/voice", chat["id"])
	if room := fixture.req("GET", path, bob, nil, 200)["room"]; room != nil {
		tester.Fatal(room)
	}
	const aliceSession = "group-voice-session-alice"
	const bobSession = "group-voice-session-bobby"
	const carolSession = "group-voice-session-carol"
	join := func(token, session string) map[string]any {
		return fixture.req("POST", path, token, map[string]any{"action": "join", "session_id": session}, 200)["room"].(map[string]any)
	}
	roomID := join(fixture.Token, aliceSession)["id"]
	if actual := join(bob, bobSession)["id"]; actual != roomID {
		tester.Fatal(actual)
	}
	room := join(carol, carolSession)
	if room["id"] != roomID || len(room["members"].([]any)) != 3 {
		tester.Fatal(room)
	}
	event := map[string]any{
		"action": "signal", "room_id": roomID, "session_id": bobSession,
		"target_user_id": cid, "target_session_id": carolSession,
		"signal_data": map[string]any{"type": "offer", "sdp": "group offer"},
	}
	fixture.req("POST", path, bob, event, 200)
	poll := func(token, session string) map[string]any {
		return fixture.req("POST", path, token, map[string]any{"action": "sync", "room_id": roomID, "session_id": session}, 200)
	}
	signals := poll(carol, carolSession)["signals"].([]any)
	if len(signals) != 1 || signals[0].(map[string]any)["from_session_id"] != bobSession {
		tester.Fatal(signals)
	}
	if own := poll(fixture.Token, aliceSession)["signals"].([]any); len(own) != 0 {
		tester.Fatal(own)
	}
	leave := func(token, session string) {
		fixture.req("POST", path, token, map[string]any{"action": "leave", "room_id": roomID, "session_id": session}, 200)
	}
	leave(fixture.Token, aliceSession)
	if members := poll(bob, bobSession)["room"].(map[string]any)["members"].([]any); len(members) != 2 {
		tester.Fatal(members)
	}
	leave(bob, bobSession)
	leave(carol, carolSession)
	if room := fixture.req("GET", path, bob, nil, 200)["room"]; room != nil {
		tester.Fatal(room)
	}
}

func TestLocalUploadAndPrivateVideo(tester *testing.T) {
	fixture := setup(tester)
	bob, _ := fixture.second()
	response := fixture.req("POST", "/videos/create_presigned_url", fixture.Token, map[string]any{"filename": "movie.mp4", "content_type": "video/mp4"}, 200)
	req, _ := http.NewRequest("PUT", response["url"].(string), strings.NewReader("video content"))
	req.Header.Set("Authorization", "Bearer "+fixture.Token)
	res, err := fixture.http.Client().Do(req)
	if err != nil {
		tester.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		tester.Fatal(res.Status)
	}
	fixture.req("POST", "/videos/complete_upload", bob, map[string]any{"upload_id": response["upload_id"], "key": response["key"]}, 404)
	Done := fixture.req("POST", "/videos/complete_upload", fixture.Token, map[string]any{"upload_id": response["upload_id"], "key": response["key"], "title": "Private"}, 200)
	id := int64(Done["id"].(float64))
	fixture.req("GET", fmt.Sprintf("/videos/%d", id), "", nil, 403)
	fixture.req("GET", fmt.Sprintf("/videos/%d", id), fixture.Token, nil, 200)
	fixture.req("POST", "/videos/complete_upload", fixture.Token, map[string]any{"upload_id": response["upload_id"], "key": response["key"]}, 409)
	resp, err := fixture.http.Client().Get(Done["location"].(string))
	if err != nil {
		tester.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		tester.Fatal(resp.Status)
	}
	parsed, _ := url.Parse(Done["location"].(string))
	parsed.RawQuery = ""
	resp, err = fixture.http.Client().Get(parsed.String())
	if err != nil {
		tester.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 403 {
		tester.Fatal(resp.Status)
	}
	fixture.req("POST", "/videos/api/save_playback", fixture.Token, map[string]any{"video_id": id, "playback_time": 12.5}, 200)
	playbackResponse := fixture.req("GET", fmt.Sprintf("/videos/api/get_playback_time?video_id=%d", id), fixture.Token, nil, 200)
	if playbackResponse["playback_time"] != 12.5 {
		tester.Fatal(playbackResponse)
	}
}
func TestCommunityModeration(tester *testing.T) {
	fixture := setup(tester)
	bob, _ := fixture.second()
	response := fixture.req("POST", "/communities/create", fixture.Token, map[string]any{"name": "Go Readers"}, 201)
	id := int64(response["id"].(float64))
	path := fmt.Sprintf("/communities/%d", id)
	fixture.req("GET", "/communities/@go-readers", "", nil, 200)
	fixture.req("POST", path+"/join", bob, map[string]any{}, 200)
	postResponse := fixture.req("POST", "/posts/create", bob, map[string]any{"title": "Discussion"}, 201)
	cp := fixture.req("POST", path+"/posts/request", bob, map[string]any{"post_id": postResponse["id"]}, 201)
	cid := int64(cp["id"].(float64))
	fixture.req("POST", fmt.Sprintf("%s/posts/%d/approve", path, cid), bob, map[string]any{}, 403)
	fixture.req("POST", fmt.Sprintf("%s/posts/%d/approve", path, cid), fixture.Token, map[string]any{}, 200)
	Posts := fixture.req("GET", path+"/posts", "", nil, 200)
	if len(Posts["posts"].([]any)) != 1 {
		tester.Fatal(Posts)
	}
	fixture.req("DELETE", path+"/delete", bob, nil, 403)
}
func TestInvalidInputsAndMissingServices(tester *testing.T) {
	fixture := setup(tester)
	fixture.req("GET", "/users/signup", "", nil, 405)
	fixture.req("POST", "/posts/create", fixture.Token, map[string]any{"description": strings.Repeat("a", 101)}, 400)
	fixture.req("POST", "/users/password-reset", "", map[string]any{"email": "alice@example.com"}, 503)
	fixture.req("GET", "/queries/brave/search?q=go", "", nil, 503)
	fixture.req("GET", "/admin", fixture.Token, nil, 403)
	fixture.req("GET", "/maintenance/delete-tables", fixture.Token, nil, 405)
	req, _ := http.NewRequest("POST", fixture.http.URL+"/posts/create", strings.NewReader("{broken"))
	req.Header.Set("Authorization", "Bearer "+fixture.Token)
	res, err := fixture.http.Client().Do(req)
	if err != nil {
		tester.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 400 {
		tester.Fatal(res.Status)
	}
}

func TestOriginalRouteCoverage(tester *testing.T) {
	raw, err := os.ReadFile("../docs/source-routes.json")
	if err != nil {
		tester.Fatal(err)
	}
	var source []struct {
		Path   string `json:"path"`
		Source string `json:"source"`
	}
	if err = json.Unmarshal(raw, &source); err != nil {
		tester.Fatal(err)
	}
	server := New(nil, config.Config{})
	routes := map[string]bool{}
	normalize := func(path string) string {
		return regexp.MustCompile(`\{[^}]+\}|<[^>]+>`).ReplaceAllString(strings.TrimRight(path, "/"), "{}")
	}
	err = chi.Walk(server.Router, func(method, path string, handler http.Handler, middleware ...func(http.Handler) http.Handler) error {
		routes[normalize(path)] = true
		return nil
	})
	if err != nil {
		tester.Fatal(err)
	}
	for _, route := range source {
		if !routes[normalize(route.Path)] {
			tester.Errorf("Missing route %s (%s)", route.Path, route.Source)
		}
	}
}
