package models

import (
	albums "litloop/backend/albums/model"
	artists "litloop/backend/artists/model"
	broadcasts "litloop/backend/broadcasts/model"
	chats "litloop/backend/chats/model"
	comments "litloop/backend/comments/model"
	communities "litloop/backend/communities/model"
	images "litloop/backend/images/model"
	jobs "litloop/backend/jobs/model"
	links "litloop/backend/links/model"
	magazines "litloop/backend/magazines/model"
	memes "litloop/backend/memes/model"
	movies "litloop/backend/movies/model"
	notes "litloop/backend/notes/model"
	notifications "litloop/backend/notifications/model"
	photos "litloop/backend/photos/model"
	playlists "litloop/backend/playlists/model"
	posts "litloop/backend/posts/model"
	push "litloop/backend/push/model"
	queries "litloop/backend/queries/model"
	suggestions "litloop/backend/suggestions/model"
	todos "litloop/backend/todos/model"
	tracks "litloop/backend/tracks/model"
	users "litloop/backend/users/model"
	videos "litloop/backend/videos/model"
	websites "litloop/backend/websites/model"
)

type User = users.User
type ReservedUsername = users.ReservedUsername
type Artist = artists.Artist
type AlbumTrack = albums.AlbumTrack
type AlbumArtist = albums.AlbumArtist
type Album = albums.Album
type AlbumLike = albums.AlbumLike
type AlbumDislike = albums.AlbumDislike
type AlbumImpression = albums.AlbumImpression
type AlbumView = albums.AlbumView
type SpotifyTrack = tracks.SpotifyTrack
type Track = tracks.Track
type TrackLike = tracks.TrackLike
type TrackDislike = tracks.TrackDislike
type TrackImpression = tracks.TrackImpression
type TrackView = tracks.TrackView
type TrackPlay = tracks.TrackPlay
type Photo = photos.Photo
type PhotoLike = photos.PhotoLike
type PhotoDislike = photos.PhotoDislike
type PhotoImpression = photos.PhotoImpression
type PhotoView = photos.PhotoView
type PhotoAlbum = photos.PhotoAlbum
type PhotoAlbumItem = photos.PhotoAlbumItem
type Video = videos.Video
type VideoWatchProgress = videos.VideoWatchProgress
type VideoComment = videos.VideoComment
type VideoPlaylist = videos.VideoPlaylist
type VideoPlaylistItem = videos.VideoPlaylistItem
type Playlist = playlists.Playlist
type Comment = comments.Comment
type CommentPhoto = comments.CommentPhoto
type CommentVideo = comments.CommentVideo
type CommentAudio = comments.CommentAudio
type CommentTrack = comments.CommentTrack
type Post = posts.Post
type PostLike = posts.PostLike
type PostDislike = posts.PostDislike
type PostImpression = posts.PostImpression
type PostView = posts.PostView
type PostPhoto = posts.PostPhoto
type PostVideo = posts.PostVideo
type PostTrack = posts.PostTrack
type PostPlaylist = posts.PostPlaylist
type PostComment = posts.PostComment
type Chat = chats.Chat
type Message = chats.Message
type MessageVideo = chats.MessageVideo
type MessagePhoto = chats.MessagePhoto
type MessageTrack = chats.MessageTrack
type MessageVoice = chats.MessageVoice
type VoiceMessage = chats.VoiceMessage
type Call = chats.Call
type CallParticipant = chats.CallParticipant
type Community = communities.Community
type CommunityMembership = communities.CommunityMembership
type CommunityPost = communities.CommunityPost
type Profile = todos.Profile
type TodoList = todos.TodoList
type Todo = todos.Todo
type Page = notes.Page
type PageTag = notes.PageTag
type Block = notes.Block
type BlockTable = notes.BlockTable
type Company = jobs.Company
type Resume = jobs.Resume
type JobPosting = jobs.JobPosting
type Application = jobs.Application
type CoverLetter = jobs.CoverLetter
type Hashtag = links.Hashtag
type Link = links.Link
type LinkTag = links.LinkTag
type Movie = movies.Movie
type ImdbMovie = movies.ImdbMovie
type WatchlistItem = movies.WatchlistItem
type MovieImpression = movies.MovieImpression
type MovieView = movies.MovieView
type ImdbMovieRating = movies.ImdbMovieRating
type Query = queries.Query
type UserQuery = queries.UserQuery
type BraveQuery = queries.BraveQuery
type GoogleQuery = queries.GoogleQuery
type BingQuery = queries.BingQuery
type Website = websites.Website
type WebsiteImage = websites.WebsiteImage
type Meme = memes.Meme
type Notification = notifications.Notification
type PushSubscription = push.Subscription
type ChatTab = chats.ChatTab
type Channel = broadcasts.Channel
type ChannelPost = broadcasts.ChannelPost
type ChannelPostLike = broadcasts.ChannelPostLike
type Magazine = magazines.Magazine
type Cover = magazines.Cover
type Image = images.Image
type Suggestion = suggestions.Suggestion
type ChatParticipant = chats.ChatParticipant
type TrackArtist = tracks.TrackArtist
type MemeLikedBy = memes.MemeLikedBy

