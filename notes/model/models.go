// Package notes contains the notes domain models and SQL relationships.
package model

import (
	"litloop/backend/internal/types"
	users "litloop/backend/users/model"
	"time"
)

type JSON = types.JSON

type Page struct {
	ID          int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Title       string      `json:"title" gorm:"column:title;size:200"`
	ParentID    *int64      `json:"parent_id" gorm:"column:parent_id;index"`
	IsLocked    bool        `json:"is_locked" gorm:"column:is_locked;default:false"`
	IsPrivate   bool        `json:"is_private" gorm:"column:is_private;default:true"`
	AuthorID    *int64      `json:"author_id" gorm:"column:author_id;index"`
	CreatedAt   time.Time   `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time   `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	Lft         int64       `json:"lft" gorm:"column:lft;default:0"`
	Rght        int64       `json:"rght" gorm:"column:rght;default:0"`
	TreeID      int64       `json:"tree_id" gorm:"column:tree_id;default:0"`
	Level       int64       `json:"level" gorm:"column:level;default:0"`
	ParentIDRef *Page       `json:"-" gorm:"belongsTo:true;foreignKey:ParentID;references:ID;constraint:OnDelete:CASCADE"`
	AuthorIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:AuthorID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Page) TableName() string { return "notes_page" }

type PageTag struct {
	ID        int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PageID    int64     `json:"page_id" gorm:"column:page_id;index;uniqueIndex:uq_notes_pagetag"`
	Name      string    `json:"name" gorm:"column:name;uniqueIndex:uq_notes_pagetag;size:50"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	PageIDRef *Page     `json:"-" gorm:"belongsTo:true;foreignKey:PageID;references:ID;constraint:OnDelete:CASCADE"`
}

func (PageTag) TableName() string { return "notes_pagetag" }

type Block struct {
	ID        int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PageID    int64  `json:"page_id" gorm:"column:page_id;index"`
	Type      string `json:"type" gorm:"column:type;size:20;default:text"`
	Content   string `json:"content" gorm:"column:content"`
	Order     int64  `json:"order" gorm:"column:order;default:0"`
	PageIDRef *Page  `json:"-" gorm:"belongsTo:true;foreignKey:PageID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Block) TableName() string { return "notes_block" }

type BlockTable struct {
	ID         int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	BlockID    int64  `json:"block_id" gorm:"column:block_id;uniqueIndex;index"`
	Columns    JSON   `json:"columns" gorm:"column:columns;type:json"`
	Rows       JSON   `json:"rows" gorm:"column:rows;type:json"`
	BlockIDRef *Block `json:"-" gorm:"belongsTo:true;foreignKey:BlockID;references:ID;constraint:OnDelete:CASCADE"`
}

func (BlockTable) TableName() string { return "notes_blocktable" }
