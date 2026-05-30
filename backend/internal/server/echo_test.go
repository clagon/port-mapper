package server

import (
	"fmt"
	"log/slog"
	"testing"
)

func TestServerHandlerUsesEcho(t *testing.T) {
	tests := []struct {
		name string
		addr string
	}{
		{
			name: "echo handler",
			addr: "127.0.0.1:8080",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.addr, slog.Default(), nil)
			if got := fmt.Sprintf("%T", s.Handler()); got != "*echo.Echo" {
				t.Fatalf("handler type = %s, want *echo.Echo", got)
			}
		})
	}
}
