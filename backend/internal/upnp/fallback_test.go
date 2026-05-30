package upnp

import (
	"net"
	"testing"
)

func TestFallbackGatewayLocations(t *testing.T) {
	tests := []struct {
		name string
		cidr string
		want string
	}{
		{
			name: "includes default root description",
			cidr: "192.168.1.20/24",
			want: "http://192.168.1.1:5000/rootDesc.xml",
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
				if location == tt.want {
					return
				}
			}
			t.Fatalf("fallbackGatewayLocations() missing %s", tt.want)
		})
	}
}

func TestFallbackControlCandidates(t *testing.T) {
	tests := []struct {
		name            string
		cidr            string
		wantURL         string
		wantServiceType string
	}{
		{
			name:            "includes wan ip candidate",
			cidr:            "192.168.1.20/24",
			wantURL:         "http://192.168.1.1:5000/upnp/control/WANIPConn1",
			wantServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
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
				if candidate.ControlURL == tt.wantURL && candidate.ServiceType == tt.wantServiceType {
					return
				}
			}
			t.Fatalf("fallbackControlCandidates() missing %s", tt.wantURL)
		})
	}
}
