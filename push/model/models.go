// Package push contains the web push subscription models and SQL relationships.
package model

import (
	users "litloop/backend/users/model"
	"time"
)

type Subscription struct {
	ID        int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64       `json:"user_id" gorm:"column:user_id;index"`
	Endpoint  string      `json:"endpoint" gorm:"column:endpoint;size:500;uniqueIndex"`
	P256DH    string      `json:"p256dh" gorm:"column:p256dh;size:200"`
	Auth      string      `json:"auth" gorm:"column:auth;size:100"`
	CreatedAt time.Time   `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UserIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Subscription) TableName() string { return "push_subscription" }
