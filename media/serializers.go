package media

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strings"
)

func (handler *Handler) MediaJSON(value any) map[string]any {
	out := Object(value)
	r2Key := Str(out["r2_key"])
	gcsKey := Str(out["gcs_key"])
	s3Key := Str(out["s3_key"])
	uri := handler.MediaURL(r2Key, gcsKey, s3Key)
	out["url"] = uri
	out["r2_url"] = uri
	out["gcs_url"] = uri
	out["pk"] = out["id"]
	out["file"] = uri
	out["file_path"] = r2Key
	if r2Key == "" {
		out["file_path"] = gcsKey
	}
	if r2Key == "" && gcsKey == "" {
		out["file_path"] = s3Key
	}
	if _, ok := value.(models.Video); ok {
		out["video_id"] = out["id"]
		out["name"] = out["title"]
		out["thumbNail"] = out["thumbnail"]
		for _, key := range []string{"likes", "dislikes", "views", "impressions"} {
			out[key+"_count"] = out[key]
		}
	}
	if key := Str(out["hls_s3"]); key != "" {
		if strings.HasPrefix(key, "r2/hls/") {
			out["hls_url"] = handler.LocalURL(key)
		} else {
			out["hls_url"] = handler.MediaURL("", "", key)
		}
	}
	if key := Str(out["thumbnail"]); key != "" {
		uri := handler.MediaURL(strings.TrimPrefix(key, "r2/"), "", "")
		out["thumbnail_url"] = uri
		if uri != "" {
			out["thumbNail"] = uri
		}
	}
	return out
}
func (handler *Handler) MediaList(value any) any {
	out := []any{}
	switch list := value.(type) {
	case []models.Photo:
		for _, photo := range list {
			out = append(out, handler.MediaJSON(photo))
		}
	case []models.Video:
		for _, video := range list {
			out = append(out, handler.MediaJSON(video))
		}
	}
	return out
}
func safeMedia(value any) any	{ return value }
