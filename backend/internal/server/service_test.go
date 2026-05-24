package server

import (
	"testing"

	"github.com/clagon/port-mapper/backend/internal/upnp"
)

func TestDefaultPortMapperFactoryUsesTimeoutClient(t *testing.T) {
	client := defaultPortMapperFactory(upnp.DiscoveryResult{ServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2", ControlURL: "http://192.168.1.1:1900/control"})
	soap, ok := client.(*upnp.SOAPClient)
	if !ok {
		t.Fatalf("defaultPortMapperFactory() type = %T, want *upnp.SOAPClient", client)
	}
	if soap.HTTPClient == nil {
		t.Fatal("HTTPClient = nil")
	}
	if soap.HTTPClient.Timeout <= 0 {
		t.Fatalf("HTTPClient timeout = %v, want > 0", soap.HTTPClient.Timeout)
	}
}
