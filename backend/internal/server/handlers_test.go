package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/clagon/port-mapper/backend/internal/config"
	"github.com/clagon/port-mapper/backend/internal/upnp"
	"github.com/labstack/echo/v4"
)

type fakeDiscovery struct {
	result upnp.DiscoveryResult
	err    error
	calls  int
}

func (f *fakeDiscovery) Discover() (upnp.DiscoveryResult, error) {
	f.calls++
	return f.result, f.err
}

type deleteCall struct {
	protocol     string
	externalPort int
}

type fakeMapper struct {
	mu         sync.Mutex
	externalIP  string
	externalErr error
	addErr     error
	deleteErr  error
	addCalls   []upnp.PortMapping
	deleteCalls []deleteCall
}

func (f *fakeMapper) GetExternalIPAddress() (string, error) {
	if f.externalErr != nil {
		return "", f.externalErr
	}
	return f.externalIP, nil
}

func (f *fakeMapper) AddPortMapping(m upnp.PortMapping) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.addCalls = append(f.addCalls, m)
	return f.addErr
}

func (f *fakeMapper) DeletePortMapping(protocol string, externalPort int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteCalls = append(f.deleteCalls, deleteCall{protocol: protocol, externalPort: externalPort})
	return f.deleteErr
}

func newTestServer(t *testing.T, cfgPath string, cfg config.Config, discovery DiscoveryClient, mapper *fakeMapper) *Server {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return New("127.0.0.1:8080", logger,
		WithConfigPath(cfgPath),
		WithConfig(cfg),
		WithDiscoveryClient(discovery),
		WithPortMapperFactory(func(upnp.DiscoveryResult) PortMapper { return mapper }),
	)
}

func decodeJSON[T any](t *testing.T, body *bytes.Buffer, dst *T) {
	t.Helper()
	if err := json.Unmarshal(body.Bytes(), dst); err != nil {
		t.Fatalf("unmarshal body: %v\n%s", err, body.String())
	}
}

func TestHealthAndSettingsEndpoints(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	initial := config.DefaultConfig()
	mapper := &fakeMapper{externalIP: "203.0.113.42"}
	srv := newTestServer(t, cfgPath, initial, &fakeDiscovery{}, mapper)

	t.Run("health", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var got HealthResponse
		decodeJSON(t, rec.Body, &got)
		if !got.Ok {
			t.Fatalf("health response = %+v, want ok=true", got)
		}
	})

	t.Run("get settings", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var got config.Config
		decodeJSON(t, rec.Body, &got)
		if got.ListenAddr != initial.ListenAddr {
			t.Fatalf("listen_addr = %q, want %q", got.ListenAddr, initial.ListenAddr)
		}
		if got.AutoDiscover == nil || !*got.AutoDiscover {
			t.Fatalf("auto_discover = %v, want true", got.AutoDiscover)
		}
	})

	t.Run("post settings persists to disk", func(t *testing.T) {
		payload := config.Config{ListenAddr: "127.0.0.1:9090", AutoDiscover: config.BoolPtr(false)}
		buf, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/settings", bytes.NewReader(buf))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if _, err := os.Stat(cfgPath); err != nil {
			t.Fatalf("config not saved: %v", err)
		}
		loaded, err := config.Load(cfgPath)
		if err != nil {
			t.Fatalf("load config: %v", err)
		}
		if loaded.ListenAddr != payload.ListenAddr {
			t.Fatalf("listen_addr = %q, want %q", loaded.ListenAddr, payload.ListenAddr)
		}
		if loaded.AutoDiscover == nil || *loaded.AutoDiscover {
			t.Fatalf("auto_discover = %v, want false", loaded.AutoDiscover)
		}
	})
}

func TestDiscoveryAndMappingEndpoints(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	discovery := &fakeDiscovery{
		result: upnp.DiscoveryResult{
			ServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
			ControlURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn2",
		},
	}
	mapper := &fakeMapper{externalIP: "203.0.113.42"}
	srv := newTestServer(t, cfgPath, config.DefaultConfig(), discovery, mapper)

	mapping := upnp.PortMapping{
		Protocol:             "TCP",
		ExternalPort:         8080,
		InternalIP:           "192.168.1.20",
		InternalPort:         8080,
		Description:          "test mapping",
		LeaseDurationSeconds: 3600,
	}
	mappingBody := []byte(`{"protocol":"TCP","external_port":8080,"internal_ip":"192.168.1.20","internal_port":8080,"description":"test mapping","lease_duration_seconds":3600}`)

	t.Run("status starts empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var got StatusResponse
		decodeJSON(t, rec.Body, &got)
		if got.Discovered {
			t.Fatalf("discovered = true, want false")
		}
		if len(got.Ports) != 0 {
			t.Fatalf("ports = %d, want 0", len(got.Ports))
		}
	})

	t.Run("discover updates status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/discover", nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
		}
		if discovery.calls != 1 {
			t.Fatalf("discover calls = %d, want 1", discovery.calls)
		}
		var got StatusResponse
		decodeJSON(t, rec.Body, &got)
		if !got.Discovered {
			t.Fatal("discovered = false, want true")
		}
		if got.ControlURL != discovery.result.ControlURL {
			t.Fatalf("control_url = %q, want %q", got.ControlURL, discovery.result.ControlURL)
		}
		if got.ExternalIP != mapper.externalIP {
			t.Fatalf("external_ip = %q, want %q", got.ExternalIP, mapper.externalIP)
		}
	})

	t.Run("discover timeout is soft failure", func(t *testing.T) {
		timeoutDiscovery := &fakeDiscovery{err: upnp.ErrNoGateway}
		timeoutSrv := newTestServer(t, cfgPath, config.DefaultConfig(), timeoutDiscovery, &fakeMapper{})
		req := httptest.NewRequest(http.MethodPost, "/api/discover", nil)
		rec := httptest.NewRecorder()
		timeoutSrv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
		}
		var got StatusResponse
		decodeJSON(t, rec.Body, &got)
		if got.Discovered {
			t.Fatal("discovered = true, want false")
		}
	})

	t.Run("open port adds mapping", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/ports/open", bytes.NewReader(mappingBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
		}
		if len(mapper.addCalls) != 1 {
			t.Fatalf("add calls = %d, want 1", len(mapper.addCalls))
		}
		var got StatusResponse
		decodeJSON(t, rec.Body, &got)
		if len(got.Ports) != 1 {
			t.Fatalf("ports = %d, want 1", len(got.Ports))
		}
		if got.Ports[0].ExternalPort != mapping.ExternalPort {
			t.Fatalf("external port = %d, want %d", got.Ports[0].ExternalPort, mapping.ExternalPort)
		}
	})

	t.Run("close port removes mapping", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/ports/close", bytes.NewReader(mappingBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
		}
		if len(mapper.deleteCalls) != 1 {
			t.Fatalf("delete calls = %d, want 1", len(mapper.deleteCalls))
		}
		var got StatusResponse
		decodeJSON(t, rec.Body, &got)
		if len(got.Ports) != 0 {
			t.Fatalf("ports = %d, want 0", len(got.Ports))
		}
	})
}
