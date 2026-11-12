// Package links contains the links domain models and SQL relationships.
package model

import (
	users "litloop/backend/users/model"
	"time"
)

type Hashtag struct {
	ID   int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name string `json:"name" gorm:"column:name;uniqueIndex;size:100"`
}

func (Hashtag) TableName() string { return "links_hashtag" }

type Link struct {
	ID        int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	UserID    *int64      `json:"user_id" gorm:"column:user_id;index"`
	Title     string      `json:"title" gorm:"column:title;size:255"`
	URL       string      `json:"url" gorm:"column:url"`
	Status    string      `json:"status" gorm:"column:status;size:20;default:unread"`
	CreatedAt time.Time   `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time   `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	TimeAdded *time.Time  `json:"time_added" gorm:"column:time_added"`
	UserIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:SET NULL"`
}

func (Link) TableName() string { return "links_link" }

type LinkTag struct {
	ID           int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	LinkID       int64     `json:"link_id" gorm:"column:link_id;index;uniqueIndex:uq_links_linktag"`
	HashtagID    int64     `json:"hashtag_id" gorm:"column:hashtag_id;index;uniqueIndex:uq_links_linktag"`
	AddedAt      time.Time `json:"added_at" gorm:"column:added_at;autoCreateTime"`
	LinkIDRef    *Link     `json:"-" gorm:"belongsTo:true;foreignKey:LinkID;references:ID;constraint:OnDelete:CASCADE"`
	HashtagIDRef *Hashtag  `json:"-" gorm:"belongsTo:true;foreignKey:HashtagID;references:ID;constraint:OnDelete:CASCADE"`
}

func (LinkTag) TableName() string { return "links_linktag" }
