package memes

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (s *Handler) Memes(q *Request) any {
	if q.R.URL.Path == "/memes" {
		data := s.FetchJSON(q, "GET", "https://meme-api.com/gimme/20", nil, nil)
		if payload, ok := data.(map[string]any); ok {
			if list, ok := payload["memes"].([]any); ok {
				for _, entry := range list {
					if m, ok := entry.(map[string]any); ok {
						url := Str(m["postLink"])
						if url == "" {
							continue
						}
						preview := models.AsJSON(m["preview"])
						_ = s.DB.Where("meme_id = ?", url).FirstOrCreate(&models.Meme{MemeID: url, Title: Str(m["title"]), URL: Str(m["url"]), PostLink: url, Subreddit: Str(m["subreddit"]), Author: Str(m["author"]), Preview: string(preview)}).Error
					}
				}
			}
		}
		return data
	}
	return q.Page(s.DB.Model(&models.Meme{}).Order("created_at DESC"), &[]models.Meme{}, func(v any) any { return v })
}
