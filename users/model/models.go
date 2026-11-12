// Package users contains the users domain models and SQL relationships.
package model

import (
	"time"
)

type User struct {
	ID          int64      `json:"id"    gorm:"column:id;primaryKey;autoIncrement"`
	Username    string     `json:"username"  gorm:"column:username;uniqueIndex;index;size:255"`
	Email       string     `json:"email"   gorm:"column:email;uniqueIndex;index;size:255"`
	Avatar      string     `json:"avatar"   gorm:"column:avatar;size:500"`
	IsVerified  bool       `json:"is_verified"  gorm:"column:is_verified;default:false"`
	IsActive    bool       `json:"is_active"  gorm:"column:is_active;default:true"`
	IsStaff     bool       `json:"is_staff"  gorm:"column:is_staff;default:false"`
	CreatedAt   time.Time  `json:"created_at"  gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at"  gorm:"column:updated_at;autoUpdateTime"`
	Password    string     `json:"-"    gorm:"column:password;size:128"`
	LastLogin   *time.Time `json:"last_login"  gorm:"column:last_login"`
	IsSuperuser bool       `json:"is_superuser" gorm:"column:is_superuser;default:false"`
}

func (User) TableName() string { return "users_user" }
