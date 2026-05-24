package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clagon/port-mapper/backend/internal/config"
)

type browserOpenerFunc func(string) error

func (f browserOpenerFunc) Open(url string) error { return f(url) }

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		opts     AppOptions
		wantAddr string
	}{
		{
			name:     "default address",
			opts:     AppOptions{},
			wantAddr: "127.0.0.1:8080",
		},
		{
			name:     "custom address",
			opts:     AppOptions{ListenAddr: "127.0.0.1:9090"},
			wantAddr: "127.0.0.1:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(tt.opts)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			if got := a.Addr(); got != tt.wantAddr {
				t.Fatalf("Addr() = %q, want %q", got, tt.wantAddr)
			}
		})
	}
}

func TestNewRejectsNonLocalListenAddr(t *testing.T) {
	tests := []struct {
		name string
		addr string
	}{
		{name: "non local bind", addr: "0.0.0.0:8080"},
		{name: "localhost bind", addr: "localhost:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := New(AppOptions{ListenAddr: tt.addr})
			if tt.addr == "localhost:8080" {
				if err != nil {
					t.Fatalf("New() error = %v", err)
				}
				if got := a.Addr(); got != tt.addr {
					t.Fatalf("Addr() = %q, want %q", got, tt.addr)
				}
				return
			}
			if err == nil {
				t.Fatal("New() error = nil, want error")
			}
		})
	}
}

func TestConfigPathDefaultUsesBinaryDir(t *testing.T) {
	a, err := New(AppOptions{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if got, want := a.ConfigPath(), config.DefaultPath(); got != want {
		t.Fatalf("ConfigPath() = %q, want %q", got, want)
	}
}

func TestStartOpensBrowserOnlyWhenEnabled(t *testing.T) {
	tests := []struct {
		name         string
		openBrowser  bool
		wantCalls    int
		wantURL      string
	}{
		{name: "disabled", openBrowser: false, wantCalls: 0},
		{name: "enabled", openBrowser: true, wantCalls: 1, wantURL: "http://127.0.0.1:8080/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			a, err := New(AppOptions{
				OpenBrowser:  tt.openBrowser,
				BrowserOpener: browserOpenerFunc(func(url string) error {
					got = append(got, url)
					return nil
				}),
			})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}
			if err := a.Start(); err != nil {
				t.Fatalf("Start() error = %v", err)
			}
			if len(got) != tt.wantCalls {
				t.Fatalf("browser calls = %d, want %d", len(got), tt.wantCalls)
			}
			if tt.wantURL != "" && got[0] != tt.wantURL {
				t.Fatalf("browser url = %q, want %q", got[0], tt.wantURL)
			}
		})
	}
}

func TestHealthHandler(t *testing.T) {
	a, err := New(AppOptions{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	a.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got map[string]bool
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if !got["ok"] {
		t.Fatalf("body = %s, want {\"ok\":true}", rec.Body.String())
	}
}
