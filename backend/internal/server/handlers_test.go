package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clagon/port-mapper/backend/internal/config"
	"github.com/clagon/port-mapper/backend/internal/service"
	"github.com/clagon/port-mapper/backend/internal/upnp"
	"github.com/labstack/echo/v4"
)

type fakeAPIService struct {
	statusValue   service.Status
	settingsValue config.Config
	discoverErr   error
	openErr       error
	closeErr      error
	settingsErr   error
	openRequest   upnp.PortMapping
	closeRequest  upnp.PortMapping
	settingsReq   config.Config
}

func (f *fakeAPIService) Status() service.Status { return f.statusValue }
func (f *fakeAPIService) Discover() (service.Status, error) {
	return f.statusValue, f.discoverErr
}
func (f *fakeAPIService) OpenPort(m upnp.PortMapping) (service.Status, error) {
	f.openRequest = m
	return f.statusValue, f.openErr
}
func (f *fakeAPIService) ClosePort(m upnp.PortMapping) (service.Status, error) {
	f.closeRequest = m
	return f.statusValue, f.closeErr
}
func (f *fakeAPIService) Settings() config.Config { return f.settingsValue }
func (f *fakeAPIService) UpdateSettings(c config.Config) (config.Config, error) {
	f.settingsReq = c
	return c, f.settingsErr
}

func decodeJSON[T any](t *testing.T, body *bytes.Buffer, dst *T) {
	t.Helper()
	if err := json.Unmarshal(body.Bytes(), dst); err != nil {
		t.Fatalf("unmarshal body: %v\n%s", err, body.String())
	}
}

func TestHealthAndReadEndpoints(t *testing.T) {
	svc := &fakeAPIService{
		statusValue:   service.Status{Discovered: true, ControlURL: "http://192.168.1.1/control"},
		settingsValue: config.Config{ListenAddr: "127.0.0.1:8080", AutoDiscover: config.BoolPtr(true)},
	}
	srv := New("127.0.0.1:8080", nil, svc)

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

	t.Run("status", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var got StatusResponse
		decodeJSON(t, rec.Body, &got)
		if !got.Discovered || got.ControlURL != svc.statusValue.ControlURL {
			t.Fatalf("status response = %+v, want %+v", got, svc.statusValue)
		}
	})

	t.Run("settings", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/settings", nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var got config.Config
		decodeJSON(t, rec.Body, &got)
		if got.ListenAddr != svc.settingsValue.ListenAddr {
			t.Fatalf("listen_addr = %q, want %q", got.ListenAddr, svc.settingsValue.ListenAddr)
		}
	})
}

func TestMutatingEndpointsBindRequests(t *testing.T) {
	svc := &fakeAPIService{statusValue: service.Status{Discovered: true}}
	srv := New("127.0.0.1:8080", nil, svc)

	t.Run("discover", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/discover", nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
		}
	})

	t.Run("open port", func(t *testing.T) {
		body := []byte(`{"protocol":"TCP","external_port":8080,"internal_ip":"192.168.1.20","internal_port":8080}`)
		req := httptest.NewRequest(http.MethodPost, "/api/ports/open", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
		}
		if svc.openRequest.ExternalPort != 8080 || svc.openRequest.Protocol != "TCP" {
			t.Fatalf("open request = %+v", svc.openRequest)
		}
	})

	t.Run("close port", func(t *testing.T) {
		body := []byte(`{"protocol":"UDP","external_port":5353}`)
		req := httptest.NewRequest(http.MethodPost, "/api/ports/close", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusAccepted {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
		}
		if svc.closeRequest.ExternalPort != 5353 || svc.closeRequest.Protocol != "UDP" {
			t.Fatalf("close request = %+v", svc.closeRequest)
		}
	})

	t.Run("update settings", func(t *testing.T) {
		body := []byte(`{"listen_addr":"127.0.0.1:9090","auto_discover":false}`)
		req := httptest.NewRequest(http.MethodPost, "/api/settings", bytes.NewReader(body))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if svc.settingsReq.ListenAddr != "127.0.0.1:9090" {
			t.Fatalf("settings request = %+v", svc.settingsReq)
		}
	})
}

func TestEndpointErrorConversion(t *testing.T) {
	svc := &fakeAPIService{
		discoverErr: errors.New("discover failed"),
		openErr:     errors.New("open failed"),
		closeErr:    errors.New("close failed"),
		settingsErr: errors.New("settings failed"),
	}
	srv := New("127.0.0.1:8080", nil, svc)

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{name: "discover error", method: http.MethodPost, path: "/api/discover", wantStatus: http.StatusBadGateway},
		{name: "open error", method: http.MethodPost, path: "/api/ports/open", body: `{"protocol":"TCP"}`, wantStatus: http.StatusBadRequest},
		{name: "close error", method: http.MethodPost, path: "/api/ports/close", body: `{"protocol":"TCP"}`, wantStatus: http.StatusBadRequest},
		{name: "settings error", method: http.MethodPost, path: "/api/settings", body: `{"listen_addr":"0.0.0.0:8080"}`, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader([]byte(tt.body)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
