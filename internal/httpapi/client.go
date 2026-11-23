package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (server *Server) FetchJSON(req *Request, method, target string, body []byte, headers map[string]string) any {
	httpReq, err := http.NewRequestWithContext(req.R.Context(), method, target, bytes.NewReader(body))
	Check(err)
	for key, value := range headers {
		httpReq.Header.Set(key, value)
	}
	res, err := server.Client.Do(httpReq)
	if err != nil {
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
