package upnp

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
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
		name     string
		xml      string
		baseURL  string
		wantErr  bool
		wantURL  string
		wantType string
	}{
		{
			name:     "wanipconnection1",
			xml:      readTestData(t, "rootdesc-wanipconnection1.xml"),
			baseURL:  "http://192.168.1.1:1900/root.xml",
			wantURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn1",
			wantType: "urn:schemas-upnp-org:service:WANIPConnection:1",
		},
		{
			name:     "wanipconnection2 preferred over ppp",
			xml:      readTestData(t, "rootdesc-wanipconnection2.xml"),
			baseURL:  "http://192.168.1.1:1900/root.xml",
			wantURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn2",
			wantType: "urn:schemas-upnp-org:service:WANIPConnection:2",
		},
		{
			name:     "nested igd tree",
			xml:      readTestData(t, "rootdesc-nested-igd.xml"),
			baseURL:  "http://192.168.1.1:1900/root.xml",
			wantURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn2",
			wantType: "urn:schemas-upnp-org:service:WANIPConnection:2",
		},
		{
			name:     "absolute same-host control url is allowed",
			xml:      readTestData(t, "rootdesc-absolute-controlurl.xml"),
			baseURL:  "http://192.168.1.1:1900/root.xml",
			wantURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn2",
			wantType: "urn:schemas-upnp-org:service:WANIPConnection:2",
		},
		{
			name:    "absolute different-host control url is rejected",
			xml:     readTestData(t, "rootdesc-malicious-controlurl.xml"),
			baseURL: "http://192.168.1.1:1900/root.xml",
			wantErr: true,
		},
		{
			name: "nested wan service",
			xml: `<?xml version="1.0"?>
<root>
  <device>
    <deviceList>
      <device>
        <deviceList>
          <device>
            <serviceList>
              <service>
                <serviceType>urn:schemas-upnp-org:service:WANIPConnection:1</serviceType>
                <controlURL>/ctl/IPConn</controlURL>
              </service>
            </serviceList>
          </device>
        </deviceList>
      </device>
    </deviceList>
  </device>
</root>`,
			baseURL:  "http://192.168.1.1:1900/root.xml",
			wantURL:  "http://192.168.1.1:1900/ctl/IPConn",
			wantType: "urn:schemas-upnp-org:service:WANIPConnection:1",
		},
		{
			name: "urlbase preferred for relative control url",
			xml: `<?xml version="1.0"?>
<root>
  <URLBase>http://192.168.1.1:5431/</URLBase>
  <device>
    <serviceList>
      <service>
        <serviceType>urn:schemas-upnp-org:service:WANIPConnection:1</serviceType>
        <controlURL>upnp/control/WANIPConn1</controlURL>
      </service>
    </serviceList>
  </device>
</root>`,
			baseURL:  "http://192.168.1.1:1900/root.xml",
			wantURL:  "http://192.168.1.1:5431/upnp/control/WANIPConn1",
			wantType: "urn:schemas-upnp-org:service:WANIPConnection:1",
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

func TestDiscoverFromLocation(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "discovers control url from location",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, readTestData(t, "rootdesc-wanipconnection2.xml"))
			}))
			defer server.Close()

			got, err := discoverFromLocation(server.URL, func(location string) ([]byte, error) {
				resp, err := http.Get(location)
				if err != nil {
					return nil, err
				}
				defer resp.Body.Close()
				return io.ReadAll(resp.Body)
			})
			if err != nil {
				t.Fatalf("discoverFromLocation() error = %v", err)
			}
			if got.ControlURL != server.URL+"/upnp/control/WANIPConn2" {
				t.Fatalf("ControlURL = %q", got.ControlURL)
			}
		})
	}
}

func TestLiveDiscover(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "live discover",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if os.Getenv("PORT_MAPPER_LIVE_UPNP") != "1" {
				t.Skip("set PORT_MAPPER_LIVE_UPNP=1 to run live UPnP discovery")
			}

			ifaces, err := discoverInterfaces()
			if err != nil {
				t.Fatalf("discoverInterfaces() error = %v", err)
			}
			for _, iface := range ifaces {
				t.Logf("interface ip=%s name=%s", iface.ListenAddr.IP, interfaceName(iface.Interface))
				responses, err := collectSSDPResponses(iface)
				if err != nil {
					t.Logf("collectSSDPResponses() error = %v", err)
					continue
				}
				for _, response := range responses {
					t.Logf("ssdp target=%q st=%q usn=%q location=%q score=%d", response.SearchTarget, response.ST, response.USN, response.Location, ssdpCandidateScore(response))
				}
			}
			ipv6Ifaces, err := discoverIPv6Interfaces()
			if err != nil {
				t.Logf("discoverIPv6Interfaces() error = %v", err)
			}
			for _, iface := range ipv6Ifaces {
				t.Logf("ipv6 interface bind=%s name=%s", iface.ListenAddr.IP, interfaceName(iface.Interface))
				responses, err := collectSSDPResponsesIPv6(iface)
				if err != nil {
					t.Logf("collectSSDPResponsesIPv6() error = %v", err)
					continue
				}
				for _, response := range responses {
					t.Logf("ipv6 ssdp target=%q st=%q usn=%q location=%q score=%d", response.SearchTarget, response.ST, response.USN, response.Location, ssdpCandidateScore(response))
				}
			}

			got, err := Discover()
			if err != nil {
				t.Fatalf("Discover() error = %v", err)
			}
			if got.ServiceType == "" {
				t.Fatal("ServiceType is empty")
			}
			if got.ControlURL == "" {
				t.Fatal("ControlURL is empty")
			}
		})
	}
}

func interfaceName(iface *net.Interface) string {
	if iface == nil {
		return ""
	}
	return iface.Name
}
