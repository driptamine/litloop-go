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
	parsedURL, err := url.ParseRequestURI(raw)
	return err == nil && (parsedURL.Scheme == "http" || parsedURL.Scheme == "https") && parsedURL.Host != ""
}
func CSVRows(req *Request) []map[string]string {
	req.R.Body = http.MaxBytesReader(req.W, req.R.Body, 32<<20)
	if req.R.ParseMultipartForm(4<<20) != nil {
		Fail(400, "Invalid multipart CSV upload")
	}
	if req.R.MultipartForm != nil {
		defer req.R.MultipartForm.RemoveAll()
	}
	file, _, err := req.R.FormFile("csv_file")
	if err != nil {
		file, _, err = req.R.FormFile("file")
	}
	if err != nil {
		Fail(400, "csv_file is required")
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true
	headers, err := reader.Read()
	if err != nil {
		Fail(400, "Missing CSV header")
	}
	headers[0] = strings.TrimPrefix(headers[0], "\ufeff")
	out := []map[string]string{}
	for rowIndex := 0; ; rowIndex++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			Fail(400, fmt.Sprintf("Invalid CSV row %d", rowIndex+2))
		}
		if rowIndex >= 50000 {
			Fail(400, "CSV exceeds 50000 rows")
		}
		item := map[string]string{}
		for index, header := range headers {
			item[strings.TrimSpace(header)] = row[index]
		}
		out = append(out, item)
	}
	return out
}
func ParseDate(text string) *time.Time {
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, text); err == nil {
			return &parsed
		}
	}
	return nil
}
func ParseInt(text string) *int64 {
	if parsed, err := strconv.ParseInt(text, 10, 64); err == nil {
		return &parsed
	}
	return nil
}
func ParseFloat(text string) *float64 {
	if parsed, err := strconv.ParseFloat(text, 64); err == nil {
		return &parsed
	}
	return nil
}
