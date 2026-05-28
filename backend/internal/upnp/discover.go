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
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	ssdpAddress    = "239.255.255.250:1900"
	ssdpReadWindow = 800 * time.Millisecond
)

var ssdpSearchTargets = []string{
	"urn:schemas-upnp-org:device:InternetGatewayDevice:2",
	"urn:schemas-upnp-org:device:InternetGatewayDevice:1",
	"urn:schemas-upnp-org:device:WANDevice:2",
	"urn:schemas-upnp-org:device:WANDevice:1",
	"urn:schemas-upnp-org:device:WANConnectionDevice:2",
	"urn:schemas-upnp-org:device:WANConnectionDevice:1",
	"urn:schemas-upnp-org:service:WANIPConnection:2",
	"urn:schemas-upnp-org:service:WANIPConnection:1",
	"urn:schemas-upnp-org:service:WANPPPConnection:1",
	"upnp:rootdevice",
	"ssdp:all",
}

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

type ssdpResponse struct {
	Location     string
	ST           string
	USN          string
	SearchTarget string
}

type discoverInterface struct {
	ListenAddr *net.UDPAddr
	IPNet      *net.IPNet
	Interface  *net.Interface
}

type discoverIPv6Interface struct {
	ListenAddr *net.UDPAddr
	Interface  *net.Interface
}

var (
	errOnlyWFADevices = errors.New("UPnP discovery found only WPS/WFA devices; no InternetGatewayDevice/WAN service responded")
	errNoWANService   = errors.New("no UPnP InternetGatewayDevice/WAN service discovered")
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
		return DiscoveryResult{}, fmt.Errorf("%w; fallback probes failed: %v", errOnlyWFADevices, lastErr)
	}
	if lastErr != nil {
		return DiscoveryResult{}, fmt.Errorf("%w: %v", errNoWANService, lastErr)
	}
	return DiscoveryResult{}, errNoWANService
}

func discoverFromInterface(iface discoverInterface) (DiscoveryResult, error) {
	responses, err := collectSSDPResponses(iface)
	if err != nil {
		return DiscoveryResult{}, err
	}

	var lastErr error
	wfaCount := 0
	for _, response := range responses {
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
		return DiscoveryResult{}, fmt.Errorf("%w: %v", errNoWANService, lastErr)
	}
	if len(responses) > 0 && wfaCount == len(responses) {
		return DiscoveryResult{}, errOnlyWFADevices
	}
	return DiscoveryResult{}, errors.New("no matching SSDP responses")
}

func isWFAResponse(response ssdpResponse) bool {
	text := strings.ToLower(response.ST + " " + response.USN + " " + response.Location)
	return strings.Contains(text, "schemas-wifialliance-org") || strings.Contains(text, "wps_device")
}

func discoverFromIPv6Interface(iface discoverIPv6Interface) (DiscoveryResult, error) {
	responses, err := collectSSDPResponsesIPv6(iface)
	if err != nil {
		return DiscoveryResult{}, err
	}

	var lastErr error
	wfaCount := 0
	for _, response := range responses {
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
		return DiscoveryResult{}, fmt.Errorf("%w: %v", errNoWANService, lastErr)
	}
	if len(responses) > 0 && wfaCount == len(responses) {
		return DiscoveryResult{}, errOnlyWFADevices
	}
	return DiscoveryResult{}, errors.New("no matching IPv6 SSDP responses")
}

