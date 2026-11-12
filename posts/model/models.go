// Package posts contains the posts domain models and SQL relationships.
package model

import (
	comments "litloop/backend/comments/model"
	photos "litloop/backend/photos/model"
	playlists "litloop/backend/playlists/model"
	tracks "litloop/backend/tracks/model"
	users "litloop/backend/users/model"
	videos "litloop/backend/videos/model"
	"time"
)

type Post struct {
	ID               int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Title            string      `json:"title" gorm:"column:title;size:100"`
	Description      string      `json:"description" gorm:"column:description;size:100"`
	FriendlyID       string      `json:"friendly_id" gorm:"column:friendly_id;size:100"`
	AuthorID         *int64      `json:"author_id" gorm:"column:author_id;index"`
	LikesCount       int64       `json:"likes_count" gorm:"column:likes_count;default:0"`
	DislikesCount    int64       `json:"dislikes_count" gorm:"column:dislikes_count;default:0"`
	ViewsCount       int64       `json:"views_count" gorm:"column:views_count;default:0"`
	ImpressionsCount int64       `json:"impressions_count" gorm:"column:impressions_count;default:0"`
	CreatedAt        time.Time   `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time   `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	AuthorIDRef      *users.User `json:"-" gorm:"belongsTo:true;foreignKey:AuthorID;references:ID;constraint:OnDelete:SET NULL"`
}

func (Post) TableName() string { return "posts_post" }

type PostLike struct {
	ID        int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PostID    int64       `json:"post_id" gorm:"column:post_id;index"`
	UserID    int64       `json:"user_id" gorm:"column:user_id;index"`
	LikedAt   time.Time   `json:"liked_at" gorm:"column:liked_at;autoCreateTime"`
	PostIDRef *Post       `json:"-" gorm:"belongsTo:true;foreignKey:PostID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PostLike) TableName() string { return "posts_postlike" }

type PostDislike struct {
	ID         int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PostID     int64       `json:"post_id" gorm:"column:post_id;index"`
	UserID     int64       `json:"user_id" gorm:"column:user_id;index"`
	DislikedAt time.Time   `json:"disliked_at" gorm:"column:disliked_at;autoCreateTime"`
	PostIDRef  *Post       `json:"-" gorm:"belongsTo:true;foreignKey:PostID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef  *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PostDislike) TableName() string { return "posts_postdislike" }

type PostImpression struct {
	ID        int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PostID    int64       `json:"post_id" gorm:"column:post_id;index"`
	UserID    *int64      `json:"user_id" gorm:"column:user_id;index"`
	PostIDRef *Post       `json:"-" gorm:"belongsTo:true;foreignKey:PostID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PostImpression) TableName() string { return "posts_postimpression" }

type PostView struct {
	ID        int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PostID    int64       `json:"post_id" gorm:"column:post_id;index"`
	UserID    int64       `json:"user_id" gorm:"column:user_id;index"`
	ViewedAt  time.Time   `json:"viewed_at" gorm:"column:viewed_at;autoCreateTime"`
	PostIDRef *Post       `json:"-" gorm:"belongsTo:true;foreignKey:PostID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PostView) TableName() string { return "posts_postview" }

type PostPhoto struct {
	ID         int64         `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PostID     int64         `json:"post_id" gorm:"column:post_id;index"`
	PhotoID    int64         `json:"photo_id" gorm:"column:photo_id;index"`
	Order      int64         `json:"order" gorm:"column:order;default:1"`
	PostIDRef  *Post         `json:"-" gorm:"belongsTo:true;foreignKey:PostID;references:ID;constraint:OnDelete:CASCADE"`
	PhotoIDRef *photos.Photo `json:"-" gorm:"belongsTo:true;foreignKey:PhotoID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PostPhoto) TableName() string { return "posts_postphoto" }

type PostVideo struct {
	ID         int64         `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PostID     int64         `json:"post_id" gorm:"column:post_id;index"`
	VideoID    int64         `json:"video_id" gorm:"column:video_id;index"`
	Order      int64         `json:"order" gorm:"column:order;default:1"`
	PostIDRef  *Post         `json:"-" gorm:"belongsTo:true;foreignKey:PostID;references:ID;constraint:OnDelete:CASCADE"`
	VideoIDRef *videos.Video `json:"-" gorm:"belongsTo:true;foreignKey:VideoID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PostVideo) TableName() string { return "posts_postvideo" }

type PostTrack struct {
	ID         int64         `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PostID     int64         `json:"post_id" gorm:"column:post_id;index"`
	TrackID    int64         `json:"track_id" gorm:"column:track_id;index"`
	Order      int64         `json:"order" gorm:"column:order;default:1"`
	PostIDRef  *Post         `json:"-" gorm:"belongsTo:true;foreignKey:PostID;references:ID;constraint:OnDelete:CASCADE"`
	TrackIDRef *tracks.Track `json:"-" gorm:"belongsTo:true;foreignKey:TrackID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PostTrack) TableName() string { return "posts_posttrack" }

type PostPlaylist struct {
	ID            int64               `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PostID        int64               `json:"post_id" gorm:"column:post_id;index"`
	PlaylistID    int64               `json:"playlist_id" gorm:"column:playlist_id;index"`
	Order         int64               `json:"order" gorm:"column:order;default:1"`
	PostIDRef     *Post               `json:"-" gorm:"belongsTo:true;foreignKey:PostID;references:ID;constraint:OnDelete:CASCADE"`
	PlaylistIDRef *playlists.Playlist `json:"-" gorm:"belongsTo:true;foreignKey:PlaylistID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PostPlaylist) TableName() string { return "posts_postplaylist" }

type PostComment struct {
	ID           int64             `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	CommentID    int64             `json:"comment_id" gorm:"column:comment_id;index"`
	PostID       int64             `json:"post_id" gorm:"column:post_id;index"`
	CommentIDRef *comments.Comment `json:"-" gorm:"belongsTo:true;foreignKey:CommentID;references:ID;constraint:OnDelete:CASCADE"`
	PostIDRef    *Post             `json:"-" gorm:"belongsTo:true;foreignKey:PostID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PostComment) TableName() string { return "posts_postcomment" }
