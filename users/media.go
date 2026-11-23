package users

import (
	"fmt"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (handler *Handler) UserMediaKind(req *Request, kind string) any {
	ctx := chiRouteContext(req)
	ctx.URLParams.Add("kind", kind)
	return handler.UserMedia(req)
}
func chiRouteContext(req *Request) *chi.Context	{ return chi.RouteContext(req.R.Context()) }
func (handler *Handler) UserMedia(req *Request) any {
	var user models.User
	key := req.Param("username")
	if id, err := strconv.ParseInt(key, 10, 64); err == nil {
		Check(handler.DB.First(&user, id).Error)
	} else {
		Check(handler.DB.First(&user, "username = ?", key).Error)
	}
	kind := req.Param("kind")
	db := handler.DB
	var result map[string]any
	switch kind {
	case "posts":
		out := []any{}
		for _, post := range All[models.Post](db.Where("author_id = ?", user.ID).Order("created_at DESC")) {
			out = append(out, handler.PostJSON(post, req.User))
		}
		return map[string]any{"username": user.Username, "count": len(out), "results": out, "posts": out}
	case "photos":
		result = req.PageSize(db.Model(&models.Photo{}).Where("user_id = ?", user.ID).Order("id DESC"), &[]models.Photo{}, func(value any) any {
			out := []any{}
			for _, photo := range *value.(*[]models.Photo) {
				mediaJSON := handler.MediaJSON(photo)
				mediaJSON["pk"] = photo.ID
				mediaJSON["image"] = mediaJSON["url"]
				mediaJSON["file_path"] = mediaJSON["url"]
				name := photo.Title
				if name == "" {
					name = photo.Filename
				}
				if name == "" {
					name = fmt.Sprintf("Photo %d", photo.ID)
				}
				mediaJSON["name"] = name
				mediaJSON["title"] = name
				out = append(out, mediaJSON)
			}
			return out
		}, 30).(map[string]any)
	case "videos":
		query := db.Model(&models.Video{}).Where("user_id = ?", user.ID)
		if req.User == nil || req.User.ID != user.ID {
			query = query.Where("visibility = ? OR id IN (?)", "public", db.Model(&models.PostVideo{}).Select("video_id"))
		}
		result = req.PageSize(query.Order("id DESC"), &[]models.Video{}, func(value any) any { return handler.MediaList(*value.(*[]models.Video)) }, 30).(map[string]any)
	case "albums":
		result = req.PageSize(db.Model(&models.PhotoAlbum{}).Where("user_id = ?", user.ID).Order("add_date DESC"), &[]models.PhotoAlbum{}, func(value any) any {
			out := []any{}
			for _, album := range *value.(*[]models.PhotoAlbum) {
				mediaJSON := Object(album)
				var count int64
				Check(db.Model(&models.PhotoAlbumItem{}).Where("photo_album_id = ?", album.ID).Count(&count).Error)
				mediaJSON["photo_count"] = count
				mediaJSON["thumbnail_url"] = nil
				var item models.PhotoAlbumItem
				if db.Where("photo_album_id = ?", album.ID).Order("ordering").First(&item).Error == nil {
					photo := Get[models.Photo](db, item.PhotoID)
					mediaJSON["thumbnail_url"] = handler.MediaURL(photo.R2Key, photo.GCSKey, photo.S3Key)
				}
				out = append(out, mediaJSON)
			}
			return out
		}, 30).(map[string]any)
	case "tracks":
		result = req.Page(db.Model(&models.Track{}).Where("user_id = ?", user.ID).Order("id DESC"), &[]models.Track{}, func(value any) any {
			out := []any{}
			for _, track := range *value.(*[]models.Track) {
				out = append(out, handler.TrackJSON(track))
			}
			return out
		}).(map[string]any)
		result["return_code"] = "0000"
		result["return_message"] = "Success"
	case "watchlist":
		Owned(user.ID, req.User.ID)
		return handler.PrivateWatchlist(req, user)
	default:
		Fail(404, "Unknown resource")
	}
	result[kind] = result["items"]
	return result
}
func (handler *Handler) PrivateWatchlist(req *Request, user models.User) any {
	page, size := Pagination(req, 10, "per_page")
	var total int64
	db := handler.DB.Model(&models.WatchlistItem{}).Where("user_id = ?", user.ID)
	Check(db.Count(&total).Error)
	out := []any{}
	for _, watchlistItem := range All[models.WatchlistItem](db.Order("id DESC").Offset((page - 1) * size).Limit(size)) {
		movie := Get[models.ImdbMovie](handler.DB, watchlistItem.MovieID)
		mediaJSON := map[string]any{"movie_id": movie.ID, "title": movie.Title, "imdb_created_at": Date(movie.CreatedAt)}
		out = append(out, mediaJSON)
	}
	result := map[string]any{"page": page, "per_page": size, "total_items": total, "total_pages": (total + int64(size) - 1) / int64(size), "watchlist": out}
	if _, err := strconv.ParseInt(req.Param("username"), 10, 64); err == nil {
		result["user_id"] = user.ID
	} else {
		result["username"] = user.Username
	}
	return result
}
