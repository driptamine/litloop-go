package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"litloop/backend/internal/config"
	"litloop/backend/internal/models"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"gorm.io/gorm"
)

type Server struct {
	Operations
	DB     *gorm.DB
	Config config.Config
	Hub    *Hub
	Client *http.Client
	Router chi.Router
}
type Endpoint func(*Request) any
type Request struct {
	S      *Server
	W      http.ResponseWriter
	R      *http.Request
	User   *models.User
	Status int
	body   map[string]any
}
type Error struct {
	Code    int
	Message string
}

func Fail(code int, msg string) { panic(Error{code, msg}) }
func Check(e error) {
	if e != nil {
		if errors.Is(e, gorm.ErrRecordNotFound) {
			Fail(404, "Not found")
		}
		if errors.Is(e, gorm.ErrForeignKeyViolated) {
			Fail(400, "Invalid related resource")
		}
		if errors.Is(e, gorm.ErrDuplicatedKey) {
			Fail(409, "Already exists")
		}
		panic(e)
	}
}
func NewServer(db *gorm.DB, c config.Config) *Server {
	s := &Server{DB: db, Config: c, Hub: NewHub(), Client: &http.Client{Timeout: 30 * time.Second}}
	if c.RedisURL != "" {
		if e := s.Hub.ConnectRedis(c.RedisURL); e != nil {
			panic(fmt.Errorf("Redis: %w", e))
		}
	}
	r := chi.NewRouter()
	s.Router = r
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: c.Origins,
		AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		// The deployed frontend's search client sends a numeric header named "0".
		AllowedHeaders:   []string{"0", "Accept-Version", "Authorization", "Content-Type", "User-Agent"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/" {
				r.URL.Path = strings.TrimRight(r.URL.Path, "/")
			}
			next.ServeHTTP(w, r)
		})
	})

	s.Router.NotFound(func(w http.ResponseWriter, r *http.Request) { Write(w, 404, map[string]any{"error": "Not found"}) })
	s.Router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		Write(w, 405, map[string]any{"error": "Method not allowed"})
	})
	return s
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.Router.ServeHTTP(w, r) }
func (s *Server) Route(methods, path string, auth bool, fn Endpoint) {
	for _, method := range strings.Split(methods, ",") {
		s.Router.MethodFunc(method, path, func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					switch e := v.(type) {
					case Error:
						Write(w, e.Code, map[string]any{"error": e.Message})
					default:
						slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", fmt.Sprint(v))
						Write(w, 500, map[string]any{"error": "Internal server error"})
					}
				}
			}()
			q := &Request{S: s, W: w, R: r, Status: 200}
			if r.Header.Get("Authorization") == "" && r.Method != "GET" && r.Method != "HEAD" {
				if _, err := r.Cookie("access_token"); err == nil {
					allowed := false
					for _, origin := range s.Config.Origins {
						if origin == r.Header.Get("Origin") {
							allowed = true
						}
					}
					if !allowed {
						Fail(403, "Cookie authentication requires an allowed Origin")
					}
				}
			}
			q.User = s.Authenticate(r)
			if auth && q.User == nil {
				Fail(401, "Authentication required")
			}
			defer func() {
				if r.MultipartForm != nil {
					_ = r.MultipartForm.RemoveAll()
				}
			}()
			out := fn(q)
			if q.Status != 0 {
				Write(w, q.Status, out)
			}
		})
	}
}
func Write(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if code != 204 {
		_ = json.NewEncoder(w).Encode(v)
	}
}
func (q *Request) Body() map[string]any {
	if q.body != nil {
		return q.body
	}
	if strings.HasPrefix(q.R.Header.Get("Content-Type"), "multipart/form-data") || strings.HasPrefix(q.R.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		q.R.Body = http.MaxBytesReader(q.W, q.R.Body, 64<<20)
		if strings.HasPrefix(q.R.Header.Get("Content-Type"), "multipart/") {
			if q.R.ParseMultipartForm(8<<20) != nil {
				Fail(400, "Invalid multipart request")
			}
		} else {
			if q.R.ParseForm() != nil {
				Fail(400, "Invalid form")
			}
		}
		q.body = map[string]any{}
		for key, values := range q.R.PostForm {
			if len(values) == 0 {
				continue
			}
			if strings.HasSuffix(key, "_ids") || key == "attachments" {
				if len(values) > 1 {
					list := []any{}
					for _, v := range values {
						list = append(list, v)
					}
					q.body[key] = list
				} else {
					var list []any
					if json.Unmarshal([]byte(values[0]), &list) != nil {
						for _, v := range strings.Split(values[0], ",") {
							list = append(list, strings.TrimSpace(v))
						}
					}
					q.body[key] = list
				}
			} else {
				q.body[key] = values[0]
			}
		}
		return q.body
	}
	q.R.Body = http.MaxBytesReader(q.W, q.R.Body, 4<<20)
	d := json.NewDecoder(q.R.Body)
	d.UseNumber()
	if e := d.Decode(&q.body); e != nil || q.body == nil {
		Fail(400, "Invalid JSON object")
	}
	var tail any
	if d.Decode(&tail) != io.EOF {
		Fail(400, "Invalid JSON")
	}
	return q.body
}
func (q *Request) ID(key string) int64 {
	v, e := strconv.ParseInt(chi.URLParam(q.R, key), 10, 64)
	if e != nil || v <= 0 {
		Fail(400, "Invalid ID")
	}
	return v
}
func (q *Request) Param(k string) string { return chi.URLParam(q.R, k) }
func (q *Request) Admin() {
	if q.User == nil || !q.User.IsSuperuser {
		Fail(403, "Forbidden")
	}
}
func (q *Request) Tx(fn func(*gorm.DB) error) { Check(q.S.DB.Transaction(fn)) }
func Ptr[T any](v T) *T                       { return &v }
func Val[T any](v *T) (z T) {
	if v != nil {
		return *v
	}
	return
}
func Str(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
func Num(v any) int64 {
	i, e := strconv.ParseInt(Str(v), 10, 64)
	if e != nil {
		Fail(400, "Expected an integer")
	}
	return i
}
func Boolean(v any) bool {
	b, ok := v.(bool)
	if !ok {
		Fail(400, "Expected a boolean")
	}
	return b
}
func IDs(v any) []int64 {
	if v == nil {
		return []int64{}
	}
	a, ok := v.([]any)
	if !ok {
		Fail(400, "Expected an array")
	}
	r := []int64{}
	for _, x := range a {
		n := Num(x)
		if n <= 0 {
			Fail(400, "Invalid ID")
		}
		r = append(r, n)
	}
	return r
}
func Token() string {
	b := make([]byte, 24)
	_, e := rand.Read(b)
	Check(e)
	return hex.EncodeToString(b)
}
func Object(v any) map[string]any {
	b, e := json.Marshal(v)
	Check(e)
	var out map[string]any
	Check(json.Unmarshal(b, &out))
	return out
}
func Get[T any](db *gorm.DB, id any) T { var v T; Check(db.First(&v, "id = ?", id).Error); return v }
func All[T any](db *gorm.DB) []T       { v := []T{}; Check(db.Find(&v).Error); return v }
func (q *Request) Page(db *gorm.DB, dst any, serialize func(any) any) any {
	return q.PageSize(db, dst, serialize, 10)
}
func (q *Request) PageSize(db *gorm.DB, dst any, serialize func(any) any, defaultSize int) any {
	page, _ := strconv.Atoi(q.R.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(q.R.URL.Query().Get("page_size"))
	if size < 1 {
		size = defaultSize
	}
	if size > 100 {
		size = 100
	}
	var count int64
	Check(db.Count(&count).Error)
	Pages := (count + int64(size) - 1) / int64(size)
	if Pages == 0 {
		Pages = 1
	}
	if int64(page) > Pages {
		page = int(Pages)
	}
	Check(db.Offset((page - 1) * size).Limit(size).Find(dst).Error)
	Pages = (count + int64(size) - 1) / int64(size)
	if Pages == 0 {
		Pages = 1
	}
	var next, prev any
	if int64(page) < Pages {
		next = page + 1
	}
	if page > 1 {
		prev = page - 1
	}
	return map[string]any{"total": count, "num_pages": Pages, "current_page": page, "items": serialize(dst), "next": next, "previous": prev}
}
func Owned(actual, expected int64) {
	if actual != expected {
		Fail(403, "Permission denied")
	}
}
