// Package notifications contains the notifications domain models and SQL relationships.
package model

import (
	users "litloop/backend/users/model"
	"time"
)

type Notification struct {
	ID                  int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	FromUserID          int64       `json:"from_user_id" gorm:"column:from_user_id;index"`
	ToUserID            int64       `json:"to_user_id" gorm:"column:to_user_id;index"`
	Date                *time.Time  `json:"date" gorm:"column:date;autoCreateTime"`
	IsRead              bool        `json:"is_read" gorm:"column:is_read;default:false"`
	TargetContentTypeID *int64      `json:"target_content_type_id" gorm:"column:target_content_type_id;index"`
	TargetObjectID      *int64      `json:"target_object_id" gorm:"column:target_object_id"`
	Target              string      `json:"target" gorm:"column:target"`
	FromUserIDRef       *users.User `json:"-" gorm:"belongsTo:true;foreignKey:FromUserID;references:ID;constraint:OnDelete:CASCADE"`
	ToUserIDRef         *users.User `json:"-" gorm:"belongsTo:true;foreignKey:ToUserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Notification) TableName() string { return "notifications_notification" }
