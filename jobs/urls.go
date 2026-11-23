package jobs

import (
	. "litloop/backend/internal/httpapi"
)

func Register(server *Server) {
	server.Route("GET", "/tasks/{id:[0-9]+}", true, server.TaskStatus)
	server.Route("GET", "/jobs/api/vacancies", false, server.Vacancies)
	server.Route("POST", "/jobs/api/vacancies", true, server.Vacancies)
	server.Route("GET", "/jobs/api/companies", false, server.Companies)
	server.Route("POST", "/jobs/api/companies", true, server.Companies)
	server.Route("GET,POST", "/jobs/api/resumes", true, server.Resumes)
	server.Route("GET,POST", "/jobs/api/applications", true, server.Application)
}
