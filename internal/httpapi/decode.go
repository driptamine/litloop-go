package httpapi

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func ValidURL(raw string) bool {
	u, e := url.ParseRequestURI(raw)
	return e == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
func CSVRows(q *Request) []map[string]string {
	q.R.Body = http.MaxBytesReader(q.W, q.R.Body, 32<<20)
	if q.R.ParseMultipartForm(4<<20) != nil {
		Fail(400, "Invalid multipart CSV upload")
	}
	if q.R.MultipartForm != nil {
		defer q.R.MultipartForm.RemoveAll()
	}
	f, _, e := q.R.FormFile("csv_file")
	if e != nil {
		f, _, e = q.R.FormFile("file")
	}
	if e != nil {
		Fail(400, "csv_file is required")
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.TrimLeadingSpace = true
	headers, e := reader.Read()
	if e != nil {
		Fail(400, "Missing CSV header")
	}
	headers[0] = strings.TrimPrefix(headers[0], "\ufeff")
	out := []map[string]string{}
	for n := 0; ; n++ {
		row, e := reader.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			Fail(400, fmt.Sprintf("Invalid CSV row %d", n+2))
		}
		if n >= 50000 {
			Fail(400, "CSV exceeds 50000 rows")
		}
		item := map[string]string{}
		for i, h := range headers {
			item[strings.TrimSpace(h)] = row[i]
		}
		out = append(out, item)
	}
	return out
}
func ParseDate(s string) *time.Time {
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"} {
		if t, e := time.Parse(layout, s); e == nil {
			return &t
		}
	}
	return nil
}
func ParseInt(s string) *int64 {
	if n, e := strconv.ParseInt(s, 10, 64); e == nil {
		return &n
	}
	return nil
}
func ParseFloat(s string) *float64 {
	if n, e := strconv.ParseFloat(s, 64); e == nil {
		return &n
	}
	return nil
}
