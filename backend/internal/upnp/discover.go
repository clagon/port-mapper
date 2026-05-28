package upnp

import (
	"errors"
	"fmt"
)

var (
	ErrNoGateway      = errors.New("no UPnP gateway discovered")
	errOnlyWFADevices = errors.New("UPnP discovery found only WPS/WFA devices; no InternetGatewayDevice/WAN service responded")
)

// Discover sends SSDP M-SEARCH requests and returns the first supported gateway it can resolve.
func Discover() (DiscoveryResult, error) {
	ifaces, err := discoverInterfaces()
	if err != nil {
		return DiscoveryResult{}, err
	}

	var lastErr error
	sawOnlyWFA := false
	for _, iface := range ifaces {
		result, err := discoverFromInterface(iface)
		if err == nil {
			return result, nil
		}
		if errors.Is(err, errOnlyWFADevices) {
			sawOnlyWFA = true
		}
		lastErr = err
	}

	ipv6Ifaces, err := discoverIPv6Interfaces()
	if err == nil {
		for _, iface := range ipv6Ifaces {
			result, err := discoverFromIPv6Interface(iface)
			if err == nil {
				return result, nil
			}
			if errors.Is(err, errOnlyWFADevices) {
				sawOnlyWFA = true
			}
			lastErr = err
		}
	} else {
		lastErr = err
	}

	result, err := probeGatewayControlURLs(ifaces)
	if err == nil {
		return result, nil
	}
	lastErr = err

	result, err = probeGatewayDescriptions(ifaces)
	if err == nil {
		return result, nil
	}
	lastErr = err

	if sawOnlyWFA {
		return DiscoveryResult{}, fmt.Errorf("%w: %v; fallback probes failed: %v", ErrNoGateway, errOnlyWFADevices, lastErr)
	}
	if lastErr != nil {
		return DiscoveryResult{}, fmt.Errorf("%w: %v", ErrNoGateway, lastErr)
	}
	return DiscoveryResult{}, ErrNoGateway
}

func discoverFromInterface(iface discoverInterface) (DiscoveryResult, error) {
	responses, err := collectSSDPResponses(iface)
	if err != nil {
		return DiscoveryResult{}, err
	}
	return discoverFromSSDPResponses(responses, "no matching SSDP responses")
}

func discoverFromIPv6Interface(iface discoverIPv6Interface) (DiscoveryResult, error) {
	responses, err := collectSSDPResponsesIPv6(iface)
	if err != nil {
		return DiscoveryResult{}, err
	}
	return discoverFromSSDPResponses(responses, "no matching IPv6 SSDP responses")
}

func discoverFromSSDPResponses(responses []ssdpResponse, noMatchMessage string) (DiscoveryResult, error) {
	var lastErr error
	wfaCount := 0
	for _, response := range responses {
		// WPS/WFA は UPnP ではあるが、ポートマッピング用 IGD ではないので候補から外す。
		if isWFAResponse(response) {
			wfaCount++
			continue
		}
		result, err := discoverFromLocation(response.Location, defaultLocationFetcher)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return DiscoveryResult{}, fmt.Errorf("%w: %v", ErrNoGateway, lastErr)
	}
	if len(responses) > 0 && wfaCount == len(responses) {
		return DiscoveryResult{}, fmt.Errorf("%w: %w", ErrNoGateway, errOnlyWFADevices)
	}
	return DiscoveryResult{}, errors.New(noMatchMessage)
}