func collectSSDPResponses(iface discoverInterface) ([]ssdpResponse, error) {
	conn, err := net.ListenUDP("udp4", iface.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("listen for SSDP: %w", err)
	}
	defer conn.Close()

	packetConn := ipv4.NewPacketConn(conn)
	if iface.Interface != nil {
		if err := packetConn.SetMulticastInterface(iface.Interface); err != nil {
			return nil, fmt.Errorf("set SSDP multicast interface %s: %w", iface.Interface.Name, err)
		}
	}
	if err := packetConn.SetMulticastTTL(2); err != nil {
		return nil, fmt.Errorf("set SSDP multicast ttl: %w", err)
	}

	searchAddrs, err := ssdpSearchAddrs(iface)
	if err != nil {
		return nil, fmt.Errorf("resolve SSDP address: %w", err)
	}

	buf := make([]byte, 65535)
	seenResponses := map[string]struct{}{}
	var responses []ssdpResponse
	for _, target := range ssdpSearchTargets {
		search := buildMSearch(target)
		for _, addr := range searchAddrs {
			if _, err := conn.WriteToUDP([]byte(search), addr); err != nil {
				return nil, fmt.Errorf("send SSDP search for %s to %s: %w", target, addr, err)
			}
		}

		if err := conn.SetReadDeadline(time.Now().Add(ssdpReadWindow)); err != nil {
			return nil, fmt.Errorf("set SSDP read deadline: %w", err)
		}
		for {
			n, _, err := conn.ReadFrom(buf)
			if err != nil {
				if isTimeout(err) {
					break
				}
				return nil, fmt.Errorf("read SSDP response: %w", err)
			}

			response, err := parseSSDPResponse(buf[:n])
			if err != nil {
				continue
			}
			response.SearchTarget = target
			key := response.Location + "\x00" + response.ST + "\x00" + response.USN
			if _, ok := seenResponses[key]; ok {
				continue
			}
			seenResponses[key] = struct{}{}
			responses = append(responses, response)
		}
	}

	sort.SliceStable(responses, func(i, j int) bool {
		return ssdpCandidateScore(responses[i]) > ssdpCandidateScore(responses[j])
	})

	return responses, nil
}

func collectSSDPResponsesIPv6(iface discoverIPv6Interface) ([]ssdpResponse, error) {
	conn, err := net.ListenUDP("udp6", iface.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("listen for IPv6 SSDP: %w", err)
	}
	defer conn.Close()

	packetConn := ipv6.NewPacketConn(conn)
	if iface.Interface != nil {
		if err := packetConn.SetMulticastInterface(iface.Interface); err != nil {
			return nil, fmt.Errorf("set IPv6 SSDP multicast interface %s: %w", iface.Interface.Name, err)
		}
	}
	if err := packetConn.SetMulticastHopLimit(2); err != nil {
		return nil, fmt.Errorf("set IPv6 SSDP multicast hop limit: %w", err)
	}

	addr := &net.UDPAddr{IP: net.ParseIP("ff02::c"), Port: 1900}
	if iface.Interface != nil {
		addr.Zone = iface.Interface.Name
	}

	buf := make([]byte, 65535)
	seenResponses := map[string]struct{}{}
	var responses []ssdpResponse
	for _, target := range ssdpSearchTargets {
		search := buildMSearchWithHost("[FF02::C]:1900", target)
		if _, err := conn.WriteToUDP([]byte(search), addr); err != nil {
			return nil, fmt.Errorf("send IPv6 SSDP search for %s to %s: %w", target, addr, err)
		}

		if err := conn.SetReadDeadline(time.Now().Add(ssdpReadWindow)); err != nil {
			return nil, fmt.Errorf("set IPv6 SSDP read deadline: %w", err)
		}
		for {
			n, _, err := conn.ReadFrom(buf)
			if err != nil {
				if isTimeout(err) {
					break
				}
				return nil, fmt.Errorf("read IPv6 SSDP response: %w", err)
			}

			response, err := parseSSDPResponse(buf[:n])
			if err != nil {
				continue
			}
			response.SearchTarget = target
			key := response.Location + "\x00" + response.ST + "\x00" + response.USN
			if _, ok := seenResponses[key]; ok {
				continue
			}
			seenResponses[key] = struct{}{}
			responses = append(responses, response)
		}
	}

	sort.SliceStable(responses, func(i, j int) bool {
		return ssdpCandidateScore(responses[i]) > ssdpCandidateScore(responses[j])
	})

	return responses, nil
}

func ssdpSearchAddrs(iface discoverInterface) ([]*net.UDPAddr, error) {
	multicastAddr, err := net.ResolveUDPAddr("udp4", ssdpAddress)
	if err != nil {
		return nil, err
	}
	addrs := []*net.UDPAddr{multicastAddr}
	if gateway := firstUsableIPv4(iface.IPNet); gateway != nil && !gateway.Equal(iface.ListenAddr.IP) {
		addrs = append(addrs, &net.UDPAddr{IP: gateway, Port: 1900})
	}
	return addrs, nil
}

