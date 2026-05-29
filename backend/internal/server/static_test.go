package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestStatic(t *testing.T) {
	origAssetsFS := assetsFS
	assetsFS = fstest.MapFS{
		"static/index.html": {Data: []byte("<!doctype html><title>port-mapper</title>")},
	}
	t.Cleanup(func() { assetsFS = origAssetsFS })

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantString string
	}{
		{name: "root serves index", path: "/", wantStatus: http.StatusOK, wantString: "port-mapper"},
		{name: "spa fallback serves index", path: "/dashboard", wantStatus: http.StatusOK, wantString: "port-mapper"},
	}

	srv := New("127.0.0.1:8080", nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			body, _ := io.ReadAll(rec.Body)
			if !strings.Contains(string(body), tt.wantString) {
				t.Fatalf("body missing %q: %s", tt.wantString, string(body))
			}
		})
	}
}
