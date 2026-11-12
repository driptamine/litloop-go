// Package suggestions contains the suggestions domain models and SQL relationships.
package model

type Suggestion struct {
	ID         int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name       string `json:"name" gorm:"column:name;size:100"`
	SpotifyURL string `json:"spotify_url" gorm:"column:spotify_url;size:100"`
}

func (Suggestion) TableName() string { return "suggestions_suggestion" }
