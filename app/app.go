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
	"litloop/backend/push"
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

func New(db *gorm.DB, appConfig config.Config) *httpapi.Server {
	server := httpapi.NewServer(db, appConfig)
	server.Query = queries.New(server).Query
	adminHandler := admin.New(server)
	albumsHandler := albums.New(server)
	artistsHandler := artists.New(server)
	catalogHandler := catalog.New(server)
	chatsHandler := chats.New(server)
	commentsHandler := comments.New(server)
	communitiesHandler := communities.New(server)
	jobsHandler := jobs.New(server)
	linksHandler := links.New(server)
	mediaHandler := media.New(server)
	memesHandler := memes.New(server)
	moviesHandler := movies.New(server)
	notesHandler := notes.New(server)
	notificationsHandler := notifications.New(server)
	pushHandler := push.New(server)
	photosHandler := photos.New(server)
	playlistsHandler := playlists.New(server)
	postsHandler := posts.New(server)
	todosHandler := todos.New(server)
	tracksHandler := tracks.New(server)
	usersHandler := users.New(server)
	videosHandler := videos.New(server)
	viewsHandler := views.New(server)
	websitesHandler := websites.New(server)
	server.Admin = adminHandler.Admin
	server.AlbumJSON = albumsHandler.AlbumJSON
	server.AlbumLike = albumsHandler.AlbumLike
	server.Application = jobsHandler.Application
	server.ArtistJSON = artistsHandler.ArtistJSON
	server.Asset = mediaHandler.Asset
	server.Attachment = mediaHandler.Attachment
	server.Authenticate = usersHandler.Authenticate
	server.BlockJSON = notesHandler.BlockJSON
	server.BlockWrite = notesHandler.BlockWrite
	server.CancelUpload = mediaHandler.CancelUpload
	server.Catalog = catalogHandler.Catalog
	server.ChatAccess = chatsHandler.ChatAccess
	server.ChatAction = chatsHandler.ChatAction
	server.ChatJSON = chatsHandler.ChatJSON
	server.ChatList = chatsHandler.ChatList
	server.ChatTabs = chatsHandler.ChatTabs
	server.ChatTabsWrite = chatsHandler.ChatTabsWrite
	server.Presence = chatsHandler.Presence
	server.CallCommand = chatsHandler.CallCommand
	server.CallState = chatsHandler.CallState
	server.ChangePassword = usersHandler.ChangePassword
	server.GroupVoice = chatsHandler.GroupVoice
	server.IncomingCall = chatsHandler.IncomingCall
	server.MessagesAfter = chatsHandler.MessagesAfter
	server.Claims = usersHandler.Claims
	server.CommentCreate = commentsHandler.CommentCreate
	server.CommentJSON = commentsHandler.CommentJSON
	server.Comments = commentsHandler.Comments
	server.Communities = communitiesHandler.Communities
	server.Community = communitiesHandler.Community
	server.CommunityJSON = communitiesHandler.CommunityJSON
	server.CommunityJoin = communitiesHandler.CommunityJoin
	server.CommunityPostJSON = communitiesHandler.CommunityPostJSON
	server.CommunityPosts = communitiesHandler.CommunityPosts
	server.CommunityRole = communitiesHandler.CommunityRole
	server.CommunityWrite = communitiesHandler.CommunityWrite
	server.Companies = jobsHandler.Companies
	server.CreateGroup = chatsHandler.CreateGroup
	server.DirectChat = chatsHandler.DirectChat
	server.EncodeVideo = videosHandler.EncodeVideo
	server.FindTrack = tracksHandler.FindTrack
	server.ForgotPassword = usersHandler.ForgotPassword
	server.Gemini = chatsHandler.Gemini
	server.GoogleClient = mediaHandler.GoogleClient
	server.GroupMember = chatsHandler.GroupMember
	server.ImportLinks = linksHandler.ImportLinks
	server.ImportMovies = moviesHandler.ImportMovies
	server.Impressions = viewsHandler.Impressions
	server.JWT = usersHandler.JWT
	server.LaunchSpot = videosHandler.LaunchSpot
	server.LinkJSON = linksHandler.LinkJSON
	server.LinkPost = postsHandler.LinkPost
	server.LinkSave = linksHandler.LinkSave
	server.Links = linksHandler.Links
	server.LocalURL = mediaHandler.LocalURL
	server.Logout = usersHandler.Logout
	server.Maintenance = adminHandler.Maintenance
	server.MediaJSON = mediaHandler.MediaJSON
	server.MediaList = mediaHandler.MediaList
	server.MediaServe = mediaHandler.MediaServe
	server.MediaURL = mediaHandler.MediaURL
	server.Member = chatsHandler.Member
	server.Memes = memesHandler.Memes
	server.MessageJSON = chatsHandler.MessageJSON
	server.MovieAdd = moviesHandler.MovieAdd
	server.MovieFeed = moviesHandler.MovieFeed
	server.MovieLike = moviesHandler.MovieLike
	server.Movies = moviesHandler.Movies
	server.MultipartUpload = mediaHandler.MultipartUpload
	server.NativeMedia = mediaHandler.NativeMedia
	server.NativeURL = mediaHandler.NativeURL
	server.Notifications = notificationsHandler.Notifications
	server.OAuth = usersHandler.OAuth
	server.PushSubscribe = pushHandler.Subscribe
	server.PushUnsubscribe = pushHandler.Unsubscribe
	server.PushVAPIDKey = pushHandler.VAPIDKey
	server.OwnedPage = notesHandler.OwnedPage
	server.PageJSON = notesHandler.PageJSON
	server.PageTags = notesHandler.PageTags
	server.PageWrite = notesHandler.PageWrite
	server.Pages = notesHandler.Pages
	server.PendingUpload = mediaHandler.PendingUpload
	server.PhotoAlbum = photosHandler.PhotoAlbum
	server.Playback = videosHandler.Playback
	server.Playlist = playlistsHandler.Playlist
	server.PlaylistJSON = playlistsHandler.PlaylistJSON
	server.PostFiles = postsHandler.PostFiles
	server.PostJSON = postsHandler.PostJSON
	server.PostLike = postsHandler.PostLike
	server.PostWrite = postsHandler.PostWrite
	server.Posts = postsHandler.Posts
	server.PrivateWatchlist = usersHandler.PrivateWatchlist
	server.Refresh = usersHandler.Refresh
	server.Reorder = todosHandler.Reorder
	server.ResetPassword = usersHandler.ResetPassword
	server.Resumes = jobsHandler.Resumes
	server.Role = usersHandler.Role
	server.SaveLink = linksHandler.SaveLink
	server.SaveMessage = chatsHandler.SaveMessage
	server.SaveUploadedFile = mediaHandler.SaveUploadedFile
	server.Search = websitesHandler.Search
	server.SendMessage = chatsHandler.SendMessage
	server.SignMedia = mediaHandler.SignMedia
	server.Signin = usersHandler.Signin
	server.Signup = usersHandler.Signup
	server.StorageFor = mediaHandler.StorageFor
	server.TaskStatus = jobsHandler.TaskStatus
	server.TodoJSON = todosHandler.TodoJSON
	server.TodoListWrite = todosHandler.TodoListWrite
	server.TodoLists = todosHandler.TodoLists
	server.TodoWrite = todosHandler.TodoWrite
	server.Todos = todosHandler.Todos
	server.Tokens = usersHandler.Tokens
	server.TrackJSON = tracksHandler.TrackJSON
	server.TrackWrite = tracksHandler.TrackWrite
	server.Transcription = chatsHandler.Transcription
	server.TranscriptionCallback = chatsHandler.TranscriptionCallback
	server.UploadComplete = mediaHandler.UploadComplete
	server.UploadInitiate = mediaHandler.UploadInitiate
	server.UploadPart = mediaHandler.UploadPart
	server.UploadProvider = mediaHandler.UploadProvider
	server.UploadPut = mediaHandler.UploadPut
	server.UpdateProfile = usersHandler.UpdateProfile
	server.UserDetail = usersHandler.UserDetail
	server.UserList = usersHandler.UserList
	server.UserMedia = usersHandler.UserMedia
	server.UserMediaKind = usersHandler.UserMediaKind
	server.Vacancies = jobsHandler.Vacancies
	server.VideoAccess = videosHandler.VideoAccess
	server.View = viewsHandler.View
	server.VoiceJSON = chatsHandler.VoiceJSON
	server.VoIP = chatsHandler.VoIP
	server.Watchlist = moviesHandler.Watchlist
	server.WatchlistFor = moviesHandler.WatchlistFor
	server.Websocket = chatsHandler.Websocket
	admin.Register(server)
	albums.Register(server)
	artists.Register(server)
	authcookie.Register(server)
	chats.Register(server)
	comments.Register(server)
	communities.Register(server)
	jobs.Register(server)
	links.Register(server)
	media.Register(server)
	memes.Register(server)
	movies.Register(server)
	notes.Register(server)
	notifications.Register(server)
	photos.Register(server)
	playlists.Register(server)
	push.Register(server)
	posts.Register(server)
	queries.Register(server)
	todos.Register(server)
	tracks.Register(server)
	users.Register(server)
	videos.Register(server)
	views.Register(server)
	websites.Register(server)
	Register(server)
	return server
}
