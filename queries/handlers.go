package queries

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"net/url"
	"strings"

	"gorm.io/gorm/clause"
)

func (s *Handler) Query(q *Request) any {
	term := strings.TrimSpace(q.R.URL.Query().Get("q"))
	if term == "" {
		term = q.R.URL.Query().Get("query")
	}
	path := q.R.URL.Path
	if strings.HasPrefix(path, "/queries/google/all") {
		if strings.HasSuffix(path, "/search") && term == "" {
			Fail(400, "q is required")
		}
		db := s.DB.Order("created_at DESC")
		if term != "" {
			like := "%" + strings.ToLower(term) + "%"
			db = db.Where("LOWER(query_text) LIKE ? OR LOWER(suggestion) LIKE ?", like, like)
		}
		return map[string]any{"results": All[models.GoogleQuery](db)}
	}
	if term == "" {
		Fail(400, "q is required")
	}
	if len(term) > 500 {
		Fail(400, "Query too long")
	}
	target := ""
	headers := map[string]string{}
	if strings.Contains(path, "brave") {
		if s.Config.BraveKey == "" {
			Fail(503, "BRAVE_API_KEY is not configured")
		}
		kind := "suggest"
		if strings.Contains(path, "images") {
			kind = "images"
		}
		target = "https://api.search.brave.com/res/v1/" + kind + "/search?q=" + url.QueryEscape(term)
		headers["X-Subscription-Token"] = s.Config.BraveKey
	} else if strings.Contains(path, "bing") {
		if s.Config.BingKey == "" {
			Fail(503, "BING_API_KEY is not configured")
		}
		target = "https://api.bing.microsoft.com/v7.0/suggestions?q=" + url.QueryEscape(term)
		headers["Ocp-Apim-Subscription-Key"] = s.Config.BingKey
	} else {
		target = "https://suggestqueries.google.com/complete/search?client=firefox&q=" + url.QueryEscape(term)
	}
	data := s.FetchJSON(q, "GET", target, nil, headers)
	if strings.Contains(path, "google") {
		list, ok := data.([]any)
		if !ok || len(list) < 2 {
			Fail(502, "Invalid suggestions response")
		}
		words, ok := list[1].([]any)
		if !ok {
			Fail(502, "Invalid suggestions response")
		}
		results := []any{}
		for _, word := range words {
			Check(s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.GoogleQuery{QueryText: term, Suggestion: Str(word)}).Error)
			results = append(results, map[string]any{"query": word})
		}
		return map[string]any{"results": results}
	}
	if strings.Contains(path, "brave") {
		payload, ok := data.(map[string]any)
		if !ok {
			Fail(502, "Invalid Brave response")
		}
		results := payload["results"]
		if results == nil {
			results = []any{}
		}
		return map[string]any{"query": term, "query_text": term, "results": results}
	}
	return data
}
