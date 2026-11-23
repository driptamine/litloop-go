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

func (handler *Handler) MultipartUpload(req *Request) storage.UploadedFile {
	req.R.Body = http.MaxBytesReader(req.W, req.R.Body, 64<<20)
	if err := req.R.ParseMultipartForm(8 << 20); err != nil {
		Fail(400, "Invalid multipart upload or file exceeds 64 MiB")
	}
	if req.R.MultipartForm != nil {
		defer req.R.MultipartForm.RemoveAll()
	}
	file, head, err := req.R.FormFile("file")
	if err != nil {
		file, head, err = req.R.FormFile("avatar")
	}
	if err != nil {
		file, head, err = req.R.FormFile("image")
	}
	for _, field := range []string{"icon", "banner", "background_image"} {
		if err == nil {
			break
		}
		file, head, err = req.R.FormFile(field)
	}
	if err != nil {
		Fail(400, "file is required")
	}
	defer file.Close()
	return handler.SaveUploadedFile(req, file, head)
}
func (handler *Handler) SaveUploadedFile(req *Request, file multipart.File, head *multipart.FileHeader) storage.UploadedFile {
	filename := filepath.Base(head.Filename)
	chatUpload := req.R.URL.Path == "/chats/upload" && handler.Config.UsesR2ForMedia()
	prefix := "attachments/"
	if chatUpload {
		prefix = storage.ChatAttachmentPrefix(req.User.ID)
	}
	name := Token() + strings.ToLower(filepath.Ext(filename))
	if chatUpload && (strings.ContainsAny(name, `/\`) || strings.Contains(name, "..")) {
		name = strings.SplitN(name, ".", 2)[0]
	}
	key := prefix + name
	typ := head.Header.Get("Content-Type")
	sample := make([]byte, 512)
	parsed, _ := io.ReadFull(file, sample)
	detected := http.DetectContentType(sample[:parsed])
	if !strings.HasPrefix(detected, "application/octet-stream") {
		typ = detected
	}
	_, err := file.Seek(0, io.SeekStart)
	Check(err)
	provider := "local"
	if handler.Config.UsesR2ForMedia() || strings.Contains(req.R.URL.Path, "/r2") {
		provider = "r2"
	} else if strings.Contains(req.R.URL.Path, "/gcs") {
		provider = "gcs"
	}
	stableImage := strings.Contains(req.R.URL.Path, "avatar") || strings.Contains(req.R.URL.Path, "cover") || strings.Contains(req.R.URL.Path, "/icon/") || strings.Contains(req.R.URL.Path, "/banner/") || strings.Contains(req.R.URL.Path, "background-image")
	if stableImage {
		if !strings.HasPrefix(detected, "image/") || detected == "image/svg+xml" {
			Fail(400, "A raster image is required")
		}
	}
	if provider == "local" {
		path := filepath.Join(handler.Config.MediaDir, key)
		Check(os.MkdirAll(filepath.Dir(path), 0755))
		out, fileErr := os.Create(path)
		Check(fileErr)
		_, fileErr = io.Copy(out, file)
		closeErr := out.Close()
		Check(fileErr)
		Check(closeErr)
		key = "local/" + key
	} else {
		Check(handler.StorageFor(provider).Put(req.R.Context(), key, typ, file))
	}
	uri := ""
	if provider == "local" {
		uri = handler.LocalURL(strings.TrimPrefix(key, "local/"))
	} else if chatUpload {
		uri = storage.PresignedURL(handler.Config, "r2", key)
		if uri == "" {
			Fail(502, "Could not sign the chat attachment URL")
		}
	} else if provider == "r2" {
		uri = handler.MediaURL(key, "", "")
	} else {
		uri = handler.MediaURL("", key, "")
	}
	if stableImage {
		asset := models.Asset{ID: Token(), UserID: req.User.ID, Key: key, Provider: provider, ContentType: detected}
		Check(handler.DB.Create(&asset).Error)
		uri = strings.TrimRight(handler.Config.PublicURL, "/") + "/assets/" + asset.ID
	}
	return storage.UploadedFile{Provider: provider, Key: key, URL: uri, Name: filename}
}
func (handler *Handler) Attachment(req *Request) any {
	if strings.HasPrefix(req.R.URL.Path, "/communities/") {
		handler.CommunityRole(handler.Community(req), req.User, false)
	}
	file := handler.MultipartUpload(req)
	uri, name := file.URL, file.Name
	if strings.Contains(req.R.URL.Path, "avatar") {
		Check(handler.DB.Model(req.User).Update("avatar", uri).Error)
		return map[string]any{"avatar": uri, "user": Get[models.User](handler.DB, req.User.ID)}
	}
	if strings.Contains(req.R.URL.Path, "cover") {
		Check(handler.DB.Model(req.User).Update("cover_image", uri).Error)
		return map[string]any{"cover_image": uri, "user": Get[models.User](handler.DB, req.User.ID)}
	}
	if strings.Contains(req.R.URL.Path, "background-image") {
		profile := models.Profile{UserID: req.User.ID, BackgroundImage: uri}
		Check(handler.DB.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"background_image"})}).Create(&profile).Error)
		return map[string]any{"background_image_url": uri}
	}
	if strings.HasPrefix(req.R.URL.Path, "/communities/") {
		community := handler.Community(req)
		handler.CommunityRole(community, req.User, false)
		col := "icon"
		if strings.Contains(req.R.URL.Path, "/banner/") {
			col = "banner"
		}
		Check(handler.DB.Model(&community).Update(col, uri).Error)
		return handler.CommunityJSON(Get[models.Community](handler.DB, community.ID), req.User)
	}
	if strings.Contains(req.R.URL.Path, "/voice/") {
		duration, _ := strconv.ParseFloat(req.R.FormValue("duration"), 64)
		if duration < 0 {
			Fail(400, "Invalid duration")
		}
		voice := models.VoiceMessage{UserID: Ptr(req.User.ID), Filename: name, Duration: duration, TranscriptionStatus: "not_requested"}
		voice.R2Key, voice.GCSKey, voice.S3Key = file.Keys()
		Check(handler.DB.Create(&voice).Error)
		req.Status = 201
		return map[string]any{"success": true, "id": voice.ID, "url": uri, "duration": duration}
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