func ssdpCandidateScore(response ssdpResponse) int {
	text := strings.ToLower(response.SearchTarget + " " + response.ST + " " + response.USN + " " + response.Location)
	switch {
	case strings.Contains(text, "wanipconnection:2"):
		return 60
	case strings.Contains(text, "wanipconnection:1"):
		return 50
	case strings.Contains(text, "wanpppconnection:1"):
		return 40
	case strings.Contains(text, "internetgatewaydevice:2"):
		return 30
	case strings.Contains(text, "internetgatewaydevice:1"):
		return 20
	case strings.Contains(text, "upnp:rootdevice"):
		return 10
	default:
		return 0
	}
}

func discoverInterfaces() ([]discoverInterface, error) {
	var ifaces []discoverInterface
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list network interfaces: %w", err)
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		ifaceAddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, ifaceAddr := range ifaceAddrs {
			ip, ipNet := ipv4NetFromAddr(ifaceAddr)
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() {
				continue
			}
			if first := firstUsableIPv4(ipNet); first != nil && first.Equal(ip) {
				continue
			}
			ifaces = append(ifaces, discoverInterface{
				ListenAddr: &net.UDPAddr{IP: ip, Port: 0},
				IPNet:      ipNet,
				Interface:  &iface,
			})
		}
	}
	if len(ifaces) == 0 {
		ifaces = append(ifaces, discoverInterface{
			ListenAddr: &net.UDPAddr{IP: net.IPv4zero, Port: 0},
		})
	}
	return ifaces, nil
}

func discoverIPv6Interfaces() ([]discoverIPv6Interface, error) {
	var ifaces []discoverIPv6Interface
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list network interfaces: %w", err)
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		ifaceAddrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		hasIPv6 := false
		for _, ifaceAddr := range ifaceAddrs {
			ip := ipv6FromAddr(ifaceAddr)
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
				continue
			}
			hasIPv6 = true
			break
		}
		if hasIPv6 {
			ifaces = append(ifaces, discoverIPv6Interface{
				ListenAddr: &net.UDPAddr{IP: net.IPv6zero, Port: 0},
				Interface:  &iface,
			})
		}
	}
	return ifaces, nil
}

func ipv4NetFromAddr(addr net.Addr) (net.IP, *net.IPNet) {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP.To4(), v
	case *net.IPAddr:
		return v.IP.To4(), nil
	default:
		return nil, nil
	}
}

func ipv6FromAddr(addr net.Addr) net.IP {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	default:
		return nil
	}
	if ip.To4() != nil {
		return nil
	}
	return ip
}

func fallbackGatewayLocations(ifaces []discoverInterface) []string {
	paths := []string{
		"/rootDesc.xml",
		"/rootdesc.xml",
		"/igd.xml",
		"/IGD.xml",
		"/igddesc.xml",
		"/InternetGatewayDevice.xml",
		"/upnp/rootDesc.xml",
		"/upnp/IGD.xml",
	}
	ports := []int{5000, 49152, 1900}

	seenHosts := map[string]struct{}{}
	var locations []string
	for _, iface := range ifaces {
		gateway := firstUsableIPv4(iface.IPNet)
		if gateway == nil || gateway.Equal(iface.ListenAddr.IP) {
			continue
		}
		host := gateway.String()
		if _, ok := seenHosts[host]; ok {
			continue
		}
		seenHosts[host] = struct{}{}
		for _, port := range ports {
			for _, path := range paths {
				locations = append(locations, fmt.Sprintf("http://%s:%d%s", host, port, path))
			}
		}
	}
	return locations
}

func probeGatewayDescriptions(ifaces []discoverInterface) (DiscoveryResult, error) {
	locations := fallbackGatewayLocations(ifaces)
	if len(locations) == 0 {
		return DiscoveryResult{}, errors.New("no fallback description candidates")
	}

	resultCh := make(chan DiscoveryResult, 1)
	var wg sync.WaitGroup
	for _, location := range locations {
		location := location
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := discoverFromLocation(location, fallbackLocationFetcher)
			if err != nil {
				return
			}
			select {
			case resultCh <- result:
			default:
			}
		}()
	}

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case result := <-resultCh:
		return result, nil
	case <-doneCh:
		return DiscoveryResult{}, errors.New("no fallback description responded")
	}
}

