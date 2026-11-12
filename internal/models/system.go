package models

import (
	"litloop/backend/internal/types"
	"time"
)

type JSON = types.JSON

var AsJSON = types.AsJSON

type Task struct {
	ID        int64      `json:"id" gorm:"primaryKey"`
	Kind      string     `json:"kind" gorm:"index"`
	Payload   JSON       `json:"payload" gorm:"type:json"`
	Status    string     `json:"status" gorm:"index;default:pending"`
	Attempts  int        `json:"attempts"`
	Error     string     `json:"error"`
	RunAt     time.Time  `json:"run_at" gorm:"index"`
	LockedAt  *time.Time `json:"locked_at"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (Task) TableName() string { return "go_tasks" }

type Upload struct {
	ID          string    `json:"upload_id" gorm:"primaryKey"`
	UserID      int64     `json:"user_id" gorm:"index"`
	Provider    string    `json:"provider"`
	Kind        string    `json:"kind"`
	Key         string    `json:"key" gorm:"uniqueIndex"`
	ContentType string    `json:"content_type"`
	Filename    string    `json:"filename"`
	RemoteID    string    `json:"-"`
	Status      string    `json:"status"`
	ResourceID  int64     `json:"resource_id"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Upload) TableName() string { return "go_uploads" }

type ResetToken struct {
	ID        string `gorm:"primaryKey"`
	UserID    int64  `gorm:"index"`
	ExpiresAt time.Time
	Used      bool
}

func (ResetToken) TableName() string { return "go_reset_tokens" }

type RevokedToken struct {
	ID        string `gorm:"primaryKey"`
	ExpiresAt time.Time
}

func (RevokedToken) TableName() string { return "go_revoked_tokens" }

type MovieLike struct {
	ID      int64 `gorm:"primaryKey"`
	MovieID int64 `gorm:"uniqueIndex:uq_movie_like"`
	UserID  int64 `gorm:"uniqueIndex:uq_movie_like"`
}

func (MovieLike) TableName() string { return "go_movie_likes" }

// Asset provides a stable capability URL for profile and community images.
type Asset struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	UserID      int64     `json:"user_id" gorm:"index"`
	Key         string    `json:"key"`
	Provider    string    `json:"provider"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Asset) TableName() string { return "go_assets" }
