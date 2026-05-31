package upnp

import (
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxRootDescriptionBytes = 1 << 20

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
	locationURL, err := parseAllowedUPnPURL(location)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout:       timeout,
		CheckRedirect: validateUPnPRedirect,
	}
	resp, err := client.Get(locationURL.String())
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := readRootDescriptionBody(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("fetch root description failed: %s", resp.Status)
	}
	return data, nil
}

func validateUPnPRedirect(req *http.Request, via []*http.Request) error {
	if len(via) == 0 {
		return nil
	}
	if !sameURLHost(via[0].URL, req.URL) {
		return fmt.Errorf("redirect target host %q does not match location host %q", req.URL.Host, via[0].URL.Host)
	}
	if _, err := parseAllowedUPnPURL(req.URL.String()); err != nil {
		return err
	}
	return nil
}

func readRootDescriptionBody(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxRootDescriptionBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxRootDescriptionBytes {
		return nil, fmt.Errorf("root description exceeds %d bytes", maxRootDescriptionBytes)
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
	resolved, err := resolveControlURL(baseURL, resolveBase, selected.ControlURL)
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

func resolveControlURL(locationURL, baseURL, controlURL string) (string, error) {
	controlURL = strings.TrimSpace(controlURL)
	if controlURL == "" {
		return "", fmt.Errorf("empty control url")
	}
	location, err := parseAllowedUPnPURL(locationURL)
	if err != nil {
		return "", fmt.Errorf("invalid location url: %w", err)
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base url %q: %w", baseURL, err)
	}
	if base.IsAbs() && !sameURLHost(location, base) {
		return "", fmt.Errorf("urlbase host %q does not match location host %q", base.Host, location.Host)
	}
	ref, err := url.Parse(controlURL)
	if err != nil {
		return "", fmt.Errorf("parse control url %q: %w", controlURL, err)
	}
	resolved := base.ResolveReference(ref)
	// description と別ホストの URLBase/controlURL は SSRF 的な挙動になるため拒否する。
	if !sameURLHost(location, resolved) {
		return "", fmt.Errorf("control url host %q does not match location host %q", resolved.Host, location.Host)
	}
	if _, err := parseAllowedUPnPURL(resolved.String()); err != nil {
		return "", fmt.Errorf("invalid control url: %w", err)
	}
	return resolved.String(), nil
}

func sameURLHost(a, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	return strings.EqualFold(a.Hostname(), b.Hostname())
}

func parseAllowedUPnPURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("parse url %q: %w", rawURL, err)
	}
	if u.Scheme != "http" {
		return nil, fmt.Errorf("unsupported url scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("missing url host")
	}
	host := u.Hostname()
	ipHost := stripIPv6Zone(host)
	ip := net.ParseIP(ipHost)
	if ip == nil {
		return nil, fmt.Errorf("url host %q is not an IP address", host)
	}
	if !isAllowedUPnPIP(ip) {
		return nil, fmt.Errorf("url host %q is not an allowed local UPnP address", host)
	}
	return u, nil
}

func isAllowedUPnPIP(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() {
		return false
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4.IsPrivate() || ip4.IsLinkLocalUnicast()
	}
	return ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

func stripIPv6Zone(host string) string {
	addr, _, ok := strings.Cut(host, "%")
	if ok {
		return addr
	}
	return host
}
