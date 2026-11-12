package users

import (
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (s *Handler) UserMediaKind(q *Request, kind string) any {
	ctx := chiRouteContext(q)
	ctx.URLParams.Add("kind", kind)
	return s.UserMedia(q)
}
func chiRouteContext(q *Request) *chi.Context { return chi.RouteContext(q.R.Context()) }
func (s *Handler) UserMedia(q *Request) any {
	var u models.User
	key := q.Param("username")
	if id, e := strconv.ParseInt(key, 10, 64); e == nil {
		Check(s.DB.First(&u, id).Error)
	} else {
		Check(s.DB.First(&u, "username = ?", key).Error)
	}
	kind := q.Param("kind")
	db := s.DB
	var result map[string]any
	switch kind {
	case "posts":
		out := []any{}
		for _, p := range All[models.Post](db.Where("author_id = ?", u.ID).Order("created_at DESC")) {
			out = append(out, s.PostJSON(p, q.User))
		}
		return map[string]any{"username": u.Username, "count": len(out), "results": out, "posts": out}
	case "photos":
		result = q.PageSize(db.Model(&models.Photo{}).Where("user_id = ?", u.ID).Order("id DESC"), &[]models.Photo{}, func(v any) any {
			out := []any{}
			for _, p := range *v.(*[]models.Photo) {
				o := s.MediaJSON(p)
				o["pk"] = p.ID
				o["image"] = o["url"]
				o["file_path"] = o["url"]
				name := p.Title
				if name == "" {
					name = p.Filename
				}
				if name == "" {
					name = fmt.Sprintf("Photo %d", p.ID)
				}
				o["name"] = name
				o["title"] = name
				out = append(out, o)
			}
			return out
		}, 30).(map[string]any)
	case "videos":
		query := db.Model(&models.Video{}).Where("user_id = ?", u.ID)
		if q.User == nil || q.User.ID != u.ID {
			query = query.Where("visibility = ? OR id IN (?)", "public", db.Model(&models.PostVideo{}).Select("video_id"))
		}
		result = q.PageSize(query.Order("id DESC"), &[]models.Video{}, func(v any) any { return s.MediaList(*v.(*[]models.Video)) }, 30).(map[string]any)
	case "albums":
		result = q.PageSize(db.Model(&models.PhotoAlbum{}).Where("user_id = ?", u.ID).Order("add_date DESC"), &[]models.PhotoAlbum{}, func(v any) any {
			out := []any{}
			for _, a := range *v.(*[]models.PhotoAlbum) {
				o := Object(a)
				var n int64
				Check(db.Model(&models.PhotoAlbumItem{}).Where("photo_album_id = ?", a.ID).Count(&n).Error)
				o["photo_count"] = n
				o["thumbnail_url"] = nil
				var item models.PhotoAlbumItem
				if db.Where("photo_album_id = ?", a.ID).Order("ordering").First(&item).Error == nil {
					photo := Get[models.Photo](db, item.PhotoID)
					o["thumbnail_url"] = s.MediaURL(photo.R2Key, photo.GCSKey, photo.S3Key)
				}
				out = append(out, o)
			}
			return out
		}, 30).(map[string]any)
	case "tracks":
		result = q.Page(db.Model(&models.Track{}).Where("user_id = ?", u.ID).Order("id DESC"), &[]models.Track{}, func(v any) any {
			out := []any{}
			for _, t := range *v.(*[]models.Track) {
				out = append(out, s.TrackJSON(t))
			}
			return out
		}).(map[string]any)
		result["return_code"] = "0000"
		result["return_message"] = "Success"
	case "watchlist":
		Owned(u.ID, q.User.ID)
		return s.PrivateWatchlist(q, u)
	default:
		Fail(404, "Unknown resource")
	}
	result[kind] = result["items"]
	return result
}
func (s *Handler) PrivateWatchlist(q *Request, u models.User) any {
	page, size := Pagination(q, 10, "per_page")
	var total int64
	db := s.DB.Model(&models.WatchlistItem{}).Where("user_id = ?", u.ID)
	Check(db.Count(&total).Error)
	out := []any{}
	for _, w := range All[models.WatchlistItem](db.Order("id DESC").Offset((page - 1) * size).Limit(size)) {
		m := Get[models.ImdbMovie](s.DB, w.MovieID)
		o := map[string]any{"movie_id": m.ID, "title": m.Title, "imdb_created_at": Date(m.CreatedAt)}
		out = append(out, o)
	}
	result := map[string]any{"page": page, "per_page": size, "total_items": total, "total_pages": (total + int64(size) - 1) / int64(size), "watchlist": out}
	if _, e := strconv.ParseInt(q.Param("username"), 10, 64); e == nil {
		result["user_id"] = u.ID
	} else {
		result["username"] = u.Username
	}
	return result
}
