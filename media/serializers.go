package media

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
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
		o["hls_url"] = s.MediaURL("", "", key)
	}
	if key := Str(o["thumbnail"]); key != "" {
		o["thumbnail_url"] = s.MediaURL(key, "", "")
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
