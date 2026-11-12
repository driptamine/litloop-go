// Package broadcasts contains the broadcasts domain models and SQL relationships.
package model

import (
	users "litloop/backend/users/model"
	"time"
)

type Channel struct {
	ID         int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name       string      `json:"name" gorm:"column:name;size:255"`
	OwnerID    int64       `json:"owner_id" gorm:"column:owner_id;index"`
	OwnerIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:OwnerID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Channel) TableName() string { return "broadcasts_channel" }

type ChannelPost struct {
	ID                int64        `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ChannelID         int64        `json:"channel_id" gorm:"column:channel_id;index"`
	AuthorID          int64        `json:"author_id" gorm:"column:author_id;index"`
	Content           string       `json:"content" gorm:"column:content"`
	CreatedAt         time.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	OriginalPostID    *int64       `json:"original_post_id" gorm:"column:original_post_id;index"`
	ChannelIDRef      *Channel     `json:"-" gorm:"belongsTo:true;foreignKey:ChannelID;references:ID;constraint:OnDelete:CASCADE"`
	AuthorIDRef       *users.User  `json:"-" gorm:"belongsTo:true;foreignKey:AuthorID;references:ID;constraint:OnDelete:CASCADE"`
	OriginalPostIDRef *ChannelPost `json:"-" gorm:"belongsTo:true;foreignKey:OriginalPostID;references:ID;constraint:OnDelete:SET NULL"`
}

func (ChannelPost) TableName() string { return "broadcasts_channelpost" }

type ChannelPostLike struct {
	ID               int64        `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ChannelpostID    int64        `json:"channelpost_id" gorm:"column:channelpost_id;index"`
	UserID           int64        `json:"user_id" gorm:"column:user_id;index"`
	CreatedAt        time.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	ChannelpostIDRef *ChannelPost `json:"-" gorm:"belongsTo:true;foreignKey:ChannelpostID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef        *users.User  `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ChannelPostLike) TableName() string { return "broadcasts_channelpostlike" }
