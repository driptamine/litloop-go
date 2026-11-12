// Package jobs contains the jobs domain models and SQL relationships.
package model

import (
	users "litloop/backend/users/model"
	"time"
)

type Company struct {
	ID          int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Name        string    `json:"name" gorm:"column:name;size:255"`
	Description string    `json:"description" gorm:"column:description"`
	Website     string    `json:"website" gorm:"column:website"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
}

func (Company) TableName() string { return "jobs_company" }

type Resume struct {
	ID         int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	UserID     int64       `json:"user_id" gorm:"column:user_id;index"`
	Summary    string      `json:"summary" gorm:"column:summary"`
	Skills     string      `json:"skills" gorm:"column:skills"`
	Experience string      `json:"experience" gorm:"column:experience"`
	Education  string      `json:"education" gorm:"column:education"`
	UserIDRef  *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Resume) TableName() string { return "jobs_resume" }

type JobPosting struct {
	ID           int64     `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Title        string    `json:"title" gorm:"column:title;size:255"`
	Description  string    `json:"description" gorm:"column:description"`
	CompanyID    int64     `json:"company_id" gorm:"column:company_id;index"`
	Location     string    `json:"location" gorm:"column:location;size:255"`
	PostedAt     time.Time `json:"posted_at" gorm:"column:posted_at;autoCreateTime"`
	CompanyIDRef *Company  `json:"-" gorm:"belongsTo:true;foreignKey:CompanyID;references:ID;constraint:OnDelete:CASCADE"`
}

func (JobPosting) TableName() string { return "jobs_jobposting" }

type Application struct {
	ID             int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	JobID          int64       `json:"job_id" gorm:"column:job_id;index"`
	ApplicantID    int64       `json:"applicant_id" gorm:"column:applicant_id;index"`
	ResumeID       *int64      `json:"resume_id" gorm:"column:resume_id;index"`
	AppliedAt      time.Time   `json:"applied_at" gorm:"column:applied_at;autoCreateTime"`
	JobIDRef       *JobPosting `json:"-" gorm:"belongsTo:true;foreignKey:JobID;references:ID;constraint:OnDelete:CASCADE"`
	ApplicantIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:ApplicantID;references:ID;constraint:OnDelete:CASCADE"`
	ResumeIDRef    *Resume     `json:"-" gorm:"belongsTo:true;foreignKey:ResumeID;references:ID;constraint:OnDelete:SET NULL"`
}

func (Application) TableName() string { return "jobs_application" }

type CoverLetter struct {
	ID               int64        `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ApplicationID    int64        `json:"application_id" gorm:"column:application_id;uniqueIndex;index"`
	Content          string       `json:"content" gorm:"column:content"`
	ApplicationIDRef *Application `json:"-" gorm:"belongsTo:true;foreignKey:ApplicationID;references:ID;constraint:OnDelete:CASCADE"`
}

func (CoverLetter) TableName() string { return "jobs_coverletter" }
