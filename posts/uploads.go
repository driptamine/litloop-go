package posts

import (
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"litloop/backend/internal/storage"
	"strings"

	"gorm.io/gorm"
)

func (s *Handler) PostFiles(q *Request, tx *gorm.DB, p models.Post, b map[string]any) {
	if q.R.MultipartForm == nil {
		return
	}
	for _, kind := range []string{"photos", "videos", "tracks"} {
		for _, header := range q.R.MultipartForm.File[kind] {
			file, e := header.Open()
			Check(e)
			upload := func() storage.UploadedFile { defer file.Close(); return s.SaveUploadedFile(q, file, header) }()
			r2, gcs, s3 := upload.Keys()
			name := upload.Name
			var id int64
			switch kind {
			case "photos":
				photo := models.Photo{UserID: Ptr(q.User.ID), R2Key: r2, GCSKey: gcs, S3Key: s3, Filename: name, Status: "attached", FriendlyToken: Token()[:12]}
				Check(tx.Create(&photo).Error)
				id = photo.ID
			case "videos":
				video := models.Video{UserID: Ptr(q.User.ID), R2Key: r2, GCSKey: gcs, S3Key: s3, Filename: name, Status: "attached", Visibility: "private"}
				Check(tx.Create(&video).Error)
				id = video.ID
			case "tracks":
				track := models.Track{UserID: Ptr(q.User.ID), R2Key: r2, GCSKey: gcs, S3Key: s3, Filename: name, Name: name, TrackURI: Token(), Status: "attached"}
				Check(tx.Create(&track).Error)
				id = track.ID
			}
			field := strings.TrimSuffix(kind, "s") + "_ids"
			list, _ := b[field].([]any)
			b[field] = append(list, id)
		}
	}
}
