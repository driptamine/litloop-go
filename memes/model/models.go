// Package memes contains the memes domain models and SQL relationships.
package model

import (
	users "litloop/backend/users/model"
	"time"
)

type Meme struct {
	ID        int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	MemeID    string    `json:"meme_id" gorm:"column:meme_id;uniqueIndex;size:500"`
	Title     string    `json:"title" gorm:"column:title"`
	URL       string    `json:"url" gorm:"column:url"`
	PostLink  string    `json:"post_link" gorm:"column:post_link"`
	Subreddit string    `json:"subreddit" gorm:"column:subreddit;size:255"`
	Author    string    `json:"author" gorm:"column:author;size:255"`
	Nsfw      bool      `json:"nsfw" gorm:"column:nsfw;default:false"`
	Spoiler   bool      `json:"spoiler" gorm:"column:spoiler;default:false"`
	Ups       int64     `json:"ups" gorm:"column:ups;default:0"`
	Preview   string    `json:"preview" gorm:"column:preview"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (Meme) TableName() string { return "memes_meme" }

type MemeLikedBy struct {
	ID        int64       `json:"id" gorm:"primaryKey"`
	MemeID    int64       `json:"meme_id" gorm:"column:meme_id;uniqueIndex:uq_memes_meme_liked_by;index"`
	UserID    int64       `json:"user_id" gorm:"column:user_id;uniqueIndex:uq_memes_meme_liked_by;index"`
	MemeIDRef *Meme       `json:"-" gorm:"belongsTo:true;foreignKey:MemeID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (MemeLikedBy) TableName() string { return "memes_meme_liked_by" }
