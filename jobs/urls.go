package jobs

import (
	. "litloop/backend/internal/httpapi"
)

func Register(s *Server) {
	s.Route("GET", "/tasks/{id:[0-9]+}", true, s.TaskStatus)
	s.Route("GET", "/jobs/api/vacancies", false, s.Vacancies)
	s.Route("POST", "/jobs/api/vacancies", true, s.Vacancies)
	s.Route("GET", "/jobs/api/companies", false, s.Companies)
	s.Route("POST", "/jobs/api/companies", true, s.Companies)
	s.Route("GET,POST", "/jobs/api/resumes", true, s.Resumes)
	s.Route("GET,POST", "/jobs/api/applications", true, s.Application)
}
