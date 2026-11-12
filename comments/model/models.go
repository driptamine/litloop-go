// Package comments contains the comments domain models and SQL relationships.
package model

import (
	users "litloop/backend/users/model"
	"time"
)

type Comment struct {
	ID            int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	AddDate       time.Time   `json:"add_date" gorm:"column:add_date;autoCreateTime"`
	FriendlyToken string      `json:"friendly_token" gorm:"column:friendly_token;index;size:27"`
	Text          string      `json:"text" gorm:"column:text"`
	UID           string      `json:"uid" gorm:"column:uid;uniqueIndex"`
	UserID        int64       `json:"user_id" gorm:"column:user_id;index"`
	ParentID      *int64      `json:"parent_id" gorm:"column:parent_id;index"`
	Lft           int64       `json:"lft" gorm:"column:lft;default:0"`
	Rght          int64       `json:"rght" gorm:"column:rght;default:0"`
	TreeID        int64       `json:"tree_id" gorm:"column:tree_id;default:0"`
	Level         int64       `json:"level" gorm:"column:level;default:0"`
	UserIDRef     *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	ParentIDRef   *Comment    `json:"-" gorm:"belongsTo:true;foreignKey:ParentID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Comment) TableName() string { return "comments_comment" }

type CommentPhoto struct {
	ID int64 `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
}

func (CommentPhoto) TableName() string { return "comments_commentphoto" }

type CommentVideo struct {
	ID int64 `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
}

func (CommentVideo) TableName() string { return "comments_commentvideo" }

type CommentAudio struct {
	ID int64 `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
}

func (CommentAudio) TableName() string { return "comments_commentaudio" }

type CommentTrack struct {
	ID int64 `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
}

func (CommentTrack) TableName() string { return "comments_commenttrack" }
