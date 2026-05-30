package upnp

import (
	"errors"
	"strings"
	"testing"
)

func TestParseSSDPResponse(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "parses location and service fields",
			data: []byte(strings.Join([]string{
				"HTTP/1.1 200 OK",
				"ST: urn:schemas-upnp-org:service:WANIPConnection:1",
				"USN: uuid:device::urn:schemas-upnp-org:service:WANIPConnection:1",
				"LOCATION: http://192.168.1.1:1900/root.xml",
				"",
				"",
			}, "\r\n")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSSDPResponse(tt.data)
			if err != nil {
				t.Fatalf("parseSSDPResponse() error = %v", err)
			}
			if got.Location != "http://192.168.1.1:1900/root.xml" {
				t.Fatalf("Location = %q", got.Location)
			}
			if got.ST != "urn:schemas-upnp-org:service:WANIPConnection:1" {
				t.Fatalf("ST = %q", got.ST)
			}
			if got.USN == "" {
				t.Fatal("USN is empty")
			}
		})
	}
}

func TestSSDPSearchTargets(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "igd target", want: "urn:schemas-upnp-org:device:InternetGatewayDevice:1"},
		{name: "rootdevice target", want: "upnp:rootdevice"},
		{name: "all target", want: "ssdp:all"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for _, got := range ssdpSearchTargets {
				if got == tt.want {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("ssdpSearchTargets missing %q in %#v", tt.want, ssdpSearchTargets)
			}
		})
	}
}

func TestBuildMSearch(t *testing.T) {
	tests := []struct {
		name string
		st   string
		want []string
	}{
		{
			name: "request includes required headers",
			st:   "urn:schemas-upnp-org:device:InternetGatewayDevice:1",
			want: []string{
				"M-SEARCH * HTTP/1.1\r\n",
				"HOST: 239.255.255.250:1900\r\n",
				"MX: 2\r\n",
				"ST: urn:schemas-upnp-org:device:InternetGatewayDevice:1\r\n",
				"USER-AGENT: Windows/10 UPnP/1.1 port-mapper/1.0\r\n",
				"\r\n\r\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildMSearch(tt.st)
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Fatalf("buildMSearch() missing %q in %q", want, got)
				}
			}
		})
	}
}

func TestBuildSSDPSearchRequest(t *testing.T) {
	tests := []struct {
		name string
		st   string
	}{
		{
			name: "rootdevice request ends with double crlf",
			st:   "upnp:rootdevice",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := buildMSearch(tt.st)
			if !strings.Contains(msg, "ST: upnp:rootdevice") {
				t.Fatalf("search request missing ST line: %q", msg)
			}
			if !strings.HasSuffix(msg, "\r\n\r\n") {
				t.Fatalf("search request should end with CRLF CRLF: %q", msg)
			}
		})
	}
}

func TestSSDPCandidateScorePrefersWANServices(t *testing.T) {
	tests := []struct {
		name string
		wan  ssdpResponse
		root ssdpResponse
		wfa  ssdpResponse
	}{
		{
			name: "wan outranks root and wfa",
			wan:  ssdpResponse{ST: "urn:schemas-upnp-org:service:WANIPConnection:2"},
			root: ssdpResponse{ST: "upnp:rootdevice"},
			wfa:  ssdpResponse{ST: "urn:schemas-wifialliance-org:device:WFADevice:1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if ssdpCandidateScore(tt.wan) <= ssdpCandidateScore(tt.root) {
				t.Fatal("WAN service should outrank rootdevice")
			}
			if ssdpCandidateScore(tt.root) <= ssdpCandidateScore(tt.wfa) {
				t.Fatal("rootdevice should outrank unrelated WFA response")
			}
		})
	}
}

func TestDiscoverFromSSDPResponsesWFAOnlyWrapsNoGateway(t *testing.T) {
	tests := []struct {
		name      string
		responses []ssdpResponse
	}{
		{
			name: "wfa only wraps no gateway",
			responses: []ssdpResponse{{
				ST:       "urn:schemas-wifialliance-org:device:WFADevice:1",
				USN:      "uuid:device::urn:schemas-wifialliance-org:device:WFADevice:1",
				Location: "http://192.168.1.1:49152/wps_device.xml",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := discoverFromSSDPResponses(tt.responses, "no matching SSDP responses")
			if !errors.Is(err, errOnlyWFADevices) {
				t.Fatalf("error = %v, want errOnlyWFADevices", err)
			}
			if !errors.Is(err, ErrNoGateway) {
				t.Fatalf("error = %v, want ErrNoGateway", err)
			}
		})
	}
}
