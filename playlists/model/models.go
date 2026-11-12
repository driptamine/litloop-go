// Package playlists contains the playlists domain models and SQL relationships.
package model

import (
	users "litloop/backend/users/model"
)

type Playlist struct {
	ID            int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Title         string      `json:"title" gorm:"column:title;size:400"`
	Description   string      `json:"description" gorm:"column:description"`
	FriendlyToken string      `json:"friendly_token" gorm:"column:friendly_token;size:100"`
	UserID        *int64      `json:"user_id" gorm:"column:user_id;index"`
	UserIDRef     *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Playlist) TableName() string { return "playlists_playlist" }
