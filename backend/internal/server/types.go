package server

import "github.com/clagon/port-mapper/backend/internal/upnp"

// HealthResponse is the JSON payload returned by GET /api/health.
type HealthResponse struct {
	Ok bool `json:"ok"`
}

// ActionResponse is the standard JSON payload returned by mutating endpoints.
type ActionResponse struct {
	Ok bool `json:"ok"`
}

// StatusResponse describes the current discovery and mapping state.
type StatusResponse struct {
	Discovered  bool               `json:"discovered"`
	ServiceType string             `json:"service_type,omitempty"`
	ControlURL  string             `json:"control_url,omitempty"`
	ExternalIP  string             `json:"external_ip,omitempty"`
	LocalIP     string             `json:"local_ip,omitempty"`
	Ports       []upnp.PortMapping `json:"ports"`
}
