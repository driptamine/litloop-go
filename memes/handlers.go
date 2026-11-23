package memes

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

func (handler *Handler) Memes(req *Request) any {
	if req.R.URL.Path == "/memes" {
		data := handler.FetchJSON(req, "GET", "https://meme-api.com/gimme/20", nil, nil)
		if payload, ok := data.(map[string]any); ok {
			if list, ok := payload["memes"].([]any); ok {
				for _, entry := range list {
					if post, ok := entry.(map[string]any); ok {
						url := Str(post["postLink"])
						if url == "" {
							continue
						}
						preview := models.AsJSON(post["preview"])
						_ = handler.DB.Where("meme_id = ?", url).FirstOrCreate(&models.Meme{MemeID: url, Title: Str(post["title"]), URL: Str(post["url"]), PostLink: url, Subreddit: Str(post["subreddit"]), Author: Str(post["author"]), Preview: string(preview)}).Error
					}
				}
			}
		}
		return data
	}
	return req.Page(handler.DB.Model(&models.Meme{}).Order("created_at DESC"), &[]models.Meme{}, func(value any) any { return value })
}
