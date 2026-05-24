package upnp

// PortMapping represents a single port forwarding request.
type PortMapping struct {
	Protocol            string
	ExternalPort        int
	InternalIP          string
	InternalPort        int
	Description         string
	LeaseDurationSeconds int
}

const MaxLeaseDurationSeconds = 7 * 24 * 60 * 60
