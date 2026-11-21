package media

import (
	"io"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"litloop/backend/internal/storage"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gorm.io/gorm/clause"
)

func (s *Handler) MultipartUpload(q *Request) (string, string, string) {
	q.R.Body = http.MaxBytesReader(q.W, q.R.Body, 64<<20)
	if e := q.R.ParseMultipartForm(8 << 20); e != nil {
		Fail(400, "Invalid multipart upload or file exceeds 64 MiB")
	}
	if q.R.MultipartForm != nil {
		defer q.R.MultipartForm.RemoveAll()
	}
	f, head, e := q.R.FormFile("file")
	if e != nil {
		f, head, e = q.R.FormFile("avatar")
	}
	if e != nil {
		f, head, e = q.R.FormFile("image")
	}
	for _, field := range []string{"icon", "banner", "background_image"} {
		if e == nil {
			break
		}
		f, head, e = q.R.FormFile(field)
	}
	if e != nil {
		Fail(400, "file is required")
	}
	defer f.Close()
	return s.SaveUploadedFile(q, f, head)
}
func (s *Handler) SaveUploadedFile(q *Request, f multipart.File, head *multipart.FileHeader) (string, string, string) {
	filename := filepath.Base(head.Filename)
	chatUpload := q.R.URL.Path == "/chats/upload" && s.Config.R2Bucket != ""
	prefix := "attachments/"
	if chatUpload {
		prefix = storage.ChatAttachmentPrefix(q.User.ID)
	}
	name := Token() + strings.ToLower(filepath.Ext(filename))
	if chatUpload && (strings.ContainsAny(name, `/\`) || strings.Contains(name, "..")) {
		name = strings.SplitN(name, ".", 2)[0]
	}
	key := prefix + name
	typ := head.Header.Get("Content-Type")
	sample := make([]byte, 512)
	n, _ := io.ReadFull(f, sample)
	detected := http.DetectContentType(sample[:n])
	if !strings.HasPrefix(detected, "application/octet-stream") {
		typ = detected
	}
	_, err := f.Seek(0, io.SeekStart)
	Check(err)
	provider := "local"
	if chatUpload || strings.Contains(q.R.URL.Path, "/r2") {
		provider = "r2"
	} else if strings.Contains(q.R.URL.Path, "/gcs") {
		provider = "gcs"
	}
	stableImage := strings.Contains(q.R.URL.Path, "avatar") || strings.Contains(q.R.URL.Path, "/icon/") || strings.Contains(q.R.URL.Path, "/banner/") || strings.Contains(q.R.URL.Path, "background-image")
	if stableImage {
		if !strings.HasPrefix(detected, "image/") || detected == "image/svg+xml" {
			Fail(400, "A raster image is required")
		}
	}
	if provider == "local" {
		path := filepath.Join(s.Config.MediaDir, key)
		Check(os.MkdirAll(filepath.Dir(path), 0755))
		out, e := os.Create(path)
		Check(e)
		_, e = io.Copy(out, f)
		closeErr := out.Close()
		Check(e)
		Check(closeErr)
		key = "local/" + key
	} else {
		Check(s.StorageFor(provider).Put(q.R.Context(), key, typ, f))
	}
	uri := ""
	if provider == "local" {
		uri = s.LocalURL(strings.TrimPrefix(key, "local/"))
	} else if chatUpload {
		uri = storage.PresignedURL(s.Config, "r2", key)
		if uri == "" {
			Fail(502, "Could not sign the chat attachment URL")
		}
	} else if provider == "r2" {
		uri = s.MediaURL(key, "", "")
	} else {
		uri = s.MediaURL("", key, "")
	}
	if stableImage {
		a := models.Asset{ID: Token(), UserID: q.User.ID, Key: key, Provider: provider, ContentType: detected}
		Check(s.DB.Create(&a).Error)
		uri = strings.TrimRight(s.Config.PublicURL, "/") + "/assets/" + a.ID
	}
	return key, uri, filename
}
func (s *Handler) Attachment(q *Request) any {
	if strings.HasPrefix(q.R.URL.Path, "/communities/") {
		s.CommunityRole(s.Community(q), q.User, false)
	}
	key, uri, name := s.MultipartUpload(q)
	if strings.Contains(q.R.URL.Path, "avatar") {
		Check(s.DB.Model(q.User).Update("avatar", uri).Error)
		return map[string]any{"avatar": uri, "user": Get[models.User](s.DB, q.User.ID)}
	}
	if strings.Contains(q.R.URL.Path, "background-image") {
		p := models.Profile{UserID: q.User.ID, BackgroundImage: uri}
		Check(s.DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"background_image"})}).Create(&p).Error)
		return map[string]any{"background_image_url": uri}
	}
	if strings.HasPrefix(q.R.URL.Path, "/communities/") {
		c := s.Community(q)
		s.CommunityRole(c, q.User, false)
		col := "icon"
		if strings.Contains(q.R.URL.Path, "/banner/") {
			col = "banner"
		}
		Check(s.DB.Model(&c).Update(col, uri).Error)
		return s.CommunityJSON(Get[models.Community](s.DB, c.ID), q.User)
	}
	if strings.Contains(q.R.URL.Path, "/voice/") {
		duration, _ := strconv.ParseFloat(q.R.FormValue("duration"), 64)
		if duration < 0 {
			Fail(400, "Invalid duration")
		}
		v := models.VoiceMessage{UserID: Ptr(q.User.ID), Filename: name, Duration: duration, TranscriptionStatus: "not_requested"}
		if strings.Contains(q.R.URL.Path, "/r2") {
			v.R2Key = key
		} else {
			v.S3Key = key
		}
		Check(s.DB.Create(&v).Error)
		q.Status = 201
		return map[string]any{"success": true, "id": v.ID, "url": uri, "duration": duration}
	}
	typ := "file"
	ext := strings.ToLower(filepath.Ext(name))
	if strings.Contains(".jpg.jpeg.png.webp.gif.bmp", ext) {
		typ = "photo"
	} else if strings.Contains(".mp4.webm.mov.mkv", ext) {
		typ = "video"
	} else if strings.Contains(".mp3.wav.m4a.ogg.opus", ext) {
		typ = "voice"
	}
	return map[string]any{"url": uri, "type": typ, "name": name}
}
