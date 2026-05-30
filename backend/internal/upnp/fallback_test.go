package upnp

import (
	"net"
	"testing"
)

func TestFallbackGatewayLocations(t *testing.T) {
	tests := []struct {
		name            string
		cidr            string
		wantLocationURL string // fallback root description URL
	}{
		{
			name:            "includes default root description",
			cidr:            "192.168.1.20/24",
			wantLocationURL: "http://192.168.1.1:5000/rootDesc.xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ipNet, err := net.ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatal(err)
			}

			got := fallbackGatewayLocations([]discoverInterface{{
				ListenAddr: &net.UDPAddr{IP: net.ParseIP("192.168.1.20"), Port: 0},
				IPNet:      ipNet,
			}})
			if len(got) == 0 {
				t.Fatal("fallbackGatewayLocations() returned no locations")
			}

			for _, location := range got {
				if location == tt.wantLocationURL {
					return
				}
			}
			t.Fatalf("fallbackGatewayLocations() missing %s", tt.wantLocationURL)
		})
	}
}

func TestFallbackControlCandidates(t *testing.T) {
	tests := []struct {
		name               string
		cidr               string
		wantControlURL     string // DiscoveryResult.ControlURL
		wantControlService string // DiscoveryResult.ServiceType
	}{
		{
			name:               "includes wan ip candidate",
			cidr:               "192.168.1.20/24",
			wantControlURL:     "http://192.168.1.1:5000/upnp/control/WANIPConn1",
			wantControlService: "urn:schemas-upnp-org:service:WANIPConnection:2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ipNet, err := net.ParseCIDR(tt.cidr)
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

			for _, candidate := range got {
				if candidate.ControlURL == tt.wantControlURL && candidate.ServiceType == tt.wantControlService {
					return
				}
			}
			t.Fatalf("fallbackControlCandidates() missing %s", tt.wantControlURL)
		})
	}
}
