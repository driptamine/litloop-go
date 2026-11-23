package jobs

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"

	"gorm.io/gorm"
)

func (handler *Handler) Vacancies(req *Request) any {
	if req.R.Method == "GET" {
		out := []any{}
		for _, job := range All[models.JobPosting](handler.DB.Order("posted_at DESC")) {
			jobJSON := Object(job)
			jobJSON["company"] = Get[models.Company](handler.DB, job.CompanyID)
			out = append(out, jobJSON)
		}
		return out
	}
	req.Admin()
	body := req.Body()
	company := Get[models.Company](handler.DB, Num(body["company_id"]))
	title := strings.TrimSpace(Str(body["title"]))
	if title == "" {
		Fail(400, "title is required")
	}
	job := models.JobPosting{CompanyID: company.ID, Title: title, Description: Str(body["description"]), Location: Str(body["location"])}
	Check(handler.DB.Create(&job).Error)
	req.Status = 201
	return job
}
func (handler *Handler) Companies(req *Request) any {
	if req.R.Method == "GET" {
		return All[models.Company](handler.DB.Order("name"))
	}
	req.Admin()
	body := req.Body()
	name := Str(body["name"])
	if name == "" {
		Fail(400, "name is required")
	}
	company := models.Company{Name: name, Description: Str(body["description"]), Website: Str(body["website"])}
	Check(handler.DB.Create(&company).Error)
	req.Status = 201
	return company
}
func (handler *Handler) Resumes(req *Request) any {
	if req.R.Method == "GET" {
		return All[models.Resume](handler.DB.Where("user_id = ?", req.User.ID))
	}
	body := req.Body()
	resume := models.Resume{UserID: req.User.ID, Summary: Str(body["summary"]), Skills: Str(body["skills"]), Experience: Str(body["experience"]), Education: Str(body["education"])}
	Check(handler.DB.Create(&resume).Error)
	req.Status = 201
	return resume
}
func (handler *Handler) Application(req *Request) any {
	if req.R.Method == "GET" {
		return All[models.Application](handler.DB.Where("applicant_id = ?", req.User.ID))
	}
	body := req.Body()
	job := Get[models.JobPosting](handler.DB, Num(body["job_id"]))
	application := models.Application{JobID: job.ID, ApplicantID: req.User.ID}
	if body["resume_id"] != nil {
		resume := Get[models.Resume](handler.DB.Where("user_id = ?", req.User.ID), Num(body["resume_id"]))
		application.ResumeID = Ptr(resume.ID)
	}
	req.Tx(func(tx *gorm.DB) error {
		Check(tx.Create(&application).Error)
		if text := Str(body["cover_letter"]); text != "" {
			return tx.Create(&models.CoverLetter{ApplicationID: application.ID, Content: text}).Error
		}
		return nil
	})
	req.Status = 201
	return map[string]any{"success": true, "application_id": application.ID, "message": "Application submitted successfully"}
}
