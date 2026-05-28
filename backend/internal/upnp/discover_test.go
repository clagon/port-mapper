package upnp

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSDPSearchTargets(t *testing.T) {
	for _, want := range []string{
		"urn:schemas-upnp-org:device:InternetGatewayDevice:1",
		"upnp:rootdevice",
		"ssdp:all",
	} {
		found := false
		for _, got := range ssdpSearchTargets {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("ssdpSearchTargets missing %q in %#v", want, ssdpSearchTargets)
		}
	}
}

func TestBuildSSDPSearchRequest(t *testing.T) {
	msg := buildMSearch("upnp:rootdevice")
	if !strings.Contains(msg, "ST: upnp:rootdevice") {
		t.Fatalf("search request missing ST line: %q", msg)
	}
	if !strings.HasSuffix(msg, "\r\n\r\n") {
		t.Fatalf("search request should end with CRLF CRLF: %q", msg)
	}
}

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
}

func TestLiveDiscover(t *testing.T) {
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
}

func interfaceName(iface *net.Interface) string {
	if iface == nil {
		return ""
	}
	return iface.Name
}

func TestBuildMSearch(t *testing.T) {
	got := buildMSearch("urn:schemas-upnp-org:device:InternetGatewayDevice:1")
	for _, want := range []string{
		"M-SEARCH * HTTP/1.1\r\n",
		"HOST: 239.255.255.250:1900\r\n",
		"MX: 2\r\n",
		"ST: urn:schemas-upnp-org:device:InternetGatewayDevice:1\r\n",
		"USER-AGENT: Windows/10 UPnP/1.1 port-mapper/1.0\r\n",
		"\r\n\r\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("buildMSearch() missing %q in %q", want, got)
		}
	}
}

func TestFallbackControlCandidates(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.20/24")
	if err != nil {
		t.Fatal(err)
	}

	got := fallbackControlCandidates([]discoverInterface{{
		ListenAddr: &net.UDPAddr{IP: net.ParseIP("192.168.1.20"), Port: 0},
		IPNet:      ipNet,
	}})
	if len(got) == 0 {
		t.Fatal("fallbackControlCandidates() returned no candidates")
	}

	wantURL := "http://192.168.1.1:5000/upnp/control/WANIPConn1"
	for _, candidate := range got {
		if candidate.ControlURL == wantURL && candidate.ServiceType == "urn:schemas-upnp-org:service:WANIPConnection:2" {
			return
		}
	}
	t.Fatalf("fallbackControlCandidates() missing %s", wantURL)
}
