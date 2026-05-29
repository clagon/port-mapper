package upnp

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// fallbackGatewayLocations は、SSDPマルチキャストが動作しない、またはルーターからの応答が得られない場合に備え、
// ルーターの想定されるゲートウェイIPアドレス（例: 192.168.1.1等）とよく使われるUPnPポート（5000, 49152, 1900）および
// 一般的なデバイス記述XMLパス（/rootDesc.xml等）の全組み合わせから構成される、探索用URL候補の一覧を生成します。
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

// probeGatewayDescriptions は、fallbackGatewayLocations で生成されたすべてのデバイス記述XMLのURL候補に対して、
// 並行して HTTP GET 要求を送信し、最も早く解決された有効なルーターサービス探索結果（DiscoveryResult）を返します。
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

// probeGatewayControlURLs は、デバイス記述XMLを介さず、想定される制御エンドポイント（Control URL）に直接接続し、
// グローバルIP取得（GetExternalIPAddress）の SOAP 要求を送信することで、動作する有効な制御エンドポイントを並行してプローブします。
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

// fallbackControlCandidates は、既知の主要ルーター製コントロールパス（/upnp/control/WANIPConn1等）と
// よく使われるポート等のマトリクスから、直接通信を試みるための DiscoveryResult のリストを生成します。
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
