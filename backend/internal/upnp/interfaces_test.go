package upnp

import (
	"net"
	"testing"
)

func TestIPHelpers(t *testing.T) {
	ipNet := &net.IPNet{
		IP:   net.ParseIP("192.168.1.20"),
		Mask: net.CIDRMask(24, 32),
	}
	if got := firstUsableIPv4(ipNet); !got.Equal(net.ParseIP("192.168.1.1")) {
		t.Fatalf("firstUsableIPv4() = %s, want 192.168.1.1", got)
	}

	ip, parsedNet := ipv4NetFromAddr(ipNet)
	if !ip.Equal(net.ParseIP("192.168.1.20")) || parsedNet == nil {
		t.Fatalf("ipv4NetFromAddr() = %s, %v", ip, parsedNet)
	}

	if got := ipv6FromAddr(&net.IPAddr{IP: net.ParseIP("2001:db8::1")}); got == nil {
		t.Fatal("ipv6FromAddr() returned nil for IPv6")
	}
	if got := ipv6FromAddr(&net.IPAddr{IP: net.ParseIP("192.168.1.20")}); got != nil {
		t.Fatalf("ipv6FromAddr() = %s, want nil for IPv4", got)
	}
}
