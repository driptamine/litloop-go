package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (s *Server) FetchJSON(q *Request, method, target string, body []byte, headers map[string]string) any {
	req, e := http.NewRequestWithContext(q.R.Context(), method, target, bytes.NewReader(body))
	Check(e)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, e := s.Client.Do(req)
	if e != nil {
		Fail(502, "External service is unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		Fail(502, fmt.Sprintf("External service returned HTTP %d", res.StatusCode))
	}
	var out any
	decoder := json.NewDecoder(io.LimitReader(res.Body, 8<<20))
	decoder.UseNumber()
	if decoder.Decode(&out) != nil {
		Fail(502, "Invalid external service response")
	}
	return out
}
