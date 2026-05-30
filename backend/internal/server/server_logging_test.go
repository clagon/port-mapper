package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServerLogsRequests(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "logs request fields",
			path: "/api/health",
			want: []string{"msg=request", "method=GET", "path=/api/health", "status=200"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

			s := New("127.0.0.1:8080", logger, nil)
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			s.Handler().ServeHTTP(rec, req)

			logLine := buf.String()
			for _, want := range tt.want {
				if !strings.Contains(logLine, want) {
					t.Fatalf("log line missing %q: %s", want, logLine)
				}
			}
		})
	}
}
