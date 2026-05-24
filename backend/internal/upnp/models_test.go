package upnp

import (
	"encoding/json"
	"testing"
)

func TestPortMappingJSONTags(t *testing.T) {
	payload := []byte(`{"protocol":"TCP","external_port":8080,"internal_ip":"192.168.1.20","internal_port":8080,"description":"test mapping","lease_duration_seconds":3600}`)
	var got PortMapping
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got.Protocol != "TCP" {
		t.Fatalf("Protocol = %q, want TCP", got.Protocol)
	}
	if got.ExternalPort != 8080 || got.InternalPort != 8080 {
		t.Fatalf("ports = %+v", got)
	}
	if got.InternalIP != "192.168.1.20" || got.Description != "test mapping" || got.LeaseDurationSeconds != 3600 {
		t.Fatalf("mapping = %+v", got)
	}
}
