package upnp

import (
	"errors"
	"fmt"

	"github.com/clagon/port-mapper/backend/internal/domain"
)

var (
	// ErrNoGateway は、ネットワーク上で適合するUPnPゲートウェイ（ルーター）が発見できなかったことを表すエラーです。
	ErrNoGateway      = domain.ErrNoGateway
	errOnlyWFADevices = errors.New("UPnP discovery found only WPS/WFA devices; no InternetGatewayDevice/WAN service responded")
)

// Discover は、ネットワーク上のすべてのアクティブなネットワークインターフェース（IPv4/IPv6）に対して SSDP M-SEARCH マルチキャスト要求を送信し、
// 最初に応答があり、適合するインターネットゲートウェイ（IGD）を検出・解決して返します。
// SSDP応答がない、またはパースに失敗した場合は、ルーターの既知の制御パスに対するフォールバックプローブ（直接走査）を実行します。
func Discover() (DiscoveryResult, error) {
	ifaces, err := discoverInterfaces()
	if err != nil {
		return DiscoveryResult{}, err
	}

	sawOnlyWFA := false
	for _, iface := range ifaces {
		result, err := discoverFromInterface(iface)
		if err == nil {
			return result, nil
		}
		if errors.Is(err, errOnlyWFADevices) {
			sawOnlyWFA = true
		}
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
		}
	}

	result, err := probeGatewayControlURLs(ifaces)
	if err == nil {
		return result, nil
	}

	result, err = probeGatewayDescriptions(ifaces)
	if err == nil {
		return result, nil
	}
	if sawOnlyWFA {
		return DiscoveryResult{}, fmt.Errorf("%w: %v; fallback probes failed: %v", ErrNoGateway, errOnlyWFADevices, err)
	}
	return DiscoveryResult{}, fmt.Errorf("%w: %v", ErrNoGateway, err)
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
	}

	if lastErr != nil {
		return DiscoveryResult{}, fmt.Errorf("%w: %v", ErrNoGateway, lastErr)
	}
	if len(responses) > 0 && wfaCount == len(responses) {
		return DiscoveryResult{}, fmt.Errorf("%w: %w", ErrNoGateway, errOnlyWFADevices)
	}
	return DiscoveryResult{}, errors.New(noMatchMessage)
}