func All() []any {
	return []any{
		&User{},
		&Artist{},
		&AlbumTrack{},
		&AlbumArtist{},
		&Album{},
		&AlbumLike{},
		&AlbumDislike{},
		&AlbumImpression{},
		&AlbumView{},
		&SpotifyTrack{},
		&Track{},
		&TrackLike{},
		&TrackDislike{},
		&TrackImpression{},
		&TrackView{},
		&TrackPlay{},
		&Photo{},
		&PhotoLike{},
		&PhotoDislike{},
		&PhotoImpression{},
		&PhotoView{},
		&PhotoAlbum{},
		&PhotoAlbumItem{},
		&Video{},
		&VideoWatchProgress{},
		&VideoComment{},
		&VideoPlaylist{},
		&VideoPlaylistItem{},
		&Playlist{},
		&Comment{},
		&CommentPhoto{},
		&CommentVideo{},
		&CommentAudio{},
		&CommentTrack{},
		&Post{},
		&PostLike{},
		&PostDislike{},
		&PostImpression{},
		&PostView{},
		&PostPhoto{},
		&PostVideo{},
		&PostTrack{},
		&PostPlaylist{},
		&PostComment{},
		&Chat{},
		&Message{},
		&MessageVideo{},
		&MessagePhoto{},
		&MessageTrack{},
		&MessageVoice{},
		&VoiceMessage{},
		&Call{},
		&CallParticipant{},
		&Community{},
		&CommunityMembership{},
		&CommunityPost{},
		&Profile{},
		&TodoList{},
		&Todo{},
		&Page{},
		&PageTag{},
		&Block{},
		&BlockTable{},
		&Company{},
		&Resume{},
		&JobPosting{},
		&Application{},
		&CoverLetter{},
		&Hashtag{},
		&Link{},
		&LinkTag{},
		&Movie{},
		&ImdbMovie{},
		&WatchlistItem{},
		&MovieImpression{},
		&MovieView{},
		&ImdbMovieRating{},
		&Query{},
		&UserQuery{},
		&BraveQuery{},
		&GoogleQuery{},
		&BingQuery{},
		&Website{},
		&WebsiteImage{},
		&Meme{},
		&Notification{},
		&Channel{},
		&ChannelPost{},
		&ChannelPostLike{},
		&Magazine{},
		&Cover{},
		&Image{},
		&Suggestion{},
		&ChatParticipant{},
		&TrackArtist{},
		&MemeLikedBy{},
		&Task{}, &Upload{}, &ResetToken{}, &RevokedToken{}, &MovieLike{}, &Asset{},
		&PushSubscription{}, &ChatTab{}, &ReservedUsername{},
	}
}
