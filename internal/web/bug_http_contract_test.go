package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMalformedCreateRequestUsesClientError(t *testing.T) {
	srv, cleanup := setupServer(t)
	defer cleanup()
	req := httptest.NewRequest(http.MethodPost, "/api/experiments", strings.NewReader("{"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("malformed request returned %d", rec.Code)
	}
}
