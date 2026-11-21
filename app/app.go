package app

import (
	"litloop/backend/admin"
	"litloop/backend/albums"
	"litloop/backend/artists"
	authcookie "litloop/backend/auth_cookie"
	"litloop/backend/catalog"
	"litloop/backend/chats"
	"litloop/backend/comments"
	"litloop/backend/communities"
	"litloop/backend/internal/config"
	"litloop/backend/internal/httpapi"
	"litloop/backend/jobs"
	"litloop/backend/links"
	"litloop/backend/media"
	"litloop/backend/memes"
	"litloop/backend/movies"
	"litloop/backend/notes"
	"litloop/backend/notifications"
	"litloop/backend/photos"
	"litloop/backend/playlists"
	"litloop/backend/posts"
	queries "litloop/backend/queries"
	"litloop/backend/todos"
	"litloop/backend/tracks"
	"litloop/backend/users"
	"litloop/backend/videos"
	"litloop/backend/views"
	"litloop/backend/websites"

	"gorm.io/gorm"
)

func New(db *gorm.DB, c config.Config) *httpapi.Server {
	s := httpapi.NewServer(db, c)
	s.Query = queries.New(s).Query
	adminHandler := admin.New(s)
	albumsHandler := albums.New(s)
	artistsHandler := artists.New(s)
	catalogHandler := catalog.New(s)
	chatsHandler := chats.New(s)
	commentsHandler := comments.New(s)
	communitiesHandler := communities.New(s)
	jobsHandler := jobs.New(s)
	linksHandler := links.New(s)
	mediaHandler := media.New(s)
	memesHandler := memes.New(s)
	moviesHandler := movies.New(s)
	notesHandler := notes.New(s)
	notificationsHandler := notifications.New(s)
	photosHandler := photos.New(s)
	playlistsHandler := playlists.New(s)
	postsHandler := posts.New(s)
	todosHandler := todos.New(s)
	tracksHandler := tracks.New(s)
	usersHandler := users.New(s)
	videosHandler := videos.New(s)
	viewsHandler := views.New(s)
	websitesHandler := websites.New(s)
	s.Admin = adminHandler.Admin
	s.AlbumJSON = albumsHandler.AlbumJSON
	s.AlbumLike = albumsHandler.AlbumLike
	s.Application = jobsHandler.Application
	s.ArtistJSON = artistsHandler.ArtistJSON
	s.Asset = mediaHandler.Asset
	s.Attachment = mediaHandler.Attachment
	s.Authenticate = usersHandler.Authenticate
	s.BlockJSON = notesHandler.BlockJSON
	s.BlockWrite = notesHandler.BlockWrite
	s.CancelUpload = mediaHandler.CancelUpload
	s.Catalog = catalogHandler.Catalog
	s.ChatAccess = chatsHandler.ChatAccess
	s.ChatAction = chatsHandler.ChatAction
	s.ChatJSON = chatsHandler.ChatJSON
	s.ChatList = chatsHandler.ChatList
	s.MessagesAfter = chatsHandler.MessagesAfter
	s.Claims = usersHandler.Claims
	s.CommentCreate = commentsHandler.CommentCreate
	s.CommentJSON = commentsHandler.CommentJSON
	s.Comments = commentsHandler.Comments
	s.Communities = communitiesHandler.Communities
	s.Community = communitiesHandler.Community
	s.CommunityJSON = communitiesHandler.CommunityJSON
	s.CommunityJoin = communitiesHandler.CommunityJoin
	s.CommunityPostJSON = communitiesHandler.CommunityPostJSON
	s.CommunityPosts = communitiesHandler.CommunityPosts
	s.CommunityRole = communitiesHandler.CommunityRole
	s.CommunityWrite = communitiesHandler.CommunityWrite
	s.Companies = jobsHandler.Companies
	s.CreateGroup = chatsHandler.CreateGroup
	s.DirectChat = chatsHandler.DirectChat
	s.EncodeVideo = videosHandler.EncodeVideo
	s.FindTrack = tracksHandler.FindTrack
	s.ForgotPassword = usersHandler.ForgotPassword
	s.Gemini = chatsHandler.Gemini
	s.GoogleClient = mediaHandler.GoogleClient
	s.GroupMember = chatsHandler.GroupMember
	s.ImportLinks = linksHandler.ImportLinks
	s.ImportMovies = moviesHandler.ImportMovies
	s.Impressions = viewsHandler.Impressions
	s.JWT = usersHandler.JWT
	s.LaunchSpot = videosHandler.LaunchSpot
	s.LinkJSON = linksHandler.LinkJSON
	s.LinkPost = postsHandler.LinkPost
	s.LinkSave = linksHandler.LinkSave
	s.Links = linksHandler.Links
	s.LocalURL = mediaHandler.LocalURL
	s.Logout = usersHandler.Logout
	s.Maintenance = adminHandler.Maintenance
	s.MediaJSON = mediaHandler.MediaJSON
	s.MediaList = mediaHandler.MediaList
	s.MediaServe = mediaHandler.MediaServe
	s.MediaURL = mediaHandler.MediaURL
	s.Member = chatsHandler.Member
	s.Memes = memesHandler.Memes
	s.MessageJSON = chatsHandler.MessageJSON
	s.MovieAdd = moviesHandler.MovieAdd
	s.MovieFeed = moviesHandler.MovieFeed
	s.MovieLike = moviesHandler.MovieLike
	s.Movies = moviesHandler.Movies
	s.MultipartUpload = mediaHandler.MultipartUpload
	s.NativeMedia = mediaHandler.NativeMedia
	s.NativeURL = mediaHandler.NativeURL
	s.Notifications = notificationsHandler.Notifications
	s.OAuth = usersHandler.OAuth
	s.OwnedPage = notesHandler.OwnedPage
	s.PageJSON = notesHandler.PageJSON
	s.PageTags = notesHandler.PageTags
	s.PageWrite = notesHandler.PageWrite
	s.Pages = notesHandler.Pages
	s.PendingUpload = mediaHandler.PendingUpload
	s.PhotoAlbum = photosHandler.PhotoAlbum
	s.Playback = videosHandler.Playback
	s.Playlist = playlistsHandler.Playlist
	s.PlaylistJSON = playlistsHandler.PlaylistJSON
	s.PostFiles = postsHandler.PostFiles
	s.PostJSON = postsHandler.PostJSON
	s.PostLike = postsHandler.PostLike
	s.PostWrite = postsHandler.PostWrite
	s.Posts = postsHandler.Posts
	s.PrivateWatchlist = usersHandler.PrivateWatchlist
	s.Refresh = usersHandler.Refresh
	s.Reorder = todosHandler.Reorder
	s.ResetPassword = usersHandler.ResetPassword
	s.Resumes = jobsHandler.Resumes
	s.Role = usersHandler.Role
	s.SaveLink = linksHandler.SaveLink
	s.SaveMessage = chatsHandler.SaveMessage
	s.SaveUploadedFile = mediaHandler.SaveUploadedFile
	s.Search = websitesHandler.Search
	s.SendMessage = chatsHandler.SendMessage
	s.SignMedia = mediaHandler.SignMedia
	s.Signin = usersHandler.Signin
	s.Signup = usersHandler.Signup
	s.StorageFor = mediaHandler.StorageFor
	s.TaskStatus = jobsHandler.TaskStatus
	s.TodoJSON = todosHandler.TodoJSON
	s.TodoListWrite = todosHandler.TodoListWrite
	s.TodoLists = todosHandler.TodoLists
	s.TodoWrite = todosHandler.TodoWrite
	s.Todos = todosHandler.Todos
	s.Tokens = usersHandler.Tokens
	s.TrackJSON = tracksHandler.TrackJSON
	s.TrackWrite = tracksHandler.TrackWrite
	s.Transcription = chatsHandler.Transcription
	s.TranscriptionCallback = chatsHandler.TranscriptionCallback
	s.UploadComplete = mediaHandler.UploadComplete
	s.UploadInitiate = mediaHandler.UploadInitiate
	s.UploadPart = mediaHandler.UploadPart
	s.UploadProvider = mediaHandler.UploadProvider
	s.UploadPut = mediaHandler.UploadPut
	s.UserDetail = usersHandler.UserDetail
	s.UserList = usersHandler.UserList
	s.UserMedia = usersHandler.UserMedia
	s.UserMediaKind = usersHandler.UserMediaKind
	s.Vacancies = jobsHandler.Vacancies
	s.VideoAccess = videosHandler.VideoAccess
	s.View = viewsHandler.View
	s.VoiceJSON = chatsHandler.VoiceJSON
	s.VoIP = chatsHandler.VoIP
	s.Watchlist = moviesHandler.Watchlist
	s.WatchlistFor = moviesHandler.WatchlistFor
	s.Websocket = chatsHandler.Websocket
	admin.Register(s)
	albums.Register(s)
	artists.Register(s)
	authcookie.Register(s)
	chats.Register(s)
	comments.Register(s)
	communities.Register(s)
	jobs.Register(s)
	links.Register(s)
	media.Register(s)
	memes.Register(s)
	movies.Register(s)
	notes.Register(s)
	notifications.Register(s)
	photos.Register(s)
	playlists.Register(s)
	posts.Register(s)
	queries.Register(s)
	todos.Register(s)
	tracks.Register(s)
	users.Register(s)
	videos.Register(s)
	views.Register(s)
	websites.Register(s)
	Register(s)
	return s
}
