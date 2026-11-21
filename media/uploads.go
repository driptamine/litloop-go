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

func (s *Handler) UploadInitiate(q *Request) any {
	b := q.Body()
	kind := strings.Split(strings.TrimPrefix(q.R.URL.Path, "/"), "/")[0]
	filename := filepath.Base(Str(b["filename"]))
	if filename == "" || filename == "." || len(filename) > 255 {
		Fail(400, "filename is required")
	}
	typ := Str(b["content_type"])
	if typ == "" {
		typ = mime.TypeByExtension(filepath.Ext(filename))
	}
	if typ == "" {
		typ = "application/octet-stream"
	}
	provider := s.UploadProvider(q)
	if provider == "native" && s.Config.UsesR2ForMedia() {
		Fail(410, "Use the R2 multipart upload endpoints")
	}
	u := models.Upload{ID: Token(), UserID: q.User.ID, Provider: provider, Kind: kind, Key: kind + "/" + Token() + filepath.Ext(filename), ContentType: typ, Filename: filename, Status: "pending"}
	out := map[string]any{}
	if provider == "native" {
		client := s.GoogleClient(q.R.Context())
		payload, _ := json.Marshal(map[string]any{"name": u.Key, "contentType": typ})
		target := "https://storage.googleapis.com/upload/storage/v1/b/" + url.PathEscape(s.Config.GCSBucket) + "/o?uploadType=resumable"
		req, e := http.NewRequestWithContext(q.R.Context(), "POST", target, bytes.NewReader(payload))
		Check(e)
		req.Header.Set("Content-Type", "application/json")
		res, e := client.Do(req)
		Check(e)
		defer res.Body.Close()
		if res.StatusCode/100 != 2 {
			Fail(502, "GCS resumable upload failed")
		}
		u.RemoteID = res.Header.Get("Location")
		out["resumable_url"] = u.RemoteID
		out["upload_url"] = u.RemoteID
	} else if provider != "local" {
		store := s.StorageFor(provider)
		id, e := store.Initiate(q.R.Context(), u.Key, typ)
		Check(e)
		u.RemoteID = id
	}
	Check(s.DB.Create(&u).Error)
	out["upload_id"] = u.ID
	out["key"] = u.Key
	out["content_type"] = typ
	if provider == "local" {
		out["url"] = strings.TrimRight(s.Config.PublicURL, "/") + "/uploads/" + u.ID
		out["upload_url"] = out["url"]
	}
	return out
}
func (s *Handler) PendingUpload(q *Request) models.Upload {
	b := q.Body()
	var u models.Upload
	id := Str(b["upload_id"])
	if id == "" {
		Fail(400, "upload_id is required")
	}
	Check(s.DB.First(&u, "id = ? AND user_id = ?", id, q.User.ID).Error)
	if u.Kind != strings.Split(strings.TrimPrefix(q.R.URL.Path, "/"), "/")[0] || (u.Provider != s.UploadProvider(q) && u.Provider != s.legacyUploadProvider(q)) {
		Fail(400, "Provider mismatch")
	}
	if u.Status != "pending" {
		Fail(409, "Upload is already complete")
	}
	if key := Str(b["key"]); key != "" && key != u.Key {
		Fail(400, "Upload key mismatch")
	}
	return u
}
func (s *Handler) UploadPart(q *Request) any {
	u := s.PendingUpload(q)
	if u.Provider == "local" {
		return map[string]any{"url": strings.TrimRight(s.Config.PublicURL, "/") + "/uploads/" + u.ID}
	}
	n := int32(1)
	if v, ok := q.Body()["part_number"]; ok {
		n = int32(Num(v))
	}
	if n < 1 || n > 10000 {
		Fail(400, "Invalid part number")
	}
	signed, e := s.StorageFor(u.Provider).PartURL(q.R.Context(), u.Key, u.RemoteID, n)
	Check(e)
	return map[string]any{"url": signed}
}
func (s *Handler) UploadPut(q *Request) any {
	var u models.Upload
	Check(s.DB.First(&u, "id = ? AND user_id = ? AND status = ?", q.Param("upload"), q.User.ID, "pending").Error)
	if u.Provider != "local" {
		Fail(400, "Not a local upload")
	}
	path := filepath.Join(s.Config.MediaDir, u.Key)
	Check(os.MkdirAll(filepath.Dir(path), 0755))
	f, e := os.CreateTemp(filepath.Dir(path), ".upload-*")
	Check(e)
	defer os.Remove(f.Name())
	defer f.Close()
	q.R.Body = http.MaxBytesReader(q.W, q.R.Body, 512<<20)
	if _, e = io.Copy(f, q.R.Body); e != nil {
		Fail(400, "Upload failed or exceeds 512 MiB")
	}
	Check(f.Close())
	Check(os.Rename(f.Name(), path))
	return map[string]any{"status": "uploaded"}
}
func (s *Handler) UploadComplete(q *Request) any {
	u := s.PendingUpload(q)
	b := q.Body()
	if u.Provider == "local" {
		if st, e := os.Stat(filepath.Join(s.Config.MediaDir, u.Key)); e != nil || st.Size() == 0 {
			Fail(400, "Upload the file first")
		}
	} else if u.Provider == "native" {
		req, e := http.NewRequestWithContext(q.R.Context(), "GET", "https://storage.googleapis.com/storage/v1/b/"+url.PathEscape(s.Config.GCSBucket)+"/o/"+url.PathEscape(u.Key), nil)
		Check(e)
		res, e := s.GoogleClient(q.R.Context()).Do(req)
		Check(e)
		defer res.Body.Close()
		if res.StatusCode != 200 {
			Fail(400, "GCS object is not uploaded")
		}
	} else {
		parts, ok := b["parts"].([]any)
		if !ok || len(parts) == 0 || len(parts) > 10000 {
			Fail(400, "parts is required")
		}
		completed := []types.CompletedPart{}
		seen := map[int32]bool{}
		for _, v := range parts {
			p, ok := v.(map[string]any)
			if !ok {
				Fail(400, "Invalid part")
			}
			n := int32(Num(p["PartNumber"]))
			etag := Str(p["ETag"])
			if n < 1 || n > 10000 || etag == "" || seen[n] {
				Fail(400, "Invalid part")
			}
			seen[n] = true
			completed = append(completed, types.CompletedPart{PartNumber: aws.Int32(n), ETag: aws.String(etag)})
		}
		sort.Slice(completed, func(i, j int) bool { return *completed[i].PartNumber < *completed[j].PartNumber })
		Check(s.StorageFor(u.Provider).Complete(q.R.Context(), u.Key, u.RemoteID, completed))
	}
	var resource any
	q.Tx(func(tx *gorm.DB) error {
		var lock models.Upload
		Check(tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lock, "id = ?", u.ID).Error)
		if lock.Status != "pending" {
			Fail(409, "Upload already complete")
		}
		key := u.Key
		if u.Provider == "local" {
			key = "local/" + key
		}
		r, g, a := "", "", ""
		switch u.Provider {
		case "r2":
			r = key
		case "gcs", "native":
			g = key
		default:
			a = key
		}
		switch u.Kind {
		case "photos":
			v := models.Photo{UserID: Ptr(q.User.ID), S3Key: a, GCSKey: g, R2Key: r, Filename: u.Filename, Status: "draft", FriendlyToken: Token()[:12], Title: Str(b["title"])}
			Check(tx.Create(&v).Error)
			u.ResourceID = v.ID
			resource = safeMedia(v)
		case "tracks":
			v := models.Track{UserID: Ptr(q.User.ID), S3Key: a, GCSKey: g, R2Key: r, Filename: u.Filename, Status: "draft", Name: Str(b["name"]), TrackURI: Token()}
			Check(tx.Create(&v).Error)
			u.ResourceID = v.ID
			resource = safeMedia(v)
		case "videos":
			v := models.Video{UserID: Ptr(q.User.ID), S3Key: a, GCSKey: g, R2Key: r, Filename: u.Filename, Status: "draft", Visibility: "private", Title: Str(b["title"])}
			Check(tx.Create(&v).Error)
			u.ResourceID = v.ID
			resource = safeMedia(v)
		default:
			Fail(400, "Unknown media type")
		}
		u.Status = "completed"
		return tx.Save(&u).Error
	})
	location := s.MediaURL(func() string {
		if u.Provider == "r2" {
			return u.Key
		}
		return ""
	}(), func() string {
		if u.Provider == "gcs" || u.Provider == "native" {
			return u.Key
		}
		return ""
	}(), func() string {
		if u.Provider == "local" {
			return "local/" + u.Key
		}
		if u.Provider == "s3" {
			return u.Key
		}
		return ""
	}())
	return map[string]any{"status": "completed", "id": u.ResourceID, "location": location, "resource": resource}
}
func (s *Handler) CancelUpload(q *Request) any {
	u := s.PendingUpload(q)
	if u.Provider != "local" && u.Provider != "native" {
		Check(s.StorageFor(u.Provider).Abort(q.R.Context(), u.Key, u.RemoteID))
	}
	Check(s.DB.Model(&u).Update("status", "cancelled").Error)
	return map[string]any{"status": "cancelled"}
}
