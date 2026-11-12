package httpapi

import (
	"strconv"
	"time"
)

func Pagination(q *Request, defaultSize int, sizeKey string) (int, int) {
	page, size := 1, defaultSize
	if raw := q.R.URL.Query().Get("page"); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 1 || v > 10000000 {
			Fail(400, "Invalid page")
		}
		page = v
	}
	if raw := q.R.URL.Query().Get(sizeKey); raw != "" {
		v, e := strconv.Atoi(raw)
		if e != nil || v < 1 {
			Fail(400, "Invalid page size")
		}
		size = v
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
func Date(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format("2006-01-02")
}
