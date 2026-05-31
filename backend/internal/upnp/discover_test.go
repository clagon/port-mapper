package upnp

import (
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
		name            string
		xml             string
		baseURL         string
		wantErr         bool   // ParseRootDevice() error presence
		wantControlURL  string // DiscoveryResult.ControlURL
		wantServiceType string // DiscoveryResult.ServiceType
	}{
		{
			name:            "wanipconnection1",
			xml:             readTestData(t, "rootdesc-wanipconnection1.xml"),
			baseURL:         "http://192.168.1.1:1900/root.xml",
			wantControlURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn1",
			wantServiceType: "urn:schemas-upnp-org:service:WANIPConnection:1",
		},
		{
			name:            "wanipconnection2 preferred over ppp",
			xml:             readTestData(t, "rootdesc-wanipconnection2.xml"),
			baseURL:         "http://192.168.1.1:1900/root.xml",
			wantControlURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn2",
			wantServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
		},
		{
			name:            "nested igd tree",
			xml:             readTestData(t, "rootdesc-nested-igd.xml"),
			baseURL:         "http://192.168.1.1:1900/root.xml",
			wantControlURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn2",
			wantServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
		},
		{
			name:            "absolute same-host control url is allowed",
			xml:             readTestData(t, "rootdesc-absolute-controlurl.xml"),
			baseURL:         "http://192.168.1.1:1900/root.xml",
			wantControlURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn2",
			wantServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
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
			baseURL:         "http://192.168.1.1:1900/root.xml",
			wantControlURL:  "http://192.168.1.1:1900/ctl/IPConn",
			wantServiceType: "urn:schemas-upnp-org:service:WANIPConnection:1",
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
			baseURL:         "http://192.168.1.1:1900/root.xml",
			wantControlURL:  "http://192.168.1.1:5431/upnp/control/WANIPConn1",
			wantServiceType: "urn:schemas-upnp-org:service:WANIPConnection:1",
		},
		{
			name: "urlbase different host is rejected",
			xml: `<?xml version="1.0"?>
<root>
  <URLBase>http://192.168.1.2:5431/</URLBase>
  <device>
    <serviceList>
      <service>
        <serviceType>urn:schemas-upnp-org:service:WANIPConnection:1</serviceType>
        <controlURL>upnp/control/WANIPConn1</controlURL>
      </service>
    </serviceList>
  </device>
</root>`,
			baseURL: "http://192.168.1.1:1900/root.xml",
			wantErr: true,
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
			if got.ControlURL != tt.wantControlURL {
				t.Fatalf("ControlURL = %q, want %q", got.ControlURL, tt.wantControlURL)
			}
			if got.ServiceType != tt.wantServiceType {
				t.Fatalf("ServiceType = %q, want %q", got.ServiceType, tt.wantServiceType)
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
			location := "http://192.168.1.1:1900/root.xml"

			got, err := discoverFromLocation(location, func(gotLocation string) ([]byte, error) {
				if gotLocation != location {
					t.Fatalf("location = %q, want %q", gotLocation, location)
				}
				return []byte(readTestData(t, "rootdesc-wanipconnection2.xml")), nil
			})
			if err != nil {
				t.Fatalf("discoverFromLocation() error = %v", err)
			}
			if got.ControlURL != "http://192.168.1.1:1900/upnp/control/WANIPConn2" {
				t.Fatalf("ControlURL = %q", got.ControlURL)
			}
		})
	}
}

func TestParseAllowedUPnPURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{name: "private ipv4", rawURL: "http://192.168.1.1:1900/root.xml"},
		{name: "link local ipv4", rawURL: "http://169.254.10.1/root.xml"},
		{name: "unique local ipv6", rawURL: "http://[fd00::1]:1900/root.xml"},
		{name: "link local ipv6", rawURL: "http://[fe80::1]:1900/root.xml"},
		{name: "scoped link local ipv6", rawURL: "http://[fe80::1%25eth0]:1900/root.xml"},
		{name: "https rejected", rawURL: "https://192.168.1.1/root.xml", wantErr: true},
		{name: "hostname rejected", rawURL: "http://router.local/root.xml", wantErr: true},
		{name: "loopback rejected", rawURL: "http://127.0.0.1/root.xml", wantErr: true},
		{name: "unspecified rejected", rawURL: "http://0.0.0.0/root.xml", wantErr: true},
		{name: "multicast rejected", rawURL: "http://239.255.255.250/root.xml", wantErr: true},
		{name: "public rejected", rawURL: "http://8.8.8.8/root.xml", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseAllowedUPnPURL(tt.rawURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseAllowedUPnPURL() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAllowedUPnPURL() error = %v", err)
			}
		})
	}
}

func TestReadRootDescriptionBodyLimit(t *testing.T) {
	_, err := readRootDescriptionBody(strings.NewReader(strings.Repeat("x", maxRootDescriptionBytes+1)))
	if err == nil {
		t.Fatal("readRootDescriptionBody() error = nil, want size limit error")
	}
}

func TestValidateUPnPRedirect(t *testing.T) {
	from := &http.Request{URL: mustParseURL(t, "http://192.168.1.1:1900/root.xml")}
	tests := []struct {
		name    string
		to      string
		wantErr bool
	}{
		{name: "same host", to: "http://192.168.1.1:5000/rootDesc.xml"},
		{name: "different host", to: "http://192.168.1.2:1900/root.xml", wantErr: true},
		{name: "public host", to: "http://8.8.8.8/root.xml", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{URL: mustParseURL(t, tt.to)}
			err := validateUPnPRedirect(req, []*http.Request{from})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("validateUPnPRedirect() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("validateUPnPRedirect() error = %v", err)
			}
		})
	}
}

func mustParseURL(t *testing.T, rawURL string) *url.URL {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	return u
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
