// Package albums contains the albums domain models and SQL relationships.
package model

import (
	artists "litloop/backend/artists/model"
	users "litloop/backend/users/model"
	"time"
)

type AlbumTrack struct {
	ID          int64           `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	AlbumID     int64           `json:"album_id" gorm:"column:album_id;index"`
	ArtistID    int64           `json:"artist_id" gorm:"column:artist_id;index"`
	AlbumIDRef  *Album          `json:"-" gorm:"belongsTo:true;foreignKey:AlbumID;references:ID;constraint:OnDelete:CASCADE"`
	ArtistIDRef *artists.Artist `json:"-" gorm:"belongsTo:true;foreignKey:ArtistID;references:ID;constraint:OnDelete:CASCADE"`
}

func (AlbumTrack) TableName() string { return "albums_albumtrack" }

type AlbumArtist struct {
	ID          int64           `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	AlbumID     int64           `json:"album_id" gorm:"column:album_id;index"`
	ArtistID    int64           `json:"artist_id" gorm:"column:artist_id;index"`
	Role        string          `json:"role" gorm:"column:role;size:200"`
	AlbumIDRef  *Album          `json:"-" gorm:"belongsTo:true;foreignKey:AlbumID;references:ID;constraint:OnDelete:CASCADE"`
	ArtistIDRef *artists.Artist `json:"-" gorm:"belongsTo:true;foreignKey:ArtistID;references:ID;constraint:OnDelete:CASCADE"`
}

func (AlbumArtist) TableName() string { return "albums_albumartist" }

type Album struct {
	ID        int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name      string    `json:"name" gorm:"column:name;size:400"`
	AlbumURI  string    `json:"album_uri" gorm:"column:album_uri;size:400"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (Album) TableName() string { return "albums_album" }

type AlbumLike struct {
	ID           int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	AlbumID      int64       `json:"album_id" gorm:"column:album_id;index"`
	LikedByID    int64       `json:"liked_by_id" gorm:"column:liked_by_id;index"`
	AlbumIDRef   *Album      `json:"-" gorm:"belongsTo:true;foreignKey:AlbumID;references:ID;constraint:OnDelete:CASCADE"`
	LikedByIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:LikedByID;references:ID;constraint:OnDelete:CASCADE"`
}

func (AlbumLike) TableName() string { return "albums_albumlike" }

type AlbumDislike struct {
	ID             int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	AlbumID        int64       `json:"album_id" gorm:"column:album_id;index"`
	DislikeByID    int64       `json:"dislike_by_id" gorm:"column:dislike_by_id;index"`
	AlbumIDRef     *Album      `json:"-" gorm:"belongsTo:true;foreignKey:AlbumID;references:ID;constraint:OnDelete:CASCADE"`
	DislikeByIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:DislikeByID;references:ID;constraint:OnDelete:CASCADE"`
}

func (AlbumDislike) TableName() string { return "albums_albumdislike" }

type AlbumImpression struct {
	ID         int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	AlbumID    int64       `json:"album_id" gorm:"column:album_id;index"`
	UserID     int64       `json:"user_id" gorm:"column:user_id;index"`
	AlbumIDRef *Album      `json:"-" gorm:"belongsTo:true;foreignKey:AlbumID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef  *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (AlbumImpression) TableName() string { return "albums_albumimpression" }

type AlbumView struct {
	ID         int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	AlbumID    int64       `json:"album_id" gorm:"column:album_id;index"`
	UserID     int64       `json:"user_id" gorm:"column:user_id;index"`
	AlbumIDRef *Album      `json:"-" gorm:"belongsTo:true;foreignKey:AlbumID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef  *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (AlbumView) TableName() string { return "albums_albumview" }
