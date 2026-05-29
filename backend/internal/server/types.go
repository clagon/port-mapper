package server

import (
	"github.com/clagon/port-mapper/backend/internal/service"
	"github.com/clagon/port-mapper/backend/internal/upnp"
)

// HealthResponse is the JSON payload returned by GET /api/health.
type HealthResponse struct {
	Ok bool `json:"ok"`
}

// ActionResponse is the standard JSON payload returned by mutating endpoints.
type ActionResponse struct {
	Ok bool `json:"ok"`
}

// PortMappingRequest is the JSON payload accepted by POST /api/ports/open.
type PortMappingRequest struct {
	Protocol             string `json:"protocol"`
	ExternalPort         int    `json:"external_port"`
	InternalIP           string `json:"internal_ip"`
	InternalPort         int    `json:"internal_port"`
	Description          string `json:"description"`
	LeaseDurationSeconds int    `json:"lease_duration_seconds"`
}

// ClosePortRequest is the JSON payload accepted by POST /api/ports/close.
type ClosePortRequest struct {
	Protocol     string `json:"protocol"`
	ExternalPort int    `json:"external_port"`
}

// PortMappingResponse is the JSON payload returned for a port mapping.
type PortMappingResponse struct {
	Protocol             string `json:"protocol"`
	ExternalPort         int    `json:"external_port"`
	InternalIP           string `json:"internal_ip"`
	InternalPort         int    `json:"internal_port"`
	Description          string `json:"description"`
	LeaseDurationSeconds int    `json:"lease_duration_seconds"`
}

// StatusResponse describes the current discovery and mapping state.
type StatusResponse struct {
	Discovered  bool                  `json:"discovered"`
	ServiceType string                `json:"service_type,omitempty"`
	ControlURL  string                `json:"control_url,omitempty"`
	ExternalIP  string                `json:"external_ip,omitempty"`
	LocalIP     string                `json:"local_ip,omitempty"`
	Ports       []PortMappingResponse `json:"ports"`
}

func (r PortMappingRequest) toPortMapping() upnp.PortMapping {
	return upnp.PortMapping{
		Protocol:             r.Protocol,
		ExternalPort:         r.ExternalPort,
		InternalIP:           r.InternalIP,
		InternalPort:         r.InternalPort,
		Description:          r.Description,
		LeaseDurationSeconds: r.LeaseDurationSeconds,
	}
}

func (r ClosePortRequest) toPortMapping() upnp.PortMapping {
	return upnp.PortMapping{
		Protocol:     r.Protocol,
		ExternalPort: r.ExternalPort,
	}
}

func newPortMappingResponse(mapping upnp.PortMapping) PortMappingResponse {
	return PortMappingResponse{
		Protocol:             mapping.Protocol,
		ExternalPort:         mapping.ExternalPort,
		InternalIP:           mapping.InternalIP,
		InternalPort:         mapping.InternalPort,
		Description:          mapping.Description,
		LeaseDurationSeconds: mapping.LeaseDurationSeconds,
	}
}

func newPortMappingResponses(mappings []upnp.PortMapping) []PortMappingResponse {
	ports := make([]PortMappingResponse, len(mappings))
	for i, mapping := range mappings {
		ports[i] = newPortMappingResponse(mapping)
	}
	return ports
}

func newStatusResponse(s service.Status) StatusResponse {
	return StatusResponse{
		Discovered:  s.Discovered,
		ServiceType: s.ServiceType,
		ControlURL:  s.ControlURL,
		ExternalIP:  s.ExternalIP,
		LocalIP:     s.LocalIP,
		Ports:       newPortMappingResponses(s.Ports),
	}
}
