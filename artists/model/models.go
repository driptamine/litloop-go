// Package artists contains the artists domain models and SQL relationships.
package model

type Artist struct {
	ID        int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name      string `json:"name" gorm:"column:name;size:300"`
	ArtistURI string `json:"artist_uri" gorm:"column:artist_uri;size:300"`
}

func (Artist) TableName() string { return "artists_artist" }
