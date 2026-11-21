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

type fixture struct {
	t     *testing.T
	s     *Server
	http  *httptest.Server
	Token string
	user  int64
}

func setup(t *testing.T) *fixture {
	t.Helper()
	c := config.Config{SQLitePath: filepath.Join(t.TempDir(), "test.db"), JWTSecret: "test-key-with-at-least-thirty-two-characters", MediaDir: t.TempDir(), AccessTTL: time.Hour, RefreshTTL: 24 * time.Hour, Origins: []string{"http://localhost:3000"}}
	db := testutil.Open(t, &c)
	s := New(db, c)
	ts := httptest.NewServer(s)
	s.Config.PublicURL = ts.URL
	t.Cleanup(func() { ts.Close(); pool, _ := db.DB(); _ = pool.Close() })
	f := &fixture{t: t, s: s, http: ts}
	out := f.req("POST", "/users/signup/", "", map[string]any{"email": "alice@example.com", "username": "alice", "password": "password123"}, 201)
	f.Token = out["access"].(string)
	f.user = int64(out["user"].(map[string]any)["id"].(float64))
	return f
}
func (f *fixture) req(method, path, Token string, body any, status int) map[string]any {
	f.t.Helper()
	var data io.Reader
	if body != nil {
		b, e := json.Marshal(body)
		if e != nil {
			f.t.Fatal(e)
		}
		data = bytes.NewReader(b)
	}
	req, e := http.NewRequest(method, f.http.URL+path, data)
	if e != nil {
		f.t.Fatal(e)
	}
	if Token != "" {
		req.Header.Set("Authorization", "Bearer "+Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, e := f.http.Client().Do(req)
	if e != nil {
		f.t.Fatal(e)
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(res.Body)
	if e != nil {
		f.t.Fatal(e)
	}
	if res.StatusCode != status {
		f.t.Fatalf("%s %s: expected %d, got %d: %s", method, path, status, res.StatusCode, raw)
	}
	out := map[string]any{}
	if len(raw) > 0 {
		if e = json.Unmarshal(raw, &out); e != nil {
			f.t.Fatalf("JSON object %s: %v", raw, e)
		}
	}
	return out
}
func (f *fixture) second() (string, int64) {
	out := f.req("POST", "/users/signup", "", map[string]any{"email": "bob@example.com", "username": "bob", "password": "password123"}, 201)
	return out["access"].(string), int64(out["user"].(map[string]any)["id"].(float64))
}
func TestAuthenticationAndDjangoPasswords(t *testing.T) {
	f := setup(t)
	f.req("GET", "/users/me", "", nil, 401)
	f.req("POST", "/users/signin", "", map[string]any{"email_or_username": "alice", "password": "wrong"}, 401)
	signed := f.req("POST", "/users/signin", "", map[string]any{"email_or_username": "alice", "password": "password123"}, 200)
	f.req("POST", "/auth/refresh", "", map[string]any{"refresh": signed["refresh"]}, 200)
	f.req("POST", "/auth/refresh", "", map[string]any{"refresh": signed["access"]}, 401)
	f.req("PUT", fmt.Sprintf("/users/%d/role", f.user), f.Token, map[string]any{"is_superuser": true}, 403)
	key := pbkdf2.Key([]byte("legacy-password"), []byte("django-salt"), 600000, 32, sha256.New)
	encoded := "pbkdf2_sha256$600000$django-salt$" + base64.StdEncoding.EncodeToString(key)
	if e := f.s.DB.Model(&models.User{}).Where("id = ?", f.user).Update("password", encoded).Error; e != nil {
		t.Fatal(e)
	}
	f.req("POST", "/users/signin", "", map[string]any{"email_or_username": "alice@example.com", "password": "legacy-password"}, 200)
	f.req("POST", "/auth/logout", f.Token, map[string]any{}, 200)
	f.req("GET", "/users/me", f.Token, nil, 401)
}
func TestPostsOwnershipReactionsAndComments(t *testing.T) {
	f := setup(t)
	bob, _ := f.second()
	p := f.req("POST", "/posts/create/", f.Token, map[string]any{"title": "<b>Hello</b>", "description": "world"}, 201)
	id := int64(p["id"].(float64))
	path := fmt.Sprintf("/posts/%d", id)
	if p["title"] != "Hello" {
		t.Fatal(p)
	}
	f.req("PATCH", path, bob, map[string]any{"title": "stolen"}, 403)
	f.req("GET", path, "", nil, 200)
	f.req("POST", path+"/like", bob, map[string]any{}, 200)
	p = f.req("GET", path, bob, nil, 200)
	if p["is_liked"] != true || p["likes_count"] != float64(1) {
		t.Fatal(p)
	}
	f.req("POST", path+"/like", bob, map[string]any{}, 200)
	c := f.req("POST", path+"/comments/create", bob, map[string]any{"text": "Comment"}, 201)
	f.req("GET", path+fmt.Sprintf("/comments/%d", int64(c["id"].(float64))), "", nil, 200)
	f.req("POST", "/posts/impressions/batch", bob, map[string]any{"post_ids": []int64{id}}, 200)
	f.req("DELETE", path, bob, nil, 403)
	f.req("DELETE", path, f.Token, nil, 204)
	f.req("GET", path, "", nil, 404)
	var n int64
	if e := f.s.DB.Model(&models.PostComment{}).Count(&n).Error; e != nil || n != 0 {
		t.Fatal(e, n)
	}
}

func TestPostEngagementQueuesViewsAndLikeChanges(t *testing.T) {
	f := setup(t)
	f.s.Config.AstraDatabaseID = "test-database"
	f.s.Config.AstraToken = "test-token"
	post := f.req("POST", "/posts/create", f.Token, map[string]any{"title": "Engagement"}, 201)
	id := int64(post["id"].(float64))
	like := fmt.Sprintf("/posts/%d/like", id)
	f.req("POST", "/views/up", f.Token, map[string]any{"post_id": id}, 202)
	f.req("POST", like, f.Token, nil, 200)
	f.req("POST", like, f.Token, nil, 200)
	f.req("POST", "/views/up", f.Token, map[string]any{"post_id": id + 1}, 404)

	var tasks []models.Task
	if err := f.s.DB.Where("kind = ?", engagement.TaskKind).Order("id").Find(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	actions := []string{"view", "like", "unlike"}
	if len(tasks) != len(actions) {
		t.Fatalf("got %d engagement tasks, want %d", len(tasks), len(actions))
	}
	for i, task := range tasks {
		var event engagement.Event
		if err := json.Unmarshal(task.Payload, &event); err != nil {
			t.Fatal(err)
		}
		if event.ResourceType != "post" || event.ResourceID != id || event.ActorID != f.user ||
			event.Action != actions[i] || event.OccurredAt.IsZero() {
			t.Fatalf("unexpected engagement event: %+v", event)
		}
	}
}

func TestNotesAndTodoBoundaries(t *testing.T) {
	f := setup(t)
	bob, _ := f.second()
	p := f.req("POST", "/notes/pages/create", f.Token, map[string]any{"title": "Notes"}, 201)
	id := int64(p["id"].(float64))
	path := fmt.Sprintf("/notes/pages/%d", id)
	f.req("GET", path, bob, nil, 404)
	b := f.req("POST", path+"/blocks/create", f.Token, map[string]any{"type": "table", "table_data": map[string]any{"columns": []string{"Name"}, "rows": [][]string{{"Alice"}}}}, 201)
	block := int64(b["id"].(float64))
	f.req("PUT", fmt.Sprintf("/notes/blocks/%d/table", block), bob, map[string]any{"rows": []any{}}, 404)
	f.req("PUT", fmt.Sprintf("/notes/blocks/%d/table", block), f.Token, map[string]any{"rows": [][]string{{"Bob"}}}, 200)
	f.req("POST", path+"/tags/add", f.Token, map[string]any{"name": "Go"}, 201)
	p = f.req("GET", path, f.Token, nil, 200)
	if len(p["blocks"].([]any)) != 1 || len(p["tags"].([]any)) != 1 {
		t.Fatal(p)
	}
	f.req("PUT", path+"/update", f.Token, map[string]any{"parent_id": id}, 400)
	list := f.req("POST", "/todos/lists/create", f.Token, map[string]any{"name": "Work"}, 201)
	lid := int64(list["id"].(float64))
	f.req("POST", "/todos/create", bob, map[string]any{"title": "x", "todo_list": lid}, 404)
	todo := f.req("POST", "/todos/create", f.Token, map[string]any{"title": "Write Go", "todo_list": lid}, 201)
	tid := int64(todo["id"].(float64))
	f.req("PUT", fmt.Sprintf("/todos/%d/update", tid), f.Token, map[string]any{"completed": true, "order": 0}, 200)
	f.req("DELETE", fmt.Sprintf("/todos/lists/%d/delete", lid), f.Token, nil, 200)
	f.req("GET", fmt.Sprintf("/todos/%d", tid), f.Token, nil, 404)
	f.req("DELETE", path+"/delete", f.Token, nil, 200)
	var n int64
	f.s.DB.Model(&models.BlockTable{}).Count(&n)
	if n != 0 {
		t.Fatal("orphan table", n)
	}
}
func TestChatsWebsocketAndMembership(t *testing.T) {
	f := setup(t)
	bob, bid := f.second()
	f.req("POST", "/chats/1/send", bob, map[string]any{"text": "intrusion"}, 403)
	c := f.req("GET", fmt.Sprintf("/chats/direct/%d", bid), f.Token, nil, 200)
	id := int64(c["id"].(float64))
	chats := f.req("GET", "/chats/me", f.Token, nil, 200)["chats"].([]any)
	found := false
	for _, item := range chats {
		chat := item.(map[string]any)
		if chat["id"] != c["id"] {
			continue
		}
		participant := chat["other_participant"].(map[string]any)
		if participant["id"] != float64(bid) || chat["target_user"] != nil {
			t.Fatal(chat)
		}
		found = true
	}
	if !found {
		t.Fatal("direct chat missing from chat list")
	}
	path := fmt.Sprintf("/ws/chat/%d/?token=%s", id, url.QueryEscape(bob))
	ws, _, e := Websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(f.http.URL, "http")+path, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer ws.Close()
	f.req("POST", fmt.Sprintf("/chats/%d/send", id), f.Token, map[string]any{"text": "Hello Bob"}, 200)
	_ = ws.SetReadDeadline(time.Now().Add(3 * time.Second))
	var event map[string]any
	if e = ws.ReadJSON(&event); e != nil {
		t.Fatal(e)
	}
	if event["type"] != "new_message" {
		t.Fatal(event)
	}
	firstMessageID := event["message"].(map[string]any)["id"]
	if e = ws.WriteJSON(map[string]any{"type": "send_message", "text": "Hi Alice"}); e != nil {
		t.Fatal(e)
	}
	if e = ws.ReadJSON(&event); e != nil {
		t.Fatal(e)
	}
	c = f.req("GET", fmt.Sprintf("/chats/direct/%d", bid), f.Token, nil, 200)
	if len(c["messages"].([]any)) != 2 {
		t.Fatal(c)
	}
	f.req("POST", fmt.Sprintf("/chats/%d/read", id), bob, map[string]any{}, 200)
	if e = ws.ReadJSON(&event); e != nil {
		t.Fatal(e)
	}
	if ids, ok := event["message_ids"].([]any); event["type"] != "read_receipt" || !ok || len(ids) != 1 || ids[0] != firstMessageID {
		t.Fatal(event)
	}
}
func TestLocalUploadAndPrivateVideo(t *testing.T) {
	f := setup(t)
	bob, _ := f.second()
	u := f.req("POST", "/videos/create_presigned_url", f.Token, map[string]any{"filename": "movie.mp4", "content_type": "video/mp4"}, 200)
	req, _ := http.NewRequest("PUT", u["url"].(string), strings.NewReader("video content"))
	req.Header.Set("Authorization", "Bearer "+f.Token)
	res, e := f.http.Client().Do(req)
	if e != nil {
		t.Fatal(e)
	}
	res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatal(res.Status)
	}
	f.req("POST", "/videos/complete_upload", bob, map[string]any{"upload_id": u["upload_id"], "key": u["key"]}, 404)
	Done := f.req("POST", "/videos/complete_upload", f.Token, map[string]any{"upload_id": u["upload_id"], "key": u["key"], "title": "Private"}, 200)
	id := int64(Done["id"].(float64))
	f.req("GET", fmt.Sprintf("/videos/%d", id), "", nil, 403)
	f.req("GET", fmt.Sprintf("/videos/%d", id), f.Token, nil, 200)
	f.req("POST", "/videos/complete_upload", f.Token, map[string]any{"upload_id": u["upload_id"], "key": u["key"]}, 409)
	r, e := f.http.Client().Get(Done["location"].(string))
	if e != nil {
		t.Fatal(e)
	}
	r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatal(r.Status)
	}
	parsed, _ := url.Parse(Done["location"].(string))
	parsed.RawQuery = ""
	r, e = f.http.Client().Get(parsed.String())
	if e != nil {
		t.Fatal(e)
	}
	r.Body.Close()
	if r.StatusCode != 403 {
		t.Fatal(r.Status)
	}
	f.req("POST", "/videos/api/save_playback", f.Token, map[string]any{"video_id": id, "playback_time": 12.5}, 200)
	p := f.req("GET", fmt.Sprintf("/videos/api/get_playback_time?video_id=%d", id), f.Token, nil, 200)
	if p["playback_time"] != 12.5 {
		t.Fatal(p)
	}
}
func TestCommunityModeration(t *testing.T) {
	f := setup(t)
	bob, _ := f.second()
	c := f.req("POST", "/communities/create", f.Token, map[string]any{"name": "Go Readers"}, 201)
	id := int64(c["id"].(float64))
	path := fmt.Sprintf("/communities/%d", id)
	f.req("GET", "/communities/@go-readers", "", nil, 200)
	f.req("POST", path+"/join", bob, map[string]any{}, 200)
	p := f.req("POST", "/posts/create", bob, map[string]any{"title": "Discussion"}, 201)
	cp := f.req("POST", path+"/posts/request", bob, map[string]any{"post_id": p["id"]}, 201)
	cid := int64(cp["id"].(float64))
	f.req("POST", fmt.Sprintf("%s/posts/%d/approve", path, cid), bob, map[string]any{}, 403)
	f.req("POST", fmt.Sprintf("%s/posts/%d/approve", path, cid), f.Token, map[string]any{}, 200)
	Posts := f.req("GET", path+"/posts", "", nil, 200)
	if len(Posts["posts"].([]any)) != 1 {
		t.Fatal(Posts)
	}
	f.req("DELETE", path+"/delete", bob, nil, 403)
}
func TestInvalidInputsAndMissingServices(t *testing.T) {
	f := setup(t)
	f.req("GET", "/users/signup", "", nil, 405)
	f.req("POST", "/posts/create", f.Token, map[string]any{"description": strings.Repeat("a", 101)}, 400)
	f.req("POST", "/users/password-reset", "", map[string]any{"email": "alice@example.com"}, 503)
	f.req("GET", "/queries/brave/search?q=go", "", nil, 503)
	f.req("GET", "/admin", f.Token, nil, 403)
	f.req("GET", "/maintenance/delete-tables", f.Token, nil, 405)
	req, _ := http.NewRequest("POST", f.http.URL+"/posts/create", strings.NewReader("{broken"))
	req.Header.Set("Authorization", "Bearer "+f.Token)
	res, e := f.http.Client().Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	if res.StatusCode != 400 {
		t.Fatal(res.Status)
	}
}

func TestOriginalRouteCoverage(t *testing.T) {
	raw, err := os.ReadFile("../docs/source-routes.json")
	if err != nil {
		t.Fatal(err)
	}
	var source []struct {
		Path   string `json:"path"`
		Source string `json:"source"`
	}
	if err = json.Unmarshal(raw, &source); err != nil {
		t.Fatal(err)
	}
	s := New(nil, config.Config{})
	routes := map[string]bool{}
	normalize := func(path string) string {
		return regexp.MustCompile(`\{[^}]+\}|<[^>]+>`).ReplaceAllString(strings.TrimRight(path, "/"), "{}")
	}
	err = chi.Walk(s.Router, func(method, path string, h http.Handler, middleware ...func(http.Handler) http.Handler) error {
		routes[normalize(path)] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range source {
		if !routes[normalize(r.Path)] {
			t.Errorf("Missing route %s (%s)", r.Path, r.Source)
		}
	}
}
