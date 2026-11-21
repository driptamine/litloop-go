package media

import (
	"context"
	"io"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"litloop/backend/internal/storage"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (s *Handler) NativeMedia(q *Request) any {
	key := strings.TrimPrefix(q.R.URL.Path, "/gcs-media/")
	exp := q.R.URL.Query().Get("expires")
	n, e := strconv.ParseInt(exp, 10, 64)
	if e != nil || n < time.Now().Unix() || !SecureEqual(q.R.URL.Query().Get("signature"), s.SignMedia("gcs/"+key, exp)) {
		Fail(403, "Invalid or expired media URL")
	}
	req, e := http.NewRequestWithContext(q.R.Context(), "GET", "https://storage.googleapis.com/storage/v1/b/"+url.PathEscape(s.Config.GCSBucket)+"/o/"+url.PathEscape(key)+"?alt=media", nil)
	Check(e)
	if r := q.R.Header.Get("Range"); r != "" {
		req.Header.Set("Range", r)
	}
	client, e := storage.NativeReadClient(q.R.Context(), s.Config)
	if e != nil {
		Fail(503, "Native GCS is not configured")
	}
	res, e := client.Do(req)
	Check(e)
	defer res.Body.Close()
	if res.StatusCode != 200 && res.StatusCode != 206 {
		Fail(502, "GCS media is unavailable")
	}
	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if value := res.Header.Get(h); value != "" {
			q.W.Header().Set(h, value)
		}
	}
	q.Status = 0
	q.W.WriteHeader(res.StatusCode)
	_, _ = io.Copy(q.W, res.Body)
	return nil
}
func (s *Handler) Asset(q *Request) any {
	var a models.Asset
	Check(s.DB.First(&a, "id = ?", q.Param("asset")).Error)
	q.W.Header().Set("X-Content-Type-Options", "nosniff")
	if a.Provider == "local" {
		q.W.Header().Set("Content-Type", a.ContentType)
		q.Status = 0
		http.ServeFile(q.W, q.R, filepath.Join(s.Config.MediaDir, strings.TrimPrefix(a.Key, "local/")))
		return nil
	}
	uri := ""
	if a.Provider == "gcs" && s.Config.GCSAccessKey == "" {
		uri = s.NativeURL(a.Key)
	} else {
		store := s.StorageFor(a.Provider)
		ctx, cancel := context.WithTimeout(q.R.Context(), 5*time.Second)
		defer cancel()
		var e error
		uri, e = store.GetURL(ctx, a.Key)
		Check(e)
	}
	q.Status = 0
	q.W.Header().Set("Cache-Control", "private, no-store")
	http.Redirect(q.W, q.R, uri, http.StatusTemporaryRedirect)
	return nil
}
