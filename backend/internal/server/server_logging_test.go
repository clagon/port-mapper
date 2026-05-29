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
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	s := New("127.0.0.1:8080", logger, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	s.Handler().ServeHTTP(rec, req)

	logLine := buf.String()
	for _, want := range []string{"msg=request", "method=GET", "path=/api/health", "status=200"} {
		if !strings.Contains(logLine, want) {
			t.Fatalf("log line missing %q: %s", want, logLine)
		}
	}
}
