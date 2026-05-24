package upnp

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"
)

const (
	ssdpAddress      = "239.255.255.250:1900"
	ssdpTimeout      = 3 * time.Second
)

var ssdpSearchTargets = []string{
	"urn:schemas-upnp-org:device:InternetGatewayDevice:1",
	"upnp:rootdevice",
	"ssdp:all",
}

var ErrNoGateway = errors.New("no UPnP gateway discovered")

type locationFetcher func(string) ([]byte, error)

type rootDevice struct {
	Device device `xml:"device"`
}

type device struct {
	DeviceList  []device   `xml:"deviceList>device"`
	ServiceList serviceList `xml:"serviceList"`
}

type serviceList struct {
	Services []service `xml:"service"`
}

type service struct {
	ServiceType string `xml:"serviceType"`
	ControlURL  string `xml:"controlURL"`
}

// Discover sends an SSDP M-SEARCH and returns the first supported gateway it can resolve.
func Discover() (DiscoveryResult, error) {
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return DiscoveryResult{}, fmt.Errorf("listen for SSDP: %w", err)
	}
	defer conn.Close()

	addr, err := net.ResolveUDPAddr("udp4", ssdpAddress)
	if err != nil {
		return DiscoveryResult{}, fmt.Errorf("resolve SSDP address: %w", err)
	}

	buf := make([]byte, 65535)
	for _, target := range ssdpSearchTargets {
		if err := conn.SetDeadline(time.Now().Add(ssdpTimeout)); err != nil {
			return DiscoveryResult{}, fmt.Errorf("set SSDP deadline: %w", err)
		}
		if _, err := conn.WriteTo([]byte(buildSSDPSearchRequest(target)), addr); err != nil {
			return DiscoveryResult{}, fmt.Errorf("send SSDP search: %w", err)
		}

		for {
			n, _, err := conn.ReadFrom(buf)
			if err != nil {
				if isTimeout(err) {
					break
				}
				return DiscoveryResult{}, fmt.Errorf("read SSDP response: %w", err)
			}

			location, err := parseSSDPResponse(buf[:n])
			if err != nil {
				continue
			}

			result, err := discoverFromLocation(location, defaultLocationFetcher)
			if err == nil {
				return result, nil
			}
		}
	}

	return DiscoveryResult{}, ErrNoGateway
}

func buildSSDPSearchRequest(target string) string {
	return strings.Join([]string{
		"M-SEARCH * HTTP/1.1",
		"HOST: 239.255.255.250:1900",
		`MAN: "ssdp:discover"`,
		"MX: 1",
		"ST: " + target,
		"",
		"",
	}, "\r\n")
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
	client := &http.Client{Timeout: 5 * time.Second}
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

func parseSSDPResponse(data []byte) (string, error) {
	reader := textproto.NewReader(bufio.NewReader(bytes.NewReader(data)))
	statusLine, err := reader.ReadLine()
	if err != nil {
		return "", err
	}
	statusLine = strings.ToUpper(strings.TrimSpace(statusLine))
	if !strings.HasPrefix(statusLine, "HTTP/1.1 200") && !strings.HasPrefix(statusLine, "HTTP/1.0 200") {
		return "", fmt.Errorf("unexpected SSDP response status: %s", statusLine)
	}
	headers, err := reader.ReadMIMEHeader()
	if err != nil {
		return "", err
	}
	location := strings.TrimSpace(headers.Get("LOCATION"))
	if location == "" {
		return "", fmt.Errorf("ssdp response missing location")
	}
	return location, nil
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// ParseRootDevice parses a UPnP root device description and selects the best WAN service.
func ParseRootDevice(data []byte, baseURL string) (DiscoveryResult, error) {
	var root rootDevice
	if err := xml.Unmarshal(data, &root); err != nil {
		return DiscoveryResult{}, fmt.Errorf("parse root device xml: %w", err)
	}

	selected, ok := selectService(collectServices(root.Device))
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

func collectServices(dev device) []service {
	services := append([]service{}, dev.ServiceList.Services...)
	for _, child := range dev.DeviceList {
		services = append(services, collectServices(child)...)
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
