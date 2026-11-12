// Package todos contains the todos domain models and SQL relationships.
package model

import (
	users "litloop/backend/users/model"
	"time"
)

type Profile struct {
	ID              int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	UserID          int64       `json:"user_id" gorm:"column:user_id;uniqueIndex;index"`
	BackgroundImage string      `json:"background_image" gorm:"column:background_image"`
	UserIDRef       *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Profile) TableName() string { return "todos_profile" }

type TodoList struct {
	ID        int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name      string      `json:"name" gorm:"column:name;size:255"`
	UserID    int64       `json:"user_id" gorm:"column:user_id;index"`
	CreatedAt time.Time   `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	Order     int64       `json:"order" gorm:"column:order;default:0"`
	UserIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (TodoList) TableName() string { return "todos_todolist" }

type Todo struct {
	ID            int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Title         string      `json:"title" gorm:"column:title;size:255"`
	Completed     bool        `json:"completed" gorm:"column:completed;default:false"`
	UserID        *int64      `json:"user_id" gorm:"column:user_id;index"`
	TodoListID    *int64      `json:"todo_list_id" gorm:"column:todo_list_id;index"`
	Order         int64       `json:"order" gorm:"column:order;default:0"`
	UserIDRef     *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:SET NULL"`
	TodoListIDRef *TodoList   `json:"-" gorm:"belongsTo:true;foreignKey:TodoListID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Todo) TableName() string { return "todos_todo" }
