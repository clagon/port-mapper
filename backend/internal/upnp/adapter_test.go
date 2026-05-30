package upnp

import (
	"testing"

	"github.com/clagon/port-mapper/backend/internal/domain"
)

func TestNewSOAPPortMapperUsesTimeoutClient(t *testing.T) {
	tests := []struct {
		name string
		in   domain.DiscoveryResult
	}{
		{
			name: "timeout client",
			in:   domain.DiscoveryResult{ServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2", ControlURL: "http://192.168.1.1:1900/control"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewSOAPPortMapper(tt.in)
			soap, ok := client.(*SOAPClient)
			if !ok {
				t.Fatalf("NewSOAPPortMapper() type = %T, want *SOAPClient", client)
			}
			if soap.HTTPClient == nil {
				t.Fatal("HTTPClient = nil")
			}
			if soap.HTTPClient.Timeout <= 0 {
				t.Fatalf("HTTPClient timeout = %v, want > 0", soap.HTTPClient.Timeout)
			}
		})
	}
}
