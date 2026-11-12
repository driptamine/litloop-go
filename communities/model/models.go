// Package communities contains the communities domain models and SQL relationships.
package model

import (
	posts "litloop/backend/posts/model"
	users "litloop/backend/users/model"
	"time"
)

type Community struct {
	ID             int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name           string      `json:"name" gorm:"column:name;uniqueIndex;size:100"`
	Handle         string      `json:"handle" gorm:"column:handle;uniqueIndex;size:100"`
	Description    string      `json:"description" gorm:"column:description"`
	Icon           string      `json:"icon" gorm:"column:icon;size:500"`
	Banner         string      `json:"banner" gorm:"column:banner;size:500"`
	CreatedByID    int64       `json:"created_by_id" gorm:"column:created_by_id;index"`
	CreatedAt      time.Time   `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	CreatedByIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:CreatedByID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Community) TableName() string { return "communities_community" }

type CommunityMembership struct {
	ID             int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	UserID         int64       `json:"user_id" gorm:"column:user_id;index;uniqueIndex:uq_communities_communitymembership"`
	CommunityID    int64       `json:"community_id" gorm:"column:community_id;index;uniqueIndex:uq_communities_communitymembership"`
	Role           string      `json:"role" gorm:"column:role;size:20;default:member"`
	JoinedAt       time.Time   `json:"joined_at" gorm:"column:joined_at;autoCreateTime"`
	UserIDRef      *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	CommunityIDRef *Community  `json:"-" gorm:"belongsTo:true;foreignKey:CommunityID;references:ID;constraint:OnDelete:CASCADE"`
}

func (CommunityMembership) TableName() string { return "communities_communitymembership" }

type CommunityPost struct {
	ID               int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	PostID           int64       `json:"post_id" gorm:"column:post_id;uniqueIndex;index"`
	CommunityID      int64       `json:"community_id" gorm:"column:community_id;index"`
	RequestedByID    int64       `json:"requested_by_id" gorm:"column:requested_by_id;index"`
	Status           string      `json:"status" gorm:"column:status;size:20;default:pending"`
	ReviewedByID     *int64      `json:"reviewed_by_id" gorm:"column:reviewed_by_id;index"`
	ReviewedAt       *time.Time  `json:"reviewed_at" gorm:"column:reviewed_at"`
	CreatedAt        time.Time   `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	PostIDRef        *posts.Post `json:"-" gorm:"belongsTo:true;foreignKey:PostID;references:ID;constraint:OnDelete:CASCADE"`
	CommunityIDRef   *Community  `json:"-" gorm:"belongsTo:true;foreignKey:CommunityID;references:ID;constraint:OnDelete:CASCADE"`
	RequestedByIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:RequestedByID;references:ID;constraint:OnDelete:CASCADE"`
	ReviewedByIDRef  *users.User `json:"-" gorm:"belongsTo:true;foreignKey:ReviewedByID;references:ID;constraint:OnDelete:SET NULL"`
}

func (CommunityPost) TableName() string { return "communities_communitypost" }
