package httpapi

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"litloop/backend/internal/config"
)

func TestSearchPreflightAllowsFrontendHeaders(tester *testing.T) {
	const origin = "https://litloop.netlify.app"
	const path = "/queries/google/firefox/search"
	server := NewServer(nil, config.Config{Origins: []string{origin}})
	tester.Cleanup(server.Hub.Close)
	server.Route(http.MethodGet, path, false, func(*Request) any {
		tester.Fatal("preflight reached the search handler")
		return nil
	})

	request := httptest.NewRequest(http.MethodOptions, path+"?cp=&q=goo", nil)
	frontendHeaders := []string{"0", "accept-version", "authorization", "user-agent"}
	request.Header.Set("Origin", origin)
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	request.Header.Set("Access-Control-Request-Headers", strings.Join(frontendHeaders, ","))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		tester.Fatalf("preflight status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != origin {
		tester.Fatalf("allowed origin = %q, want %q", got, origin)
	}
	allowed := strings.Split(strings.ToLower(response.Header().Get("Access-Control-Allow-Headers")), ", ")
	for _, name := range frontendHeaders {
		if !slices.Contains(allowed, name) {
			tester.Errorf("preflight did not allow %q", name)
		}
	}
}
