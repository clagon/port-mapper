package server

import (
	"fmt"
	"log/slog"
	"testing"
)

func TestServerHandlerUsesEcho(t *testing.T) {
	s := New("127.0.0.1:8080", slog.Default(), nil)
	if got := fmt.Sprintf("%T", s.Handler()); got != "*echo.Echo" {
		t.Fatalf("handler type = %s, want *echo.Echo", got)
	}
}
