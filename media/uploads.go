package media

import (
	"bytes"
	"encoding/json"
	"io"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (handler *Handler) UploadInitiate(req *Request) any {
	body := req.Body()
	kind := strings.Split(strings.TrimPrefix(req.R.URL.Path, "/"), "/")[0]
	filename := filepath.Base(Str(body["filename"]))
	if filename == "" || filename == "." || len(filename) > 255 {
		Fail(400, "filename is required")
	}
	typ := Str(body["content_type"])
	if typ == "" {
		typ = mime.TypeByExtension(filepath.Ext(filename))
	}
	if typ == "" {
		typ = "application/octet-stream"
	}
	provider := handler.UploadProvider(req)
	if provider == "native" && handler.Config.UsesR2ForMedia() {
		Fail(410, "Use the R2 multipart upload endpoints")
	}
	upload := models.Upload{ID: Token(), UserID: req.User.ID, Provider: provider, Kind: kind, Key: kind + "/" + Token() + filepath.Ext(filename), ContentType: typ, Filename: filename, Status: "pending"}
	out := map[string]any{}
	if provider == "native" {
		client := handler.GoogleClient(req.R.Context())
		payload, _ := json.Marshal(map[string]any{"name": upload.Key, "contentType": typ})
		target := "https://storage.googleapis.com/upload/storage/v1/b/" + url.PathEscape(handler.Config.GCSBucket) + "/o?uploadType=resumable"
		httpReq, err := http.NewRequestWithContext(req.R.Context(), "POST", target, bytes.NewReader(payload))
		Check(err)
		httpReq.Header.Set("Content-Type", "application/json")
		res, err := client.Do(httpReq)
		Check(err)
		defer res.Body.Close()
		if res.StatusCode/100 != 2 {
			Fail(502, "GCS resumable upload failed")
		}
		upload.RemoteID = res.Header.Get("Location")
		out["resumable_url"] = upload.RemoteID
		out["upload_url"] = upload.RemoteID
	} else if provider != "local" {
		store := handler.StorageFor(provider)
		id, err := store.Initiate(req.R.Context(), upload.Key, typ)
		Check(err)
		upload.RemoteID = id
	}
	Check(handler.DB.Create(&upload).Error)
	out["upload_id"] = upload.ID
	out["key"] = upload.Key
	out["content_type"] = typ
	if provider == "local" {
		out["url"] = strings.TrimRight(handler.Config.PublicURL, "/") + "/uploads/" + upload.ID
		out["upload_url"] = out["url"]
	}
	return out
}
func (handler *Handler) PendingUpload(req *Request) models.Upload {
	body := req.Body()
	var upload models.Upload
	id := Str(body["upload_id"])
	if id == "" {
		Fail(400, "upload_id is required")
	}
	Check(handler.DB.First(&upload, "id = ? AND user_id = ?", id, req.User.ID).Error)
	if upload.Kind != strings.Split(strings.TrimPrefix(req.R.URL.Path, "/"), "/")[0] || (upload.Provider != handler.UploadProvider(req) && upload.Provider != handler.legacyUploadProvider(req)) {
		Fail(400, "Provider mismatch")
	}
	if upload.Status != "pending" {
		Fail(409, "Upload is already complete")
	}
	if key := Str(body["key"]); key != "" && key != upload.Key {
		Fail(400, "Upload key mismatch")
	}
	return upload
}
func (handler *Handler) UploadPart(req *Request) any {
	upload := handler.PendingUpload(req)
	if upload.Provider == "local" {
		return map[string]any{"url": strings.TrimRight(handler.Config.PublicURL, "/") + "/uploads/" + upload.ID}
	}
	partNumber := int32(1)
	if value, ok := req.Body()["part_number"]; ok {
		partNumber = int32(Num(value))
	}
	if partNumber < 1 || partNumber > 10000 {
		Fail(400, "Invalid part number")
	}
	signed, err := handler.StorageFor(upload.Provider).PartURL(req.R.Context(), upload.Key, upload.RemoteID, partNumber)
	Check(err)
	return map[string]any{"url": signed}
}
func (handler *Handler) UploadPut(req *Request) any {
	var upload models.Upload
	Check(handler.DB.First(&upload, "id = ? AND user_id = ? AND status = ?", req.Param("upload"), req.User.ID, "pending").Error)
	if upload.Provider != "local" {
		Fail(400, "Not a local upload")
	}
	path := filepath.Join(handler.Config.MediaDir, upload.Key)
	Check(os.MkdirAll(filepath.Dir(path), 0755))
	file, err := os.CreateTemp(filepath.Dir(path), ".upload-*")
	Check(err)
	defer os.Remove(file.Name())
	defer file.Close()
	req.R.Body = http.MaxBytesReader(req.W, req.R.Body, 512<<20)
	if _, err = io.Copy(file, req.R.Body); err != nil {
		Fail(400, "Upload failed or exceeds 512 MiB")
	}
	Check(file.Close())
	Check(os.Rename(file.Name(), path))
	return map[string]any{"status": "uploaded"}
}
func (handler *Handler) UploadComplete(req *Request) any {
	upload := handler.PendingUpload(req)
	body := req.Body()
	if upload.Provider == "local" {
		if st, err := os.Stat(filepath.Join(handler.Config.MediaDir, upload.Key)); err != nil || st.Size() == 0 {
			Fail(400, "Upload the file first")
		}
	} else if upload.Provider == "native" {
		httpReq, err := http.NewRequestWithContext(req.R.Context(), "GET", "https://storage.googleapis.com/storage/v1/b/"+url.PathEscape(handler.Config.GCSBucket)+"/o/"+url.PathEscape(upload.Key), nil)
		Check(err)
		res, err := handler.GoogleClient(req.R.Context()).Do(httpReq)
		Check(err)
		defer res.Body.Close()
		if res.StatusCode != 200 {
			Fail(400, "GCS object is not uploaded")
		}
	} else {
		parts, ok := body["parts"].([]any)
		if !ok || len(parts) == 0 || len(parts) > 10000 {
			Fail(400, "parts is required")
		}
		completed := []types.CompletedPart{}
		seen := map[int32]bool{}
		for _, part := range parts {
			entry, ok := part.(map[string]any)
			if !ok {
				Fail(400, "Invalid part")
			}
			partNumber := int32(Num(entry["PartNumber"]))
			etag := Str(entry["ETag"])
			if partNumber < 1 || partNumber > 10000 || etag == "" || seen[partNumber] {
				Fail(400, "Invalid part")
			}
			seen[partNumber] = true
			completed = append(completed, types.CompletedPart{PartNumber: aws.Int32(partNumber), ETag: aws.String(etag)})
		}
		sort.Slice(completed, func(left, right int) bool { return *completed[left].PartNumber < *completed[right].PartNumber })
		Check(handler.StorageFor(upload.Provider).Complete(req.R.Context(), upload.Key, upload.RemoteID, completed))
	}
	var resource any
	req.Tx(func(tx *gorm.DB) error {
		var lock models.Upload
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lock, "id = ?", upload.ID).Error)
		if lock.Status != "pending" {
			Fail(409, "Upload already complete")
		}
		key := upload.Key
		if upload.Provider == "local" {
			key = "local/" + key
		}
		r2Key, gcsKey, s3Key := "", "", ""
		switch upload.Provider {
		case "r2":
			r2Key = key
		case "gcs", "native":
			gcsKey = key
		default:
			s3Key = key
		}
		switch upload.Kind {
		case "photos":
			video := models.Photo{UserID: Ptr(req.User.ID), S3Key: s3Key, GCSKey: gcsKey, R2Key: r2Key, Filename: upload.Filename, Status: "draft", FriendlyToken: Token()[:12], Title: Str(body["title"])}
			Check(tx.Create(&video).Error)
			upload.ResourceID = video.ID
			resource = safeMedia(video)
		case "tracks":
			track := models.Track{UserID: Ptr(req.User.ID), S3Key: s3Key, GCSKey: gcsKey, R2Key: r2Key, Filename: upload.Filename, Status: "draft", Name: Str(body["name"]), TrackURI: Token()}
			Check(tx.Create(&track).Error)
			upload.ResourceID = track.ID
			resource = safeMedia(track)
		case "videos":
			photo := models.Video{UserID: Ptr(req.User.ID), S3Key: s3Key, GCSKey: gcsKey, R2Key: r2Key, Filename: upload.Filename, Status: "draft", Visibility: "private", Title: Str(body["title"])}
			Check(tx.Create(&photo).Error)
			upload.ResourceID = photo.ID
			resource = safeMedia(photo)
		default:
			Fail(400, "Unknown media type")
		}
		upload.Status = "completed"
		return tx.Save(&upload).Error
	})
	location := handler.MediaURL(func() string {
		if upload.Provider == "r2" {
			return upload.Key
		}
		return ""
	}(), func() string {
		if upload.Provider == "gcs" || upload.Provider == "native" {
			return upload.Key
		}
		return ""
	}(), func() string {
		if upload.Provider == "local" {
			return "local/" + upload.Key
		}
		if upload.Provider == "s3" {
			return upload.Key
		}
		return ""
	}())
	return map[string]any{"status": "completed", "id": upload.ResourceID, "location": location, "resource": resource}
}
func (handler *Handler) CancelUpload(req *Request) any {
	upload := handler.PendingUpload(req)
	if upload.Provider != "local" && upload.Provider != "native" {
		Check(handler.StorageFor(upload.Provider).Abort(req.R.Context(), upload.Key, upload.RemoteID))
	}
	Check(handler.DB.Model(&upload).Update("status", "cancelled").Error)
	return map[string]any{"status": "cancelled"}
}
