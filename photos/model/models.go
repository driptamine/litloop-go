// Package photos contains the photos domain models and SQL relationships.
package model

import (
	users "litloop/backend/users/model"
	"time"
)

type Photo struct {
	ID            int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Status        string      `json:"status" gorm:"column:status;size:20;default:draft"`
	S3Key         string      `json:"s3_key" gorm:"column:s3_key;size:400"`
	GCSKey        string      `json:"gcs_key" gorm:"column:gcs_key;size:400"`
	R2Key         string      `json:"r2_key" gorm:"column:r2_key;size:400"`
	Filename      string      `json:"filename" gorm:"column:filename;size:400"`
	UserID        *int64      `json:"user_id" gorm:"column:user_id;index"`
	Title         string      `json:"title" gorm:"column:title"`
	UserFeatured  bool        `json:"user_featured" gorm:"column:user_featured;default:false"`
	FriendlyToken string      `json:"friendly_token" gorm:"column:friendly_token;index;size:12"`
	Likes         int64       `json:"likes" gorm:"column:likes;default:0"`
	Dislikes      int64       `json:"dislikes" gorm:"column:dislikes;default:0"`
	Views         int64       `json:"views" gorm:"column:views;default:0"`
	Impressions   int64       `json:"impressions" gorm:"column:impressions;default:0"`
	UserIDRef     *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Photo) TableName() string { return "photos_photo" }

type PhotoLike struct {
	ID           int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PhotoID      int64       `json:"photo_id" gorm:"column:photo_id;index"`
	LikedByID    int64       `json:"liked_by_id" gorm:"column:liked_by_id;index"`
	PhotoIDRef   *Photo      `json:"-" gorm:"belongsTo:true;foreignKey:PhotoID;references:ID;constraint:OnDelete:CASCADE"`
	LikedByIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:LikedByID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PhotoLike) TableName() string { return "photos_photolike" }

type PhotoDislike struct {
	ID             int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PhotoID        int64       `json:"photo_id" gorm:"column:photo_id;index"`
	DislikeByID    int64       `json:"dislike_by_id" gorm:"column:dislike_by_id;index"`
	PhotoIDRef     *Photo      `json:"-" gorm:"belongsTo:true;foreignKey:PhotoID;references:ID;constraint:OnDelete:CASCADE"`
	DislikeByIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:DislikeByID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PhotoDislike) TableName() string { return "photos_photodislike" }

type PhotoImpression struct {
	ID         int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PhotoID    int64       `json:"photo_id" gorm:"column:photo_id;index"`
	UserID     int64       `json:"user_id" gorm:"column:user_id;index"`
	PhotoIDRef *Photo      `json:"-" gorm:"belongsTo:true;foreignKey:PhotoID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef  *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PhotoImpression) TableName() string { return "photos_photoimpression" }

type PhotoView struct {
	ID         int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PhotoID    int64       `json:"photo_id" gorm:"column:photo_id;index"`
	UserID     int64       `json:"user_id" gorm:"column:user_id;index"`
	PhotoIDRef *Photo      `json:"-" gorm:"belongsTo:true;foreignKey:PhotoID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef  *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PhotoView) TableName() string { return "photos_photoview" }

type PhotoAlbum struct {
	ID            int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	AddDate       time.Time   `json:"add_date" gorm:"column:add_date;index;autoCreateTime"`
	Description   string      `json:"description" gorm:"column:description"`
	FriendlyToken string      `json:"friendly_token" gorm:"column:friendly_token;index;size:12"`
	Title         string      `json:"title" gorm:"column:title;index;size:100"`
	UserID        int64       `json:"user_id" gorm:"column:user_id;index"`
	UserIDRef     *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PhotoAlbum) TableName() string { return "photos_photoalbum" }

type PhotoAlbumItem struct {
	ID              int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ActionDate      time.Time   `json:"action_date" gorm:"column:action_date;autoUpdateTime"`
	PhotoID         int64       `json:"photo_id" gorm:"column:photo_id;index"`
	PhotoAlbumID    int64       `json:"photo_album_id" gorm:"column:photo_album_id;index"`
	Ordering        int64       `json:"ordering" gorm:"column:ordering;default:1"`
	PhotoIDRef      *Photo      `json:"-" gorm:"belongsTo:true;foreignKey:PhotoID;references:ID;constraint:OnDelete:CASCADE"`
	PhotoAlbumIDRef *PhotoAlbum `json:"-" gorm:"belongsTo:true;foreignKey:PhotoAlbumID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PhotoAlbumItem) TableName() string { return "photos_photoalbumitem" }
