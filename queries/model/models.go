// Package queries contains the queries domain models and SQL relationships.
package model

import (
	"time"
)

type Query struct {
	ID         int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	QueryText  string `json:"query_text" gorm:"column:query_text;uniqueIndex:uq_queries_query;size:255"`
	Suggestion string `json:"suggestion" gorm:"column:suggestion;uniqueIndex:uq_queries_query;size:255"`
	IsCrawled  bool   `json:"is_crawled" gorm:"column:is_crawled;default:false"`
}

func (Query) TableName() string { return "queries_query" }

type UserQuery struct {
	ID         int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	QueryText  string `json:"query_text" gorm:"column:query_text;size:255"`
	Suggestion string `json:"suggestion" gorm:"column:suggestion;size:255"`
}

func (UserQuery) TableName() string { return "queries_userquery" }

type BraveQuery struct {
	ID         int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	QueryText  string `json:"query_text" gorm:"column:query_text;size:255"`
	Suggestion string `json:"suggestion" gorm:"column:suggestion;size:255"`
}

func (BraveQuery) TableName() string { return "queries_bravequery" }

type GoogleQuery struct {
	ID         int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	QueryText  string    `json:"query_text" gorm:"column:query_text;uniqueIndex:uq_queries_googlequery;size:255"`
	Suggestion string    `json:"suggestion" gorm:"column:suggestion;uniqueIndex:uq_queries_googlequery;size:255"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (GoogleQuery) TableName() string { return "queries_googlequery" }

type BingQuery struct {
	ID         int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	QueryText  string `json:"query_text" gorm:"column:query_text;size:255"`
	Suggestion string `json:"suggestion" gorm:"column:suggestion;size:255"`
}

func (BingQuery) TableName() string { return "queries_bingquery" }
