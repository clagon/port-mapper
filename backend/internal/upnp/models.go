package upnp

// PortMapping represents a single port forwarding request.
type PortMapping struct {
	Protocol             string `json:"protocol"`
	ExternalPort         int    `json:"external_port"`
	InternalIP           string `json:"internal_ip"`
	InternalPort         int    `json:"internal_port"`
	Description          string `json:"description"`
	LeaseDurationSeconds int    `json:"lease_duration_seconds"`
}

const MaxLeaseDurationSeconds = 7 * 24 * 60 * 60

// DiscoveryResult is the selected UPnP control endpoint discovered from a root description.
type DiscoveryResult struct {
	ServiceType string
	ControlURL  string
}
