package httpapi

import (
	"strconv"
	"time"
)

func Pagination(req *Request, defaultSize int, sizeKey string) (int, int) {
	page, size := 1, defaultSize
	if raw := req.R.URL.Query().Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 10000000 {
			Fail(400, "Invalid page")
		}
		page = value
	}
	if raw := req.R.URL.Query().Get(sizeKey); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			Fail(400, "Invalid page size")
		}
		size = value
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
func Date(timestamp *time.Time) any {
	if timestamp == nil {
		return nil
	}
	return timestamp.Format("2006-01-02")
}
