package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/clagon/port-mapper/backend/internal/config"
	"github.com/clagon/port-mapper/backend/internal/domain"
	"github.com/clagon/port-mapper/backend/internal/service"
	"github.com/labstack/echo/v4"
)

type fakeAPIService struct {
	statusValue   service.Status
	settingsValue config.Config
	discoverErr   error
	openErr       error
	closeErr      error
	settingsErr   error
	openRequest   domain.PortMapping
	closeRequest  domain.PortMapping
	settingsReq   config.Config
}

func (f *fakeAPIService) Status() service.Status { return f.statusValue }
func (f *fakeAPIService) Discover() (service.Status, error) {
	return f.statusValue, f.discoverErr
}
func (f *fakeAPIService) OpenPort(m domain.PortMapping) (service.Status, error) {
	f.openRequest = m
	return f.statusValue, f.openErr
}
func (f *fakeAPIService) ClosePort(m domain.PortMapping) (service.Status, error) {
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
	tests := []struct {
		name           string
		path           string
		wantStatus     int    // httptest.ResponseRecorder.Code
		wantHealthOK   bool   // HealthResponse.Ok
		wantDiscovered bool   // StatusResponse.Discovered
		wantControlURL string // StatusResponse.ControlURL
		wantListenAddr string // config.Config.ListenAddr
	}{
		{
			name:         "ヘルスチェックを返す",
			path:         "/api/health",
			wantStatus:   http.StatusOK,
			wantHealthOK: true,
		},
		{
			name:           "ステータスを返す",
			path:           "/api/status",
			wantStatus:     http.StatusOK,
			wantDiscovered: true,
			wantControlURL: svc.statusValue.ControlURL,
		},
		{
			name:           "設定を返す",
			path:           "/api/settings",
			wantStatus:     http.StatusOK,
			wantListenAddr: svc.settingsValue.ListenAddr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			switch tt.path {
			case "/api/health":
				var got HealthResponse
				decodeJSON(t, rec.Body, &got)
				if got.Ok != tt.wantHealthOK {
					t.Fatalf("HealthResponse.Ok = %v, want %v", got.Ok, tt.wantHealthOK)
				}
			case "/api/status":
				var got StatusResponse
				decodeJSON(t, rec.Body, &got)
				if got.Discovered != tt.wantDiscovered {
					t.Fatalf("StatusResponse.Discovered = %v, want %v", got.Discovered, tt.wantDiscovered)
				}
				if got.ControlURL != tt.wantControlURL {
					t.Fatalf("StatusResponse.ControlURL = %q, want %q", got.ControlURL, tt.wantControlURL)
				}
			case "/api/settings":
				var got config.Config
				decodeJSON(t, rec.Body, &got)
				if got.ListenAddr != tt.wantListenAddr {
					t.Fatalf("Config.ListenAddr = %q, want %q", got.ListenAddr, tt.wantListenAddr)
				}
			}
		})
	}
}

func TestMutatingEndpointsBindRequests(t *testing.T) {
	tests := []struct {
		name                     string
		path                     string
		body                     []byte
		wantStatus               int                // httptest.ResponseRecorder.Code
		wantOpenRequest          domain.PortMapping // fakeAPIService.openRequest
		wantCloseRequest         domain.PortMapping // fakeAPIService.closeRequest
		wantSettingsListenAddr   string             // fakeAPIService.settingsReq.ListenAddr
		wantSettingsAutoDiscover *bool              // fakeAPIService.settingsReq.AutoDiscover
	}{
		{
			name:       "探索を受け付ける",
			path:       "/api/discover",
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "ポート開放リクエストを束縛する",
			path:       "/api/ports/open",
			body:       []byte(`{"protocol":"TCP","external_port":8080,"internal_ip":"192.168.1.20","internal_port":8080}`),
			wantStatus: http.StatusAccepted,
			wantOpenRequest: domain.PortMapping{
				Protocol:     "TCP",
				ExternalPort: 8080,
				InternalIP:   "192.168.1.20",
				InternalPort: 8080,
			},
		},
		{
			name:       "ポート閉鎖リクエストを束縛する",
			path:       "/api/ports/close",
			body:       []byte(`{"protocol":"UDP","external_port":5353}`),
			wantStatus: http.StatusAccepted,
			wantCloseRequest: domain.PortMapping{
				Protocol:     "UDP",
				ExternalPort: 5353,
			},
		},
		{
			name:                     "設定更新リクエストを束縛する",
			path:                     "/api/settings",
			body:                     []byte(`{"listen_addr":"127.0.0.1:9090","auto_discover":false}`),
			wantStatus:               http.StatusOK,
			wantSettingsListenAddr:   "127.0.0.1:9090",
			wantSettingsAutoDiscover: config.BoolPtr(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAPIService{statusValue: service.Status{Discovered: true}}
			srv := New("127.0.0.1:8080", nil, svc)

			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if svc.openRequest != tt.wantOpenRequest {
				t.Fatalf("OpenPort() request = %+v, want %+v", svc.openRequest, tt.wantOpenRequest)
			}
			if svc.closeRequest != tt.wantCloseRequest {
				t.Fatalf("ClosePort() request = %+v, want %+v", svc.closeRequest, tt.wantCloseRequest)
			}
			if svc.settingsReq.ListenAddr != tt.wantSettingsListenAddr {
				t.Fatalf("UpdateSettings() ListenAddr = %q, want %q", svc.settingsReq.ListenAddr, tt.wantSettingsListenAddr)
			}
			if (svc.settingsReq.AutoDiscover == nil) != (tt.wantSettingsAutoDiscover == nil) {
				t.Fatalf("UpdateSettings() AutoDiscover nil mismatch: got=%v want=%v", svc.settingsReq.AutoDiscover, tt.wantSettingsAutoDiscover)
			}
			if svc.settingsReq.AutoDiscover != nil && tt.wantSettingsAutoDiscover != nil && *svc.settingsReq.AutoDiscover != *tt.wantSettingsAutoDiscover {
				t.Fatalf("UpdateSettings() AutoDiscover = %v, want %v", *svc.settingsReq.AutoDiscover, *tt.wantSettingsAutoDiscover)
			}
		})
	}
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
		wantStatus int // httptest.ResponseRecorder.Code
	}{
		{name: "探索エラー", method: http.MethodPost, path: "/api/discover", wantStatus: http.StatusBadGateway},
		{name: "ポート開放エラー", method: http.MethodPost, path: "/api/ports/open", body: `{"protocol":"TCP"}`, wantStatus: http.StatusBadRequest},
		{name: "ポート閉鎖エラー", method: http.MethodPost, path: "/api/ports/close", body: `{"protocol":"TCP"}`, wantStatus: http.StatusBadRequest},
		{name: "設定エラー", method: http.MethodPost, path: "/api/settings", body: `{"listen_addr":"0.0.0.0:8080"}`, wantStatus: http.StatusBadRequest},
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
