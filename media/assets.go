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

func (handler *Handler) NativeMedia(req *Request) any {
	key := strings.TrimPrefix(req.R.URL.Path, "/gcs-media/")
	exp := req.R.URL.Query().Get("expires")
	parsed, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || parsed < time.Now().Unix() || !SecureEqual(req.R.URL.Query().Get("signature"), handler.SignMedia("gcs/"+key, exp)) {
		Fail(403, "Invalid or expired media URL")
	}
	httpReq, err := http.NewRequestWithContext(req.R.Context(), "GET", "https://storage.googleapis.com/storage/v1/b/"+url.PathEscape(handler.Config.GCSBucket)+"/o/"+url.PathEscape(key)+"?alt=media", nil)
	Check(err)
	if header := req.R.Header.Get("Range"); header != "" {
		httpReq.Header.Set("Range", header)
	}
	client, err := storage.NativeReadClient(req.R.Context(), handler.Config)
	if err != nil {
		Fail(503, "Native GCS is not configured")
	}
	res, err := client.Do(httpReq)
	Check(err)
	defer res.Body.Close()
	if res.StatusCode != 200 && res.StatusCode != 206 {
		Fail(502, "GCS media is unavailable")
	}
	for _, header := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges"} {
		if value := res.Header.Get(header); value != "" {
			req.W.Header().Set(header, value)
		}
	}
	req.Status = 0
	req.W.WriteHeader(res.StatusCode)
	_, _ = io.Copy(req.W, res.Body)
	return nil
}
func (handler *Handler) Asset(req *Request) any {
	var asset models.Asset
	Check(handler.DB.First(&asset, "id = ?", req.Param("asset")).Error)
	req.W.Header().Set("X-Content-Type-Options", "nosniff")
	if asset.Provider == "local" {
		req.W.Header().Set("Content-Type", asset.ContentType)
		req.Status = 0
		http.ServeFile(req.W, req.R, filepath.Join(handler.Config.MediaDir, strings.TrimPrefix(asset.Key, "local/")))
		return nil
	}
	uri := ""
	if asset.Provider == "gcs" && handler.Config.GCSAccessKey == "" {
		uri = handler.NativeURL(asset.Key)
	} else {
		store := handler.StorageFor(asset.Provider)
		ctx, cancel := context.WithTimeout(req.R.Context(), 5*time.Second)
		defer cancel()
		var err error
		uri, err = store.GetURL(ctx, asset.Key)
		Check(err)
	}
	req.Status = 0
	req.W.Header().Set("Cache-Control", "private, no-store")
	http.Redirect(req.W, req.R, uri, http.StatusTemporaryRedirect)
	return nil
}
