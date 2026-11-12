package media

import (
	"io"
	. "litloop/backend/internal/httpapi"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
func manifestLines(raw string, link func(string) string) string {
	lines := strings.Split(raw, "\n")
	for i, l := range lines {
		if l != "" && !strings.HasPrefix(l, "#") {
			lines[i] = link(l)
		}
	}
	return strings.Join(lines, "\n")
}
