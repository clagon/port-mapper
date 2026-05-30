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
	type want struct {
		status     int
		healthOK   bool
		discovered bool
		controlURL string
		listenAddr string
	}
	tests := []struct {
		name string
		path string
		want want
	}{
		{
			name: "ヘルスチェックを返す",
			path: "/api/health",
			want: want{
				status:   http.StatusOK,
				healthOK: true,
			},
		},
		{
			name: "ステータスを返す",
			path: "/api/status",
			want: want{
				status:     http.StatusOK,
				discovered: true,
				controlURL: svc.statusValue.ControlURL,
			},
		},
		{
			name: "設定を返す",
			path: "/api/settings",
			want: want{
				status:     http.StatusOK,
				listenAddr: svc.settingsValue.ListenAddr,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			srv.Handler().ServeHTTP(rec, req)
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}

			switch tt.path {
			case "/api/health":
				var got HealthResponse
				decodeJSON(t, rec.Body, &got)
				if got.Ok != tt.want.healthOK {
					t.Fatalf("HealthResponse.Ok = %v, want %v", got.Ok, tt.want.healthOK)
				}
			case "/api/status":
				var got StatusResponse
				decodeJSON(t, rec.Body, &got)
				if got.Discovered != tt.want.discovered {
					t.Fatalf("StatusResponse.Discovered = %v, want %v", got.Discovered, tt.want.discovered)
				}
				if got.ControlURL != tt.want.controlURL {
					t.Fatalf("StatusResponse.ControlURL = %q, want %q", got.ControlURL, tt.want.controlURL)
				}
			case "/api/settings":
				var got config.Config
				decodeJSON(t, rec.Body, &got)
				if got.ListenAddr != tt.want.listenAddr {
					t.Fatalf("Config.ListenAddr = %q, want %q", got.ListenAddr, tt.want.listenAddr)
				}
			}
		})
	}
}

func TestMutatingEndpointsBindRequests(t *testing.T) {
	type want struct {
		status       int
		openRequest  domain.PortMapping
		closeRequest domain.PortMapping
		settingsReq  config.Config
	}
	tests := []struct {
		name string
		path string
		body []byte
		want want
	}{
		{
			name: "探索を受け付ける",
			path: "/api/discover",
			want: want{
				status: http.StatusAccepted,
			},
		},
		{
			name: "ポート開放リクエストを束縛する",
			path: "/api/ports/open",
			body: []byte(`{"protocol":"TCP","external_port":8080,"internal_ip":"192.168.1.20","internal_port":8080}`),
			want: want{
				status: http.StatusAccepted,
				openRequest: domain.PortMapping{
					Protocol:     "TCP",
					ExternalPort: 8080,
					InternalIP:   "192.168.1.20",
					InternalPort: 8080,
				},
			},
		},
		{
			name: "ポート閉鎖リクエストを束縛する",
			path: "/api/ports/close",
			body: []byte(`{"protocol":"UDP","external_port":5353}`),
			want: want{
				status: http.StatusAccepted,
				closeRequest: domain.PortMapping{
					Protocol:     "UDP",
					ExternalPort: 5353,
				},
			},
		},
		{
			name: "設定更新リクエストを束縛する",
			path: "/api/settings",
			body: []byte(`{"listen_addr":"127.0.0.1:9090","auto_discover":false}`),
			want: want{
				status:      http.StatusOK,
				settingsReq: config.Config{ListenAddr: "127.0.0.1:9090", AutoDiscover: config.BoolPtr(false)},
			},
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
			if rec.Code != tt.want.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want.status)
			}
			if svc.openRequest != tt.want.openRequest {
				t.Fatalf("OpenPort() request = %+v, want %+v", svc.openRequest, tt.want.openRequest)
			}
			if svc.closeRequest != tt.want.closeRequest {
				t.Fatalf("ClosePort() request = %+v, want %+v", svc.closeRequest, tt.want.closeRequest)
			}
			if svc.settingsReq.ListenAddr != tt.want.settingsReq.ListenAddr {
				t.Fatalf("UpdateSettings() ListenAddr = %q, want %q", svc.settingsReq.ListenAddr, tt.want.settingsReq.ListenAddr)
			}
			if (svc.settingsReq.AutoDiscover == nil) != (tt.want.settingsReq.AutoDiscover == nil) {
				t.Fatalf("UpdateSettings() AutoDiscover nil mismatch: got=%v want=%v", svc.settingsReq.AutoDiscover, tt.want.settingsReq.AutoDiscover)
			}
			if svc.settingsReq.AutoDiscover != nil && tt.want.settingsReq.AutoDiscover != nil && *svc.settingsReq.AutoDiscover != *tt.want.settingsReq.AutoDiscover {
				t.Fatalf("UpdateSettings() AutoDiscover = %v, want %v", *svc.settingsReq.AutoDiscover, *tt.want.settingsReq.AutoDiscover)
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
		wantStatus int
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
