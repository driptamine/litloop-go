package tracks

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"

	"gorm.io/gorm"
)

func trackLikeStatus(db *gorm.DB, trackID, userID int64) bool {
	var like models.TrackLike
	return db.First(&like, "track_id = ? AND liked_by_id = ?", trackID, userID).Error == nil
}

// TrackLikeToggle likes or unlikes a track; the liked set is the user's
// audio library.
func (handler *Handler) TrackLikeToggle(req *Request) any {
	id := req.ID("id")
	Get[models.Track](handler.DB, id)
	liked := false
	req.Tx(func(tx *gorm.DB) error {
		var like models.TrackLike
		err := tx.First(&like, "track_id = ? AND liked_by_id = ?", id, req.User.ID).Error
		if err == gorm.ErrRecordNotFound {
			Check(tx.Create(&models.TrackLike{TrackID: id, LikedByID: req.User.ID}).Error)
			liked = true
		} else {
			Check(err)
			Check(tx.Delete(&like).Error)
		}
		return nil
	})
	return map[string]any{"liked": liked}
}

func (handler *Handler) TrackLikeStatus(req *Request) any {
	id := req.ID("id")
	Get[models.Track](handler.DB, id)
	return map[string]any{"liked": trackLikeStatus(handler.DB, id, req.User.ID)}
}

func (handler *Handler) TrackLikedList(req *Request) any {
	var likes []models.TrackLike
	Check(handler.DB.Where("liked_by_id = ?", req.User.ID).Order("id DESC").Find(&likes).Error)
	out := make([]any, 0, len(likes))
	for _, like := range likes {
		track := Get[models.Track](handler.DB, like.TrackID)
		if entry, ok := handler.TrackJSON(track).(map[string]any); ok {
			entry["liked"] = true
			out = append(out, entry)
		}
	}
	return map[string]any{"tracks": out}
}
