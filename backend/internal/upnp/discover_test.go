package upnp

import (
	"os"
	"path/filepath"
	"testing"
)

func readTestData(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", name, err)
	}
	return string(b)
}

func TestParseRootDevice(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		baseURL string
		wantErr bool
		wantURL string
		wantType string
	}{
		{
			name:    "wanipconnection1",
			xml:     readTestData(t, "rootdesc-wanipconnection1.xml"),
			baseURL: "http://192.168.1.1:1900/root.xml",
			wantURL: "http://192.168.1.1:1900/upnp/control/WANIPConn1",
			wantType: "urn:schemas-upnp-org:service:WANIPConnection:1",
		},
		{
			name:    "wanipconnection2 preferred over ppp",
			xml:     readTestData(t, "rootdesc-wanipconnection2.xml"),
			baseURL: "http://192.168.1.1:1900/root.xml",
			wantURL: "http://192.168.1.1:1900/upnp/control/WANIPConn2",
			wantType: "urn:schemas-upnp-org:service:WANIPConnection:2",
		},
		{
			name:    "wanpppconnection fallback",
			xml:     readTestData(t, "rootdesc-wanpppconnection1.xml"),
			baseURL: "http://192.168.1.1:1900/root.xml",
			wantURL: "http://192.168.1.1:1900/ppp/control/WANPPPConn1",
			wantType: "urn:schemas-upnp-org:service:WANPPPConnection:1",
		},
		{
			name:    "malformed xml",
			xml:     "<xml",
			baseURL: "http://192.168.1.1:1900/root.xml",
			wantErr: true,
		},
		{
			name:    "no matching service",
			xml:     readTestData(t, "rootdesc-nomatch.xml"),
			baseURL: "http://192.168.1.1:1900/root.xml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRootDevice([]byte(tt.xml), tt.baseURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseRootDevice() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRootDevice() error = %v", err)
			}
			if got.ControlURL != tt.wantURL {
				t.Fatalf("ControlURL = %q, want %q", got.ControlURL, tt.wantURL)
			}
			if got.ServiceType != tt.wantType {
				t.Fatalf("ServiceType = %q, want %q", got.ServiceType, tt.wantType)
			}
		})
	}
}
