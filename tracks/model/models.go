// Package tracks contains the tracks domain models and SQL relationships.
package model

import (
	albums "litloop/backend/albums/model"
	artists "litloop/backend/artists/model"
	users "litloop/backend/users/model"
	"time"
)

type SpotifyTrack struct {
	ID       int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	S3Key    string `json:"s3_key" gorm:"column:s3_key;size:400"`
	GCSKey   string `json:"gcs_key" gorm:"column:gcs_key;size:400"`
	R2Key    string `json:"r2_key" gorm:"column:r2_key;size:400"`
	Filename string `json:"filename" gorm:"column:filename;size:400"`
}

func (SpotifyTrack) TableName() string { return "tracks_spotifytrack" }

type Track struct {
	ID          int64         `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Status      string        `json:"status" gorm:"column:status;size:20;default:draft"`
	S3Key       string        `json:"s3_key" gorm:"column:s3_key;size:400"`
	GCSKey      string        `json:"gcs_key" gorm:"column:gcs_key;size:400"`
	R2Key       string        `json:"r2_key" gorm:"column:r2_key;size:400"`
	Filename    string        `json:"filename" gorm:"column:filename;size:400"`
	TrackURI    string        `json:"track_uri" gorm:"column:track_uri;size:400"`
	Name        string        `json:"name" gorm:"column:name;size:400"`
	TrackNumber string        `json:"track_number" gorm:"column:track_number;size:400"`
	Duration    float64       `json:"duration" gorm:"column:duration;default:0"`
	AlbumID     *int64        `json:"album_id" gorm:"column:album_id;index"`
	CreatedAt   time.Time     `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time     `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	ReleaseDate *time.Time    `json:"release_date" gorm:"column:release_date"`
	UserID      *int64        `json:"user_id" gorm:"column:user_id;index"`
	AlbumIDRef  *albums.Album `json:"-" gorm:"belongsTo:true;foreignKey:AlbumID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef   *users.User   `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Track) TableName() string { return "tracks_track" }

type TrackLike struct {
	ID           int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	TrackID      int64       `json:"track_id" gorm:"column:track_id;index"`
	LikedByID    int64       `json:"liked_by_id" gorm:"column:liked_by_id;index"`
	TrackIDRef   *Track      `json:"-" gorm:"belongsTo:true;foreignKey:TrackID;references:ID;constraint:OnDelete:CASCADE"`
	LikedByIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:LikedByID;references:ID;constraint:OnDelete:CASCADE"`
}

func (TrackLike) TableName() string { return "tracks_tracklike" }

type TrackDislike struct {
	ID             int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	TrackID        int64       `json:"track_id" gorm:"column:track_id;index"`
	DislikeByID    int64       `json:"dislike_by_id" gorm:"column:dislike_by_id;index"`
	TrackIDRef     *Track      `json:"-" gorm:"belongsTo:true;foreignKey:TrackID;references:ID;constraint:OnDelete:CASCADE"`
	DislikeByIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:DislikeByID;references:ID;constraint:OnDelete:CASCADE"`
}

func (TrackDislike) TableName() string { return "tracks_trackdislike" }

type TrackImpression struct {
	ID         int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	TrackID    int64       `json:"track_id" gorm:"column:track_id;index"`
	UserID     int64       `json:"user_id" gorm:"column:user_id;index"`
	TrackIDRef *Track      `json:"-" gorm:"belongsTo:true;foreignKey:TrackID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef  *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (TrackImpression) TableName() string { return "tracks_trackimpression" }

type TrackView struct {
	ID         int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	TrackID    int64       `json:"track_id" gorm:"column:track_id;index"`
	UserID     int64       `json:"user_id" gorm:"column:user_id;index"`
	TrackIDRef *Track      `json:"-" gorm:"belongsTo:true;foreignKey:TrackID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef  *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (TrackView) TableName() string { return "tracks_trackview" }

type TrackPlay struct {
	ID         int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	TrackID    int64       `json:"track_id" gorm:"column:track_id;index"`
	UserID     int64       `json:"user_id" gorm:"column:user_id;index"`
	TrackIDRef *Track      `json:"-" gorm:"belongsTo:true;foreignKey:TrackID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef  *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (TrackPlay) TableName() string { return "tracks_trackplay" }

type TrackArtist struct {
	ID          int64           `json:"id" gorm:"primaryKey"`
	TrackID     int64           `json:"track_id" gorm:"column:track_id;uniqueIndex:uq_tracks_track_artists;index"`
	ArtistID    int64           `json:"artist_id" gorm:"column:artist_id;uniqueIndex:uq_tracks_track_artists;index"`
	TrackIDRef  *Track          `json:"-" gorm:"belongsTo:true;foreignKey:TrackID;references:ID;constraint:OnDelete:CASCADE"`
	ArtistIDRef *artists.Artist `json:"-" gorm:"belongsTo:true;foreignKey:ArtistID;references:ID;constraint:OnDelete:CASCADE"`
}

func (TrackArtist) TableName() string { return "tracks_track_artists" }
