package upnp

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clagon/port-mapper/backend/internal/domain"
)

func TestSOAPEnvelope(t *testing.T) {
	tests := []struct {
		name               string
		action             string
		serviceType        string
		body               map[string]string
		wantBodySubstrings []string // SOAP envelope substrings
	}{
		{
			name:        "get external ip action and namespace",
			action:      "GetExternalIPAddress",
			serviceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
			wantBodySubstrings: []string{
				`<u:GetExternalIPAddress xmlns:u="urn:schemas-upnp-org:service:WANIPConnection:2">`,
				`<s:Envelope`,
			},
		},
		{
			name:        "add port mapping includes all fields",
			action:      "AddPortMapping",
			serviceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
			body: map[string]string{
				"NewRemoteHost":             "",
				"NewExternalPort":           "8080",
				"NewProtocol":               "TCP",
				"NewInternalPort":           "8080",
				"NewInternalClient":         "192.168.1.20",
				"NewEnabled":                "1",
				"NewPortMappingDescription": "test mapping",
				"NewLeaseDuration":          "3600",
			},
			wantBodySubstrings: []string{
				`<u:AddPortMapping xmlns:u="urn:schemas-upnp-org:service:WANIPConnection:2">`,
				`<NewExternalPort>8080</NewExternalPort>`,
				`<NewProtocol>TCP</NewProtocol>`,
				`<NewInternalClient>192.168.1.20</NewInternalClient>`,
				`<NewPortMappingDescription>test mapping</NewPortMappingDescription>`,
				`<NewLeaseDuration>3600</NewLeaseDuration>`,
			},
		},
		{
			name:        "delete port mapping includes protocol and external port",
			action:      "DeletePortMapping",
			serviceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
			body: map[string]string{
				"NewExternalPort": "8080",
				"NewProtocol":     "UDP",
			},
			wantBodySubstrings: []string{
				`<u:DeletePortMapping xmlns:u="urn:schemas-upnp-org:service:WANIPConnection:2">`,
				`<NewExternalPort>8080</NewExternalPort>`,
				`<NewProtocol>UDP</NewProtocol>`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildSOAPEnvelope(tt.action, tt.serviceType, tt.body)
			if err != nil {
				t.Fatalf("buildSOAPEnvelope() error = %v", err)
			}
			body := string(got)
			for _, want := range tt.wantBodySubstrings {
				if !strings.Contains(body, want) {
					t.Fatalf("envelope missing %q\n%s", want, body)
				}
			}
		})
	}
}

func TestSOAPGetExternalIPAddress(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "get external ip",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Fatalf("method = %s, want POST", r.Method)
				}
				if got := r.Header.Get("SOAPAction"); got != `"urn:schemas-upnp-org:service:WANIPConnection:2#GetExternalIPAddress"` {
					t.Fatalf("SOAPAction = %q", got)
				}
				_, _ = io.ReadAll(r.Body)
				w.Header().Set("Content-Type", "text/xml")
				_, _ = w.Write([]byte(`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <u:GetExternalIPAddressResponse xmlns:u="urn:schemas-upnp-org:service:WANIPConnection:2">
      <NewExternalIPAddress>203.0.113.42</NewExternalIPAddress>
    </u:GetExternalIPAddressResponse>
  </s:Body>
</s:Envelope>`))
			}))
			defer server.Close()

			c := &SOAPClient{
				Endpoint:    server.URL,
				ServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
				HTTPClient:  server.Client(),
			}
			got, err := c.GetExternalIPAddress()
			if err != nil {
				t.Fatalf("GetExternalIPAddress() error = %v", err)
			}
			if got != "203.0.113.42" {
				t.Fatalf("GetExternalIPAddress() = %q, want %q", got, "203.0.113.42")
			}
		})
	}
}

func TestSOAPClientDefaultTimeout(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "default timeout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (&SOAPClient{}).client(); got == nil {
				t.Fatal("client() = nil")
			} else if got.Timeout <= 0 {
				t.Fatalf("client timeout = %v, want > 0", got.Timeout)
			}
		})
	}
}

func TestSOAPGenericPortMappingEntryIndexInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <s:Fault>
      <faultcode>s:Client</faultcode>
      <faultstring>UPnPError</faultstring>
      <detail>
        <UPnPError xmlns="urn:schemas-upnp-org:control-1-0">
          <errorCode>713</errorCode>
          <errorDescription>SpecifiedArrayIndexInvalid</errorDescription>
        </UPnPError>
      </detail>
    </s:Fault>
  </s:Body>
</s:Envelope>`))
	}))
	defer server.Close()

	c := &SOAPClient{
		Endpoint:    server.URL,
		ServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
		HTTPClient:  server.Client(),
	}
	_, err := c.GetGenericPortMappingEntry(0)
	if !errors.Is(err, domain.ErrNoPortMappingEntry) {
		t.Fatalf("GetGenericPortMappingEntry() error = %v, want ErrNoPortMappingEntry", err)
	}
}

func TestSOAPFault(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "soap fault",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`<?xml version="1.0"?>
<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
  <s:Body>
    <s:Fault>
      <faultcode>s:Client</faultcode>
      <faultstring>UPnPError</faultstring>
      <detail>
        <UPnPError xmlns="urn:schemas-upnp-org:control-1-0">
          <errorCode>401</errorCode>
          <errorDescription>Invalid Action</errorDescription>
        </UPnPError>
      </detail>
    </s:Fault>
  </s:Body>
</s:Envelope>`))
			}))
			defer server.Close()

			c := &SOAPClient{
				Endpoint:    server.URL,
				ServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
				HTTPClient:  server.Client(),
			}
			_, err := c.GetExternalIPAddress()
			if err == nil {
				t.Fatal("GetExternalIPAddress() error = nil, want error")
			}
			if !strings.Contains(err.Error(), "Invalid Action") {
				t.Fatalf("error = %v, want SOAP fault", err)
			}
		})
	}
}
