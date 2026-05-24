package upnp

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
)

type rootDevice struct {
	XMLName xml.Name `xml:"root"`
	Device  device   `xml:"device"`
}

type device struct {
	ServiceList serviceList `xml:"serviceList"`
}

type serviceList struct {
	Services []service `xml:"service"`
}

type service struct {
	ServiceType string `xml:"serviceType"`
	ControlURL  string `xml:"controlURL"`
}

// ParseRootDevice parses a UPnP root device description and selects the best WAN service.
func ParseRootDevice(data []byte, baseURL string) (DiscoveryResult, error) {
	var root rootDevice
	if err := xml.Unmarshal(data, &root); err != nil {
		return DiscoveryResult{}, fmt.Errorf("parse root device xml: %w", err)
	}

	selected, ok := selectService(root.Device.ServiceList.Services)
	if !ok {
		return DiscoveryResult{}, fmt.Errorf("no supported WAN service found")
	}

	resolved, err := resolveControlURL(baseURL, selected.ControlURL)
	if err != nil {
		return DiscoveryResult{}, err
	}

	return DiscoveryResult{
		ServiceType: strings.TrimSpace(selected.ServiceType),
		ControlURL:  resolved,
	}, nil
}

func selectService(services []service) (service, bool) {
	priority := map[string]int{
		"urn:schemas-upnp-org:service:WANIPConnection:2": 3,
		"urn:schemas-upnp-org:service:WANIPConnection:1": 2,
		"urn:schemas-upnp-org:service:WANPPPConnection:1": 1,
	}

	var best service
	bestScore := 0
	for _, s := range services {
		score := priority[strings.TrimSpace(s.ServiceType)]
		if score > bestScore {
			bestScore = score
			best = s
		}
	}
	if bestScore == 0 {
		return service{}, false
	}
	return best, true
}

func resolveControlURL(baseURL, controlURL string) (string, error) {
	controlURL = strings.TrimSpace(controlURL)
	if controlURL == "" {
		return "", fmt.Errorf("empty control url")
	}
	if u, err := url.Parse(controlURL); err == nil && u.IsAbs() {
		return controlURL, nil
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base url %q: %w", baseURL, err)
	}
	ref, err := url.Parse(controlURL)
	if err != nil {
		return "", fmt.Errorf("parse control url %q: %w", controlURL, err)
	}
	return base.ResolveReference(ref).String(), nil
}
