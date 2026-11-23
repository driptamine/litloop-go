package media

import (
	"io"
	. "litloop/backend/internal/httpapi"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func (handler *Handler) MediaServe(req *Request) any {
	key := req.Param("*")
	if strings.Contains(key, "..") || filepath.IsAbs(key) {
		Fail(400, "Invalid path")
	}
	exp := req.R.URL.Query().Get("expires")
	parsed, err := strconv.ParseInt(exp, 10, 64)
	if err != nil || parsed < time.Now().Unix() || !SecureEqual(req.R.URL.Query().Get("signature"), handler.SignMedia(key, exp)) {
		Fail(403, "Invalid or expired media URL")
	}
	if key, ok := strings.CutPrefix(key, "r2/hls/"); ok && strings.HasSuffix(key, ".m3u8") {
		return handler.r2Manifest(req, "hls/"+key)
	}
	path := filepath.Join(handler.Config.MediaDir, key)
	if strings.HasSuffix(key, ".m3u8") {
		raw, err := os.ReadFile(path)
		if err != nil {
			Fail(404, "Manifest not found")
		}
		body := manifestLines(string(raw), func(segment string) string {
			return handler.LocalURL(filepath.ToSlash(filepath.Join(filepath.Dir(key), filepath.Base(segment))))
		})
		req.W.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		req.Status = 0
		_, _ = io.WriteString(req.W, body)
		return nil
	}
	req.Status = 0
	http.ServeFile(req.W, req.R, path)
	return nil
}

func (handler *Handler) r2Manifest(req *Request, key string) any {
	store := handler.StorageFor("r2")
	ctx := req.R.Context()
	object, err := store.Client.GetObject(ctx, &s3.GetObjectInput{Bucket: &store.Bucket, Key: &key})
	Check(err)
	defer object.Body.Close()
	const maxPlaylist = 8 << 20
	raw, err := io.ReadAll(io.LimitReader(object.Body, maxPlaylist+1))
	Check(err)
	if len(raw) > maxPlaylist {
		Fail(502, "Video playlist exceeds the supported size")
	}
	body := manifestLines(string(raw), func(segment string) string {
		uri, err := store.GetURL(ctx, path.Join(path.Dir(key), path.Base(strings.TrimSpace(segment))))
		Check(err)
		return uri
	})
	req.W.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	req.W.Header().Set("Cache-Control", "private, no-store")
	req.Status = 0
	_, _ = io.WriteString(req.W, body)
	return nil
}

func manifestLines(raw string, link func(string) string) string {
	lines := strings.Split(raw, "\n")
	for index, line := range lines {
		if line != "" && !strings.HasPrefix(line, "#") {
			lines[index] = link(line)
		}
	}
	return strings.Join(lines, "\n")
}
