// Package magazines contains the magazines domain models and SQL relationships.
package model

import (
	"time"
)

type Magazine struct {
	ID          int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Title       string    `json:"title" gorm:"column:title;size:255"`
	IssueDate   time.Time `json:"issue_date" gorm:"column:issue_date;type:date"`
	Description string    `json:"description" gorm:"column:description"`
}

func (Magazine) TableName() string { return "magazines_magazine" }

type Cover struct {
	ID             int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	MagazineID     int64     `json:"magazine_id" gorm:"column:magazine_id;uniqueIndex;index"`
	FeaturedPerson string    `json:"featured_person" gorm:"column:featured_person;size:255"`
	ImageURL       string    `json:"image_url" gorm:"column:image_url"`
	Price          float64   `json:"price" gorm:"column:price;type:decimal(6,2)"`
	IsAvailable    bool      `json:"is_available" gorm:"column:is_available;default:true"`
	MagazineIDRef  *Magazine `json:"-" gorm:"belongsTo:true;foreignKey:MagazineID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Cover) TableName() string { return "magazines_cover" }
