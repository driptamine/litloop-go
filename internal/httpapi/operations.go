package httpapi

import (
	"context"
	"litloop/backend/internal/models"
	"litloop/backend/internal/storage"
	"mime/multipart"
	"net/http"
	"time"

	JWT "github.com/golang-jwt/jwt/v5"

	"gorm.io/gorm"
)

type Operations struct {
	Query                 func(*Request) any
	Admin                 func(q *Request) any
	AlbumJSON             func(a models.Album) any
	AlbumLike             func(q *Request) any
	Application           func(q *Request) any
	ArtistJSON            func(a models.Artist) any
	Asset                 func(q *Request) any
	Attachment            func(q *Request) any
	Authenticate          func(r *http.Request) *models.User
	BlockJSON             func(b models.Block) map[string]any
	BlockWrite            func(q *Request) any
	CancelUpload          func(q *Request) any
	Catalog               func(q *Request) any
	ChatAccess            func(q *Request, id int64) models.Chat
	ChatAction            func(q *Request) any
	ChatJSON              func(c models.Chat, user int64, detail bool) map[string]any
	ChatList              func(q *Request) any
	MessagesAfter         func(q *Request) any
	Claims                func(raw string, kind string) JWT.MapClaims
	CommentCreate         func(q *Request) any
	CommentJSON           func(c models.Comment) map[string]any
	Comments              func(q *Request) any
	Communities           func(q *Request) any
	Community             func(q *Request) models.Community
	CommunityJSON         func(c models.Community, u *models.User) map[string]any
	CommunityJoin         func(q *Request) any
	CommunityPostJSON     func(cp models.CommunityPost, u *models.User) map[string]any
	CommunityPosts        func(q *Request) any
	CommunityRole         func(c models.Community, u *models.User, moderator bool)
	CommunityWrite        func(q *Request) any
	Companies             func(q *Request) any
	CreateGroup           func(q *Request) any
	DirectChat            func(q *Request) any
	EncodeVideo           func(q *Request) any
	FindTrack             func(key string) models.Track
	ForgotPassword        func(q *Request) any
	Gemini                func(q *Request) any
	GoogleClient          func(ctx context.Context) *http.Client
	GroupMember           func(q *Request) any
	ImportLinks           func(q *Request) any
	ImportMovies          func(q *Request) any
	Impressions           func(q *Request) any
	JWT                   func(user int64, kind string, ttl time.Duration) string
	LaunchSpot            func(q *Request) any
	LinkJSON              func(l models.Link) any
	LinkPost              func(tx *gorm.DB, p models.Post, b map[string]any, u int64)
	LinkSave              func(q *Request) any
	Links                 func(q *Request) any
	LocalURL              func(key string) string
	Logout                func(q *Request) any
	Maintenance           func(q *Request) any
	MediaJSON             func(v any) map[string]any
	MediaList             func(v any) any
	MediaServe            func(q *Request) any
	MediaURL              func(r, g, a string) string
	Member                func(chat, user int64) bool
	Memes                 func(q *Request) any
	MessageJSON           func(m models.Message) map[string]any
	MovieAdd              func(q *Request) any
	MovieFeed             func(q *Request) any
	MovieLike             func(q *Request) any
	Movies                func(q *Request) any
	MultipartUpload       func(q *Request) (string, string, string)
	NativeMedia           func(q *Request) any
	NativeURL             func(key string) string
	Notifications         func(q *Request) any
	OAuth                 func(q *Request) any
	OwnedPage             func(q *Request, id int64) models.Page
	PageJSON              func(p models.Page, detail bool) map[string]any
	PageTags              func(q *Request) any
	PageWrite             func(q *Request) any
	Pages                 func(q *Request) any
	PendingUpload         func(q *Request) models.Upload
	PhotoAlbum            func(q *Request) any
	Playback              func(q *Request) any
	Playlist              func(q *Request) any
	PlaylistJSON          func(p models.Playlist) map[string]any
	PostFiles             func(q *Request, tx *gorm.DB, p models.Post, b map[string]any)
	PostJSON              func(p models.Post, u *models.User) map[string]any
	PostLike              func(q *Request) any
	PostWrite             func(q *Request) any
	Posts                 func(q *Request) any
	PrivateWatchlist      func(q *Request, u models.User) any
	Refresh               func(q *Request) any
	Reorder               func(q *Request) any
	ResetPassword         func(q *Request) any
	Resumes               func(q *Request) any
	Role                  func(q *Request) any
	SaveLink              func(tx *gorm.DB, u int64, b map[string]any) models.Link
	SaveMessage           func(chat int64, u models.User, b map[string]any) models.Message
	SaveUploadedFile      func(q *Request, f multipart.File, head *multipart.FileHeader) (string, string, string)
	Search                func(q *Request) any
	SendMessage           func(q *Request) any
	SignMedia             func(key, exp string) string
	Signin                func(q *Request) any
	Signup                func(q *Request) any
	StorageFor            func(provider string) *storage.Store
	TaskStatus            func(q *Request) any
	TodoJSON              func(t models.Todo) any
	TodoListWrite         func(q *Request) any
	TodoLists             func(q *Request) any
	TodoWrite             func(q *Request) any
	Todos                 func(q *Request) any
	Tokens                func(u models.User) map[string]any
	TrackJSON             func(t models.Track) any
	TrackWrite            func(q *Request) any
	Transcription         func(q *Request) any
	TranscriptionCallback func(q *Request) any
	UploadComplete        func(q *Request) any
	UploadInitiate        func(q *Request) any
	UploadPart            func(q *Request) any
	UploadProvider        func(q *Request) string
	UploadPut             func(q *Request) any
	UserDetail            func(q *Request) any
	UserList              func(q *Request) any
	UserMedia             func(q *Request) any
	UserMediaKind         func(q *Request, kind string) any
	Vacancies             func(q *Request) any
	VideoAccess           func(q *Request, id int64) models.Video
	View                  func(q *Request) any
	VoiceJSON             func(v models.VoiceMessage) any
	VoIP                  func(q *Request, b map[string]any) bool
	Watchlist             func(q *Request) any
	WatchlistFor          func(q *Request, uid int64, ratings bool) any
	Websocket             func(q *Request) any
}
