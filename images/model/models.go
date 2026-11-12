// Package images contains the images domain models and SQL relationships.
package model

import (
	albums "litloop/backend/albums/model"
	artists "litloop/backend/artists/model"
)

type Image struct {
	ID          int64           `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Height      string          `json:"height" gorm:"column:height;size:400"`
	Width       string          `json:"width" gorm:"column:width;size:400"`
	URL         string          `json:"url" gorm:"column:url;size:400"`
	ArtistID    *int64          `json:"artist_id" gorm:"column:artist_id;index"`
	AlbumID     *int64          `json:"album_id" gorm:"column:album_id;index"`
	ArtistIDRef *artists.Artist `json:"-" gorm:"belongsTo:true;foreignKey:ArtistID;references:ID;constraint:OnDelete:CASCADE"`
	AlbumIDRef  *albums.Album   `json:"-" gorm:"belongsTo:true;foreignKey:AlbumID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Image) TableName() string { return "images_image" }
