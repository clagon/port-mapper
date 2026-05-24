package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlers(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		path         string
		wantStatus   int
		wantContains []string
	}{
		{
			name:       "health",
			method:     http.MethodGet,
			path:       "/api/health",
			wantStatus: http.StatusOK,
			wantContains: []string{
				`"ok":true`,
			},
		},
		{
			name:       "status",
			method:     http.MethodGet,
			path:       "/api/status",
			wantStatus: http.StatusOK,
			wantContains: []string{
				`"discovered":false`,
				`"ports":[]`,
			},
		},
		{
			name:       "discover",
			method:     http.MethodPost,
			path:       "/api/discover",
			wantStatus: http.StatusAccepted,
			wantContains: []string{
				`"ok":true`,
			},
		},
		{
			name:       "ports open",
			method:     http.MethodPost,
			path:       "/api/ports/open",
			wantStatus: http.StatusAccepted,
			wantContains: []string{
				`"ok":true`,
			},
		},
		{
			name:       "ports close",
			method:     http.MethodPost,
			path:       "/api/ports/close",
			wantStatus: http.StatusAccepted,
			wantContains: []string{
				`"ok":true`,
			},
		},
		{
			name:       "get settings",
			method:     http.MethodGet,
			path:       "/api/settings",
			wantStatus: http.StatusOK,
			wantContains: []string{
				`"listen_addr":"127.0.0.1:8080"`,
				`"auto_discover":true`,
			},
		},
		{
			name:       "post settings",
			method:     http.MethodPost,
			path:       "/api/settings",
			wantStatus: http.StatusOK,
			wantContains: []string{
				`"ok":true`,
			},
		},
	}

	mux := NewMux()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(`{}`))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			body := rec.Body.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Fatalf("body missing %q: %s", want, body)
				}
			}
		})
	}
}
