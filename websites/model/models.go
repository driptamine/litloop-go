// Package websites contains the websites domain models and SQL relationships.
package model

import (
	"time"
)

type Website struct {
	ID        int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	URL       string    `json:"url" gorm:"column:url;uniqueIndex"`
	Title     string    `json:"title" gorm:"column:title"`
	Body      string    `json:"body" gorm:"column:body"`
	Favicon   string    `json:"favicon" gorm:"column:favicon;size:400"`
	Query     string    `json:"query" gorm:"column:query;size:500"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (Website) TableName() string { return "websites_website" }

type WebsiteImage struct {
	ID          int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	URL         string    `json:"url" gorm:"column:url"`
	Title       string    `json:"title" gorm:"column:title"`
	Description string    `json:"description" gorm:"column:description"`
	PageURL     string    `json:"page_url" gorm:"column:page_url"`
	Thumbnail   string    `json:"thumbnail" gorm:"column:thumbnail"`
	Height      string    `json:"height" gorm:"column:height;size:400"`
	Width       string    `json:"width" gorm:"column:width;size:400"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (WebsiteImage) TableName() string { return "websites_websiteimage" }