func probeGatewayControlURLs(ifaces []discoverInterface) (DiscoveryResult, error) {
	candidates := fallbackControlCandidates(ifaces)
	if len(candidates) == 0 {
		return DiscoveryResult{}, errors.New("no fallback control url candidates")
	}

	resultCh := make(chan DiscoveryResult, 1)
	var wg sync.WaitGroup
	for _, candidate := range candidates {
		candidate := candidate
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &SOAPClient{
				Endpoint:    candidate.ControlURL,
				ServiceType: candidate.ServiceType,
				HTTPClient:  &http.Client{Timeout: 800 * time.Millisecond},
			}
			if _, err := client.GetExternalIPAddress(); err != nil {
				return
			}
			select {
			case resultCh <- candidate:
			default:
			}
		}()
	}

	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(doneCh)
	}()

	select {
	case result := <-resultCh:
		return result, nil
	case <-doneCh:
		return DiscoveryResult{}, errors.New("no fallback control url responded")
	}
}

func fallbackControlCandidates(ifaces []discoverInterface) []DiscoveryResult {
	serviceTypes := []string{
		"urn:schemas-upnp-org:service:WANIPConnection:2",
		"urn:schemas-upnp-org:service:WANIPConnection:1",
		"urn:schemas-upnp-org:service:WANPPPConnection:1",
	}
	paths := []string{
		"/upnp/control/WANIPConn1",
		"/upnp/control/WANIPConn",
		"/upnp/control/WANIPConnection",
		"/upnp/control/WANPPPConn1",
		"/upnp/control/WANPPPConn",
		"/ctl/IPConn",
		"/ctl/IPConn1",
		"/ctl/PPPConn",
	}
	ports := []int{5000, 49152, 1900}

	seenHosts := map[string]struct{}{}
	var candidates []DiscoveryResult
	for _, iface := range ifaces {
		gateway := firstUsableIPv4(iface.IPNet)
		if gateway == nil || gateway.Equal(iface.ListenAddr.IP) {
			continue
		}
		host := gateway.String()
		if _, ok := seenHosts[host]; ok {
			continue
		}
		seenHosts[host] = struct{}{}
		for _, port := range ports {
			for _, path := range paths {
				controlURL := fmt.Sprintf("http://%s:%d%s", host, port, path)
				for _, serviceType := range serviceTypes {
					candidates = append(candidates, DiscoveryResult{
						ServiceType: serviceType,
						ControlURL:  controlURL,
					})
				}
			}
		}
	}
	return candidates
}

func firstUsableIPv4(ipNet *net.IPNet) net.IP {
	if ipNet == nil {
		return nil
	}
	ip := ipNet.IP.To4()
	if ip == nil || len(ipNet.Mask) != net.IPv4len {
		return nil
	}
	network := ip.Mask(ipNet.Mask).To4()
	if network == nil {
		return nil
	}
	return net.IPv4(network[0], network[1], network[2], network[3]+1).To4()
}

func buildMSearch(searchTarget string) string {
	return buildMSearchWithHost("239.255.255.250:1900", searchTarget)
}

func buildMSearchWithHost(host, searchTarget string) string {
	return strings.Join([]string{
		"M-SEARCH * HTTP/1.1",
		"HOST: " + host,
		`MAN: "ssdp:discover"`,
		"MX: 2",
		"ST: " + searchTarget,
		"USER-AGENT: Windows/10 UPnP/1.1 port-mapper/1.0",
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

func parseSSDPResponse(data []byte) (ssdpResponse, error) {
	reader := textproto.NewReader(bufio.NewReader(bytes.NewReader(data)))
	statusLine, err := reader.ReadLine()
	if err != nil {
		return ssdpResponse{}, err
	}
	statusLine = strings.ToUpper(strings.TrimSpace(statusLine))
	if !strings.HasPrefix(statusLine, "HTTP/1.1 200") && !strings.HasPrefix(statusLine, "HTTP/1.0 200") {
		return ssdpResponse{}, fmt.Errorf("unexpected SSDP response status: %s", statusLine)
	}
	headers, err := reader.ReadMIMEHeader()
	if err != nil {
		return ssdpResponse{}, err
	}
	location := strings.TrimSpace(headers.Get("LOCATION"))
	if location == "" {
		return ssdpResponse{}, fmt.Errorf("ssdp response missing location")
	}
	return ssdpResponse{
		Location: location,
		ST:       strings.TrimSpace(headers.Get("ST")),
		USN:      strings.TrimSpace(headers.Get("USN")),
	}, nil
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
