package upnp

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type locationFetcher func(string) ([]byte, error)

type rootDevice struct {
	URLBase string `xml:"URLBase"`
	Device  device `xml:"device"`
}

type device struct {
	ServiceList serviceList `xml:"serviceList"`
	DeviceList  deviceList  `xml:"deviceList"`
}

type deviceList struct {
	Devices []device `xml:"device"`
}

type serviceList struct {
	Services []service `xml:"service"`
}

type service struct {
	ServiceType string `xml:"serviceType"`
	ControlURL  string `xml:"controlURL"`
}

func discoverFromLocation(location string, fetch locationFetcher) (DiscoveryResult, error) {
	if fetch == nil {
		fetch = defaultLocationFetcher
	}
	data, err := fetch(location)
	if err != nil {
		return DiscoveryResult{}, fmt.Errorf("fetch root description %q: %w", location, err)
	}
	return ParseRootDevice(data, location)
}

func defaultLocationFetcher(location string) ([]byte, error) {
	return fetchLocation(location, 5*time.Second)
}

func fallbackLocationFetcher(location string) ([]byte, error) {
	return fetchLocation(location, time.Second)
}

func fetchLocation(location string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(location)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("fetch root description failed: %s", resp.Status)
	}
	return data, nil
}

// ParseRootDevice は、取得した UPnP ルーターデバイス記述（XML）をパースし、最適な WAN ポートマッピング制御サービス（WANIPConnection v2/v1 または WANPPPConnection v1）を優先順位に基づいて自動選択・検証し、その結果を返します。
func ParseRootDevice(data []byte, baseURL string) (DiscoveryResult, error) {
	var root rootDevice
	if err := xml.Unmarshal(data, &root); err != nil {
		return DiscoveryResult{}, fmt.Errorf("parse root device xml: %w", err)
	}

	selected, ok := selectService(servicesFromDevice(root.Device))
	if !ok {
		return DiscoveryResult{}, fmt.Errorf("no supported WAN service found")
	}

	resolveBase := strings.TrimSpace(root.URLBase)
	if resolveBase == "" {
		resolveBase = baseURL
	}
	resolved, err := resolveControlURL(resolveBase, selected.ControlURL)
	if err != nil {
		return DiscoveryResult{}, err
	}

	return DiscoveryResult{
		ServiceType: strings.TrimSpace(selected.ServiceType),
		ControlURL:  resolved,
	}, nil
}

func servicesFromDevice(d device) []service {
	services := append([]service{}, d.ServiceList.Services...)
	for _, child := range d.DeviceList.Devices {
		services = append(services, servicesFromDevice(child)...)
	}
	return services
}

func selectService(services []service) (service, bool) {
	priority := map[string]int{
		"urn:schemas-upnp-org:service:WANIPConnection:2":  3,
		"urn:schemas-upnp-org:service:WANIPConnection:1":  2,
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
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base url %q: %w", baseURL, err)
	}
	if u, err := url.Parse(controlURL); err == nil && u.IsAbs() {
		// description と別ホストの control URL は SSRF 的な挙動になるため拒否する。
		if !sameURLHost(base, u) {
			return "", fmt.Errorf("control url host %q does not match location host %q", u.Host, base.Host)
		}
		return controlURL, nil
	}
	ref, err := url.Parse(controlURL)
	if err != nil {
		return "", fmt.Errorf("parse control url %q: %w", controlURL, err)
	}
	return base.ResolveReference(ref).String(), nil
}

func sameURLHost(a, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.EqualFold(a.Host, b.Host)
}
