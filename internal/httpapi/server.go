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
	DB	*gorm.DB
	Config	config.Config
	Hub	*Hub
	Client	*http.Client
	Router	chi.Router
}
type Endpoint func(*Request) any
type Request struct {
	S	*Server
	W	http.ResponseWriter
	R	*http.Request
	User	*models.User
	Status	int
	body	map[string]any
}
type Error struct {
	Code	int
	Message	string
}

func Fail(code int, msg string)	{ panic(Error{code, msg}) }
func Check(err error) {
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			Fail(404, "Not found")
		}
		if errors.Is(err, gorm.ErrForeignKeyViolated) {
			Fail(400, "Invalid related resource")
		}
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			Fail(409, "Already exists")
		}
		panic(err)
	}
}
func NewServer(db *gorm.DB, appConfig config.Config) *Server {
	server := &Server{DB: db, Config: appConfig, Hub: NewHub(), Client: &http.Client{Timeout: 30 * time.Second}}
	if appConfig.RedisURL != "" {
		if err := server.Hub.ConnectRedis(appConfig.RedisURL); err != nil {
			panic(fmt.Errorf("Redis: %w", err))
		}
	}
	router := chi.NewRouter()
	server.Router = router
	router.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer)
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:	appConfig.Origins,
		AllowedMethods:	[]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		// The deployed frontend's search client sends a numeric header named "0".
		AllowedHeaders:		[]string{"0", "Accept-Version", "Authorization", "Content-Type", "User-Agent"},
		AllowCredentials:	true,
		MaxAge:			300,
	}))
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, httpReq *http.Request) {
			if httpReq.URL.Path != "/" {
				httpReq.URL.Path = strings.TrimRight(httpReq.URL.Path, "/")
			}
			next.ServeHTTP(writer, httpReq)
		})
	})

	server.Router.NotFound(func(writer http.ResponseWriter, httpReq *http.Request) {
		Write(writer, 404, map[string]any{"error": "Not found"})
	})
	server.Router.MethodNotAllowed(func(writer http.ResponseWriter, httpReq *http.Request) {
		Write(writer, 405, map[string]any{"error": "Method not allowed"})
	})
	return server
}
func (server *Server) ServeHTTP(writer http.ResponseWriter, httpReq *http.Request) {
	server.Router.ServeHTTP(writer, httpReq)
}
func (server *Server) Route(methods, path string, auth bool, fn Endpoint) {
	for _, method := range strings.Split(methods, ",") {
		server.Router.MethodFunc(method, path, func(writer http.ResponseWriter, httpReq *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					switch err := recovered.(type) {
					case Error:
						Write(writer, err.Code, map[string]any{"error": err.Message})
					default:
						slog.Error("request failed", "method", httpReq.Method, "path", httpReq.URL.Path, "error", fmt.Sprint(recovered))
						Write(writer, 500, map[string]any{"error": "Internal server error"})
					}
				}
			}()
			req := &Request{S: server, W: writer, R: httpReq, Status: 200}
			if httpReq.Header.Get("Authorization") == "" && httpReq.Method != "GET" && httpReq.Method != "HEAD" {
				if _, err := httpReq.Cookie("access_token"); err == nil {
					allowed := false
					for _, origin := range server.Config.Origins {
						if origin == httpReq.Header.Get("Origin") {
							allowed = true
						}
					}
					if !allowed {
						Fail(403, "Cookie authentication requires an allowed Origin")
					}
				}
			}
			req.User = server.Authenticate(httpReq)
			if auth && req.User == nil {
				Fail(401, "Authentication required")
			}
			defer func() {
				if httpReq.MultipartForm != nil {
					_ = httpReq.MultipartForm.RemoveAll()
				}
			}()
			out := fn(req)
			if req.Status != 0 {
				Write(writer, req.Status, out)
			}
		})
	}
}
func Write(writer http.ResponseWriter, code int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(code)
	if code != 204 {
		_ = json.NewEncoder(writer).Encode(value)
	}
}
func (req *Request) Body() map[string]any {
	if req.body != nil {
		return req.body
	}
	if strings.HasPrefix(req.R.Header.Get("Content-Type"), "multipart/form-data") || strings.HasPrefix(req.R.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		req.R.Body = http.MaxBytesReader(req.W, req.R.Body, 64<<20)
		if strings.HasPrefix(req.R.Header.Get("Content-Type"), "multipart/") {
			if req.R.ParseMultipartForm(8<<20) != nil {
				Fail(400, "Invalid multipart request")
			}
		} else {
			if req.R.ParseForm() != nil {
				Fail(400, "Invalid form")
			}
		}
		req.body = map[string]any{}
		for key, values := range req.R.PostForm {
			if len(values) == 0 {
				continue
			}
			if strings.HasSuffix(key, "_ids") || key == "attachments" {
				if len(values) > 1 {
					list := []any{}
					for _, value := range values {
						list = append(list, value)
					}
					req.body[key] = list
				} else {
					var list []any
					if json.Unmarshal([]byte(values[0]), &list) != nil {
						for _, value := range strings.Split(values[0], ",") {
							list = append(list, strings.TrimSpace(value))
						}
					}
					req.body[key] = list
				}
			} else {
				req.body[key] = values[0]
			}
		}
		return req.body
	}
	req.R.Body = http.MaxBytesReader(req.W, req.R.Body, 4<<20)
	decoder := json.NewDecoder(req.R.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&req.body); err != nil || req.body == nil {
		Fail(400, "Invalid JSON object")
	}
	var tail any
	if decoder.Decode(&tail) != io.EOF {
		Fail(400, "Invalid JSON")
	}
	return req.body
}
func (req *Request) ID(key string) int64 {
	parsed, err := strconv.ParseInt(chi.URLParam(req.R, key), 10, 64)
	if err != nil || parsed <= 0 {
		Fail(400, "Invalid ID")
	}
	return parsed
}
func (req *Request) Param(key string) string	{ return chi.URLParam(req.R, key) }
func (req *Request) Admin() {
	if req.User == nil || !req.User.IsSuperuser {
		Fail(403, "Forbidden")
	}
}
func (req *Request) Tx(fn func(*gorm.DB) error)	{ Check(req.S.DB.Transaction(fn)) }
func Ptr[Model any](value Model) *Model		{ return &value }
func Val[Model any](ptr *Model) (value Model) {
	if ptr != nil {
		return *ptr
	}
	return
}
func Str(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
func Num(value any) int64 {
	parsed, err := strconv.ParseInt(Str(value), 10, 64)
	if err != nil {
		Fail(400, "Expected an integer")
	}
	return parsed
}
func Boolean(value any) bool {
	flag, ok := value.(bool)
	if !ok {
		Fail(400, "Expected a boolean")
	}
	return flag
}
func IDs(value any) []int64 {
	if value == nil {
		return []int64{}
	}
	items, ok := value.([]any)
	if !ok {
		Fail(400, "Expected an array")
	}
	ids := []int64{}
	for _, item := range items {
		parsed := Num(item)
		if parsed <= 0 {
			Fail(400, "Invalid ID")
		}
		ids = append(ids, parsed)
	}
	return ids
}
func Token() string {
	raw := make([]byte, 24)
	_, err := rand.Read(raw)
	Check(err)
	return hex.EncodeToString(raw)
}
func Object(value any) map[string]any {
	encoded, err := json.Marshal(value)
	Check(err)
	var out map[string]any
	Check(json.Unmarshal(encoded, &out))
	return out
}
func Get[Model any](db *gorm.DB, id any) Model {
	var result Model
	Check(db.First(&result, "id = ?", id).Error)
	return result
}
func All[Model any](db *gorm.DB) []Model {
	results := []Model{}
	Check(db.Find(&results).Error)
	return results
}
func (req *Request) Page(db *gorm.DB, dst any, serialize func(any) any) any {
	return req.PageSize(db, dst, serialize, 10)
}
func (req *Request) PageSize(db *gorm.DB, dst any, serialize func(any) any, defaultSize int) any {
	page, _ := strconv.Atoi(req.R.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	size, _ := strconv.Atoi(req.R.URL.Query().Get("page_size"))
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
