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

func (s *Handler) MediaServe(q *Request) any {
	key := q.Param("*")
	if strings.Contains(key, "..") || filepath.IsAbs(key) {
		Fail(400, "Invalid path")
	}
	exp := q.R.URL.Query().Get("expires")
	n, e := strconv.ParseInt(exp, 10, 64)
	if e != nil || n < time.Now().Unix() || !SecureEqual(q.R.URL.Query().Get("signature"), s.SignMedia(key, exp)) {
		Fail(403, "Invalid or expired media URL")
	}
	if key, ok := strings.CutPrefix(key, "r2/hls/"); ok && strings.HasSuffix(key, ".m3u8") {
		return s.r2Manifest(q, "hls/"+key)
	}
	path := filepath.Join(s.Config.MediaDir, key)
	if strings.HasSuffix(key, ".m3u8") {
		raw, err := os.ReadFile(path)
		if err != nil {
			Fail(404, "Manifest not found")
		}
		body := manifestLines(string(raw), func(segment string) string {
			return s.LocalURL(filepath.ToSlash(filepath.Join(filepath.Dir(key), filepath.Base(segment))))
		})
		q.W.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		q.Status = 0
		_, _ = io.WriteString(q.W, body)
		return nil
	}
	q.Status = 0
	http.ServeFile(q.W, q.R, path)
	return nil
}

func (s *Handler) r2Manifest(q *Request, key string) any {
	store := s.StorageFor("r2")
	ctx := q.R.Context()
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
	q.W.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	q.W.Header().Set("Cache-Control", "private, no-store")
	q.Status = 0
	_, _ = io.WriteString(q.W, body)
	return nil
}

func manifestLines(raw string, link func(string) string) string {
	lines := strings.Split(raw, "\n")
	for i, l := range lines {
		if l != "" && !strings.HasPrefix(l, "#") {
			lines[i] = link(l)
		}
	}
	return strings.Join(lines, "\n")
}
