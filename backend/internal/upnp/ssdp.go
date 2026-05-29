package upnp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	ssdpAddress    = "239.255.255.250:1900"
	ssdpReadWindow = 800 * time.Millisecond
)

// ssdpSearchTargets は、ポート開放をサポートする一般的なインターネットゲートウェイデバイス（IGD）を検出するために、
// SSDP M-SEARCH で照会する UPnP デバイスタイプおよびサービスタイプの一覧です（優先度順に走査されます）。
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

type ssdpResponse struct {
	Location     string
	ST           string
	USN          string
	SearchTarget string
}

// collectSSDPResponses は、指定された IPv4 インターフェース上で SSDP M-SEARCH 要求を送信し、
// 一定の受信ウィンドウ時間（800ms）の間にルーターなどのUPnPデバイスから返されたすべての SSDP 応答を収集して返します。
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

	sortSSDPResponses(responses)
	return responses, nil
}

// collectSSDPResponsesIPv6 は、指定された IPv6 インターフェース上で SSDP M-SEARCH 要求をマルチキャスト送信し、
// 一定の受信ウィンドウ時間の間に返されたすべての IPv6 SSDP 応答を収集して返します。
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

	sortSSDPResponses(responses)
	return responses, nil
}

func ssdpSearchAddrs(iface discoverInterface) ([]*net.UDPAddr, error) {
	multicastAddr, err := net.ResolveUDPAddr("udp4", ssdpAddress)
	if err != nil {
		return nil, err
	}
	addrs := []*net.UDPAddr{multicastAddr}
	if gateway := firstUsableIPv4(iface.IPNet); gateway != nil && !gateway.Equal(iface.ListenAddr.IP) {
		// multicast が期待した NIC に出ない機器向けに、推定 gateway へ unicast も送る。
		addrs = append(addrs, &net.UDPAddr{IP: gateway, Port: 1900})
	}
	return addrs, nil
}

func sortSSDPResponses(responses []ssdpResponse) {
	sort.SliceStable(responses, func(i, j int) bool {
		return ssdpCandidateScore(responses[i]) > ssdpCandidateScore(responses[j])
	})
}

// ssdpCandidateScore は、得られた SSDP 応答のサービス適合度に応じてスコアを決定します。
// WANIPConnection (v2/v1) が最もポート転送に適合するため、高スコアを獲得して優先的に選択されます。
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

func isWFAResponse(response ssdpResponse) bool {
	text := strings.ToLower(response.ST + " " + response.USN + " " + response.Location)
	return strings.Contains(text, "schemas-wifialliance-org") || strings.Contains(text, "wps_device")
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
