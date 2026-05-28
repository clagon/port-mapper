package upnp

import (
	"net"
	"testing"
)

func TestFallbackGatewayLocations(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.20/24")
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

	want := "http://192.168.1.1:5000/rootDesc.xml"
	for _, location := range got {
		if location == want {
			return
		}
	}
	t.Fatalf("fallbackGatewayLocations() missing %s", want)
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
