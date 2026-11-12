// Package videos contains the videos domain models and SQL relationships.
package model

import (
	comments "litloop/backend/comments/model"
	users "litloop/backend/users/model"
	"time"
)

type Video struct {
	ID          int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Status      string      `json:"status" gorm:"column:status;size:20;default:draft"`
	Visibility  string      `json:"visibility" gorm:"column:visibility;size:20;default:private"`
	S3Key       string      `json:"s3_key" gorm:"column:s3_key;size:400"`
	GCSKey      string      `json:"gcs_key" gorm:"column:gcs_key;size:400"`
	R2Key       string      `json:"r2_key" gorm:"column:r2_key;size:400"`
	Filename    string      `json:"filename" gorm:"column:filename;size:400"`
	HlsS3       string      `json:"hls_s3" gorm:"column:hls_s3;size:400"`
	Title       string      `json:"title" gorm:"column:title;size:100"`
	Description string      `json:"description" gorm:"column:description;size:100"`
	VideoFile   string      `json:"video_file" gorm:"column:video_file;size:400"`
	Sprites     string      `json:"sprites" gorm:"column:sprites;size:400"`
	Thumbnail   string      `json:"thumbnail" gorm:"column:thumbnail;size:400"`
	Likes       int64       `json:"likes" gorm:"column:likes;default:0"`
	Dislikes    int64       `json:"dislikes" gorm:"column:dislikes;default:0"`
	Views       int64       `json:"views" gorm:"column:views;default:0"`
	Impressions int64       `json:"impressions" gorm:"column:impressions;default:0"`
	UserID      *int64      `json:"user_id" gorm:"column:user_id;index"`
	UserIDRef   *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Video) TableName() string { return "videos_video" }

type VideoWatchProgress struct {
	ID           int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	UserID       int64       `json:"user_id" gorm:"column:user_id;index;uniqueIndex:uq_videos_videowatchprogress"`
	VideoID      int64       `json:"video_id" gorm:"column:video_id;index;uniqueIndex:uq_videos_videowatchprogress"`
	PlaybackTime float64     `json:"playback_time" gorm:"column:playback_time;default:0.0"`
	UpdatedAt    time.Time   `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	UserIDRef    *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	VideoIDRef   *Video      `json:"-" gorm:"belongsTo:true;foreignKey:VideoID;references:ID;constraint:OnDelete:CASCADE"`
}

func (VideoWatchProgress) TableName() string { return "videos_videowatchprogress" }

type VideoComment struct {
	ID           int64             `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	CommentID    int64             `json:"comment_id" gorm:"column:comment_id;index"`
	VideoID      int64             `json:"video_id" gorm:"column:video_id;index"`
	CommentIDRef *comments.Comment `json:"-" gorm:"belongsTo:true;foreignKey:CommentID;references:ID;constraint:OnDelete:CASCADE"`
	VideoIDRef   *Video            `json:"-" gorm:"belongsTo:true;foreignKey:VideoID;references:ID;constraint:OnDelete:CASCADE"`
}

func (VideoComment) TableName() string { return "videos_videocomment" }

type VideoPlaylist struct {
	ID    int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Title string `json:"title" gorm:"column:title;size:900"`
}

func (VideoPlaylist) TableName() string { return "videos_videoplaylist" }

type VideoPlaylistItem struct {
	ID            int64          `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	VideoID       int64          `json:"video_id" gorm:"column:video_id;index"`
	PlaylistID    int64          `json:"playlist_id" gorm:"column:playlist_id;index"`
	Ordering      int64          `json:"ordering" gorm:"column:ordering;default:1"`
	VideoIDRef    *Video         `json:"-" gorm:"belongsTo:true;foreignKey:VideoID;references:ID;constraint:OnDelete:CASCADE"`
	PlaylistIDRef *VideoPlaylist `json:"-" gorm:"belongsTo:true;foreignKey:PlaylistID;references:ID;constraint:OnDelete:CASCADE"`
}

func (VideoPlaylistItem) TableName() string { return "videos_videoplaylistitem" }
