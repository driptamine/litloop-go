package jobs

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) Vacancies(q *Request) any {
	if q.R.Method == "GET" {
		out := []any{}
		for _, j := range All[models.JobPosting](s.DB.Order("posted_at DESC")) {
			o := Object(j)
			o["company"] = Get[models.Company](s.DB, j.CompanyID)
			out = append(out, o)
		}
		return out
	}
	q.Admin()
	b := q.Body()
	company := Get[models.Company](s.DB, Num(b["company_id"]))
	title := strings.TrimSpace(Str(b["title"]))
	if title == "" {
		Fail(400, "title is required")
	}
	j := models.JobPosting{CompanyID: company.ID, Title: title, Description: Str(b["description"]), Location: Str(b["location"])}
	Check(s.DB.Create(&j).Error)
	q.Status = 201
	return j
}
func (s *Handler) Companies(q *Request) any {
	if q.R.Method == "GET" {
		return All[models.Company](s.DB.Order("name"))
	}
	q.Admin()
	b := q.Body()
	name := Str(b["name"])
	if name == "" {
		Fail(400, "name is required")
	}
	c := models.Company{Name: name, Description: Str(b["description"]), Website: Str(b["website"])}
	Check(s.DB.Create(&c).Error)
	q.Status = 201
	return c
}
func (s *Handler) Resumes(q *Request) any {
	if q.R.Method == "GET" {
		return All[models.Resume](s.DB.Where("user_id = ?", q.User.ID))
	}
	b := q.Body()
	r := models.Resume{UserID: q.User.ID, Summary: Str(b["summary"]), Skills: Str(b["skills"]), Experience: Str(b["experience"]), Education: Str(b["education"])}
	Check(s.DB.Create(&r).Error)
	q.Status = 201
	return r
}
func (s *Handler) Application(q *Request) any {
	if q.R.Method == "GET" {
		return All[models.Application](s.DB.Where("applicant_id = ?", q.User.ID))
	}
	b := q.Body()
	job := Get[models.JobPosting](s.DB, Num(b["job_id"]))
	a := models.Application{JobID: job.ID, ApplicantID: q.User.ID}
	if b["resume_id"] != nil {
		resume := Get[models.Resume](s.DB.Where("user_id = ?", q.User.ID), Num(b["resume_id"]))
		a.ResumeID = Ptr(resume.ID)
	}
	q.Tx(func(tx *gorm.DB) error {
		Check(tx.Create(&a).Error)
		if text := Str(b["cover_letter"]); text != "" {
			return tx.Create(&models.CoverLetter{ApplicationID: a.ID, Content: text}).Error
		}
		return nil
	})
	q.Status = 201
	return map[string]any{"success": true, "application_id": a.ID, "message": "Application submitted successfully"}
}
