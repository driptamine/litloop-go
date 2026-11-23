package websites

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"net/url"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (handler *Handler) Search(req *Request) any {
	term := strings.TrimSpace(req.R.URL.Query().Get("q"))
	if term == "" {
		term = req.R.URL.Query().Get("query")
	}
	if req.R.Method == "POST" {
		term = strings.TrimSpace(Str(req.Body()["q"]))
	}
	path := req.R.URL.Path
	if path == "/websites/all" || path == "/websites/search" {
		db := handler.DB.Order("created_at DESC")
		if term != "" {
			like := "%" + strings.ToLower(term) + "%"
			db = db.Where("LOWER(title) LIKE ? OR LOWER(body) LIKE ?", like, like)
		}
		return map[string]any{"results": All[models.Website](db)}
	}
	if term == "" {
		Fail(400, "q is required")
	}
	if len(term) > 500 {
		Fail(400, "Query too long")
	}
	images := strings.Contains(path, "images") || strings.Contains(path, "custom")
	brave := strings.Contains(path, "brave")
	if brave && !images {
		var cached models.Query
		if handler.DB.First(&cached, "query_text = ? AND is_crawled = ?", term, true).Error == nil {
			return map[string]any{"query_text": term, "is_crawled": true, "cached": true, "results": All[models.Website](handler.DB.Where("query = ?", term).Order("created_at DESC"))}
		}
	}
	target := ""
	headers := map[string]string{}
	provider := "google"
	switch {
	case brave:
		provider = "brave"
		if handler.Config.BraveKey == "" {
			Fail(503, "BRAVE_API_KEY is not configured")
		}
		kind := "web"
		if images {
			kind = "images"
		}
		target = "https://api.search.brave.com/res/v1/" + kind + "/search?q=" + url.QueryEscape(term)
		headers["X-Subscription-Token"] = handler.Config.BraveKey
	case strings.Contains(path, "bing"):
		provider = "bing"
		if handler.Config.BingKey == "" {
			Fail(503, "BING_API_KEY is not configured")
		}
		kind := "search"
		if images {
			kind = "images/search"
		}
		target = "https://api.bing.microsoft.com/v7.0/" + kind + "?q=" + url.QueryEscape(term)
		headers["Ocp-Apim-Subscription-Key"] = handler.Config.BingKey
	default:
		if handler.Config.GoogleSearchKey == "" || handler.Config.GoogleSearchCX == "" {
			Fail(503, "Google custom search is not configured")
		}
		target = "https://www.googleapis.com/customsearch/v1?" + url.Values{"key": {handler.Config.GoogleSearchKey}, "cx": {handler.Config.GoogleSearchCX}, "q": {term}, "searchType": {"image"}}.Encode()
	}
	payload, ok := handler.FetchJSON(req, "GET", target, nil, headers).(map[string]any)
	if !ok {
		Fail(502, "Invalid search response")
	}
	results := []any{}
	if images {
		switch provider {
		case "google":
			results, _ = payload["items"].([]any)
		case "bing":
			results, _ = payload["value"].([]any)
		case "brave":
			if nested, ok := payload["images"].(map[string]any); ok {
				results, _ = nested["results"].([]any)
			}
			if results == nil {
				results, _ = payload["results"].([]any)
			}
		}
		if results == nil {
			results = []any{}
		}
		for _, value := range results {
			record, ok := value.(map[string]any)
			if !ok {
				continue
			}
			uri := Str(record["url"])
			if uri == "" {
				uri = Str(record["link"])
			}
			if !ValidURL(uri) {
				continue
			}
			thumb := Str(record["thumbnail"])
			if nested, ok := record["thumbnail"].(map[string]any); ok {
				thumb = Str(nested["src"])
			}
			image := models.WebsiteImage{URL: uri, Title: Str(record["title"]), Description: Str(record["description"]), PageURL: Str(record["page_url"]), Thumbnail: thumb, Height: Str(record["height"]), Width: Str(record["width"])}
			Check(handler.DB.Where("url = ?", uri).FirstOrCreate(&image).Error)
		}
		if provider == "brave" {
			return results
		}
		if provider == "google" {
			return map[string]any{"items": results}
		}
		return map[string]any{"results": results}
	}
	key := "web"
	listKey := "results"
	if provider == "bing" {
		key = "webPages"
		listKey = "value"
	}
	if nested, ok := payload[key].(map[string]any); ok {
		results, _ = nested[listKey].([]any)
	}
	if results == nil {
		results = []any{}
	}
	if brave {
		normalized := []any{}
		req.Tx(func(tx *gorm.DB) error {
			for _, entry := range results {
				record, ok := entry.(map[string]any)
				if !ok {
					continue
				}
				uri := Str(record["url"])
				if !ValidURL(uri) {
					continue
				}
				website := models.Website{URL: uri, Title: Str(record["title"]), Body: Str(record["description"]), Favicon: Str(record["favicon"]), Query: term}
				Check(tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "url"}}, DoUpdates: clause.AssignmentColumns([]string{"title", "body", "favicon", "query", "updated_at"})}).Create(&website).Error)
				normalized = append(normalized, map[string]any{"url": website.URL, "title": website.Title, "body": website.Body, "favicon": website.Favicon})
			}
			var query models.Query
			Check(tx.Where("query_text = ?", term).FirstOrCreate(&query, models.Query{QueryText: term}).Error)
			return tx.Model(&query).Update("is_crawled", true).Error
		})
		return map[string]any{"query_text": term, "is_crawled": true, "cached": false, "results": normalized}
	}
	return map[string]any{"results": results}
}
