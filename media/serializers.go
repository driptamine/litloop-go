package media

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"
)

func (s *Handler) MediaJSON(v any) map[string]any {
	o := Object(v)
	r := Str(o["r2_key"])
	g := Str(o["gcs_key"])
	a := Str(o["s3_key"])
	uri := s.MediaURL(r, g, a)
	o["url"] = uri
	o["r2_url"] = uri
	o["gcs_url"] = uri
	o["pk"] = o["id"]
	o["file"] = uri
	o["file_path"] = r
	if r == "" {
		o["file_path"] = g
	}
	if r == "" && g == "" {
		o["file_path"] = a
	}
	if _, ok := v.(models.Video); ok {
		o["video_id"] = o["id"]
		o["name"] = o["title"]
		o["thumbNail"] = o["thumbnail"]
		for _, key := range []string{"likes", "dislikes", "views", "impressions"} {
			o[key+"_count"] = o[key]
		}
	}
	if key := Str(o["hls_s3"]); key != "" {
		if strings.HasPrefix(key, "r2/hls/") {
			o["hls_url"] = s.LocalURL(key)
		} else {
			o["hls_url"] = s.MediaURL("", "", key)
		}
	}
	if key := Str(o["thumbnail"]); key != "" {
		uri := s.MediaURL(strings.TrimPrefix(key, "r2/"), "", "")
		o["thumbnail_url"] = uri
		if uri != "" {
			o["thumbNail"] = uri
		}
	}
	return o
}
func (s *Handler) MediaList(v any) any {
	out := []any{}
	switch list := v.(type) {
	case []models.Photo:
		for _, p := range list {
			out = append(out, s.MediaJSON(p))
		}
	case []models.Video:
		for _, p := range list {
			out = append(out, s.MediaJSON(p))
		}
	}
	return out
}
func safeMedia(v any) any { return v }
