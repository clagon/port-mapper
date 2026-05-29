// Package service は Porto のコアビジネスロジックであるルーター探索状態の管理、ポートマッピング要求の検証・実行、設定の永続化連携を実装します。
package service

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"sync"

	"github.com/clagon/port-mapper/backend/internal/application"
	"github.com/clagon/port-mapper/backend/internal/config"
)

// SettingsStore は、ユーザーが変更可能な環境設定情報を永続化（保存）するためのインターフェースです。
type SettingsStore interface {
	// Save は提供された設定オブジェクトを永続的ストレージに保存します。
	Save(config.Config) error
}

// Status は、現在アクティブなゲートウェイ（ルーター）探索結果と、登録されているポートマッピング一覧などのアプリケーション状態を表す構造体です。
type Status struct {
	Discovered  bool                      `json:"discovered"`
	ServiceType string                    `json:"service_type,omitempty"`
	ControlURL  string                    `json:"control_url,omitempty"`
	ExternalIP  string                    `json:"external_ip,omitempty"`
	LocalIP     string                    `json:"local_ip,omitempty"`
	Ports       []application.PortMapping `json:"ports"`
}

// Options は、Service の生成時に注入される依存関係および設定を指定するための構造体です。
type Options struct {
	ConfigPath    string
	Config        config.Config
	Logger        *slog.Logger
	SettingsStore SettingsStore

	Discovery         application.DiscoveryClient
	PortMapperFactory application.PortMapperFactory
}

// Service は、Porto のポートマッピング機能の中核であり、ルーター探索、ポートの開閉処理、メモリ上での状態の同期を担当するドメインサービスです。
type Service struct {
	mu                sync.RWMutex
	cfg               config.Config
	configPath        string
	settingsStore     SettingsStore
	discovery         application.DiscoveryClient
	portMapperFactory application.PortMapperFactory
	gateway           *application.DiscoveryResult
	externalIP        string
	localIP           string
	ports             []application.PortMapping
	logger            *slog.Logger
}

// service 内で gateway 未選択を表すエラー。UPnP discovery 自体の失敗は application.ErrNoGateway を使う。
var errNoGateway = errors.New("no UPnP gateway discovered")

// New は、指定された Options を使用して Service インスタンスを新しく生成します。
func New(opts Options) *Service {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	cfg := opts.Config.WithDefaults()
	if opts.ConfigPath == "" {
		opts.ConfigPath = config.DefaultPath()
	}
	if opts.SettingsStore == nil {
		opts.SettingsStore = config.FileStore{Path: opts.ConfigPath}
	}
	return &Service{
		cfg:               cfg,
		configPath:        opts.ConfigPath,
		settingsStore:     opts.SettingsStore,
		discovery:         opts.Discovery,
		portMapperFactory: opts.PortMapperFactory,
		logger:            logger,
	}
}

// Settings は、現在の環境設定のコピーを安全に返します。
func (s *Service) Settings() config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.WithDefaults()
}

// UpdateSettings は、環境設定を検証し、設定永続化ストアに保存した後にメモリ上のキャッシュ設定値を更新します。
func (s *Service) UpdateSettings(next config.Config) (config.Config, error) {
	next = next.WithDefaults()
	if err := config.ValidateLocalListenAddr(next.ListenAddr); err != nil {
		return config.Config{}, err
	}
	if err := s.settingsStore.Save(next); err != nil {
		return config.Config{}, err
	}

	s.mu.Lock()
	s.cfg = next
	s.mu.Unlock()

	if s.logger != nil {
		s.logger.Info("settings saved",
			"config_path", s.configPath,
			"listen_addr", next.ListenAddr,
			"auto_discover", boolValue(next.AutoDiscover),
		)
	}
	return next, nil
}

// Status は、現在検知されているルーター探索状態や登録されているポートマッピング一覧を安全に構築して返します。
func (s *Service) Status() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := Status{
		Discovered: s.gateway != nil,
		Ports:      append([]application.PortMapping{}, s.ports...),
	}
	if s.gateway != nil {
		resp.ServiceType = s.gateway.ServiceType
		resp.ControlURL = s.gateway.ControlURL
		resp.ExternalIP = s.externalIP
		resp.LocalIP = s.localIP
	}
	return resp
}

// Discover は、ネットワーク上のUPnPデバイス探索を実行し、ルーターの発見、グローバルIPおよびローカルIPの自動特定、ルーター既存転送ルールの同期を行います。
func (s *Service) Discover() (Status, error) {
	if s.discovery == nil {
		return s.Status(), errors.New("discovery client not configured")
	}
	result, err := s.discovery.Discover()
	if err != nil {
		// discovery 未検出は UI 上の通常状態として扱い、現在の status を返す。
		if errors.Is(err, application.ErrNoGateway) {
			if s.logger != nil {
				s.logger.Info("router not discovered", "reason", err.Error())
			}
			return s.Status(), nil
		}
		return Status{}, err
	}

	var mapper application.PortMapper
	if s.portMapperFactory != nil {
		mapper = s.portMapperFactory(result)
	}
	var externalIP string
	if mapper != nil {
		if ip, err := mapper.GetExternalIPAddress(); err == nil {
			externalIP = ip
		} else if s.logger != nil {
			s.logger.Warn("external IP lookup failed", "error", err)
		}
	}

	// 自身のローカルIPアドレスの特定
	var localIP string
	if u, err := url.Parse(result.ControlURL); err == nil {
		if conn, err := net.Dial("udp", u.Host); err == nil {
			if localAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok && !localAddr.IP.IsUnspecified() {
				localIP = localAddr.IP.String()
			}
			conn.Close()
		}
	}
	if localIP == "" {
		if ip, err := s.fallbackLocalIP(); err == nil {
			localIP = ip
		}
	}

	s.mu.Lock()
	resultCopy := result
	s.gateway = &resultCopy
	s.externalIP = externalIP
	s.localIP = localIP
	s.mu.Unlock()

	// ルーターからのポートマッピング自動同期
	if mapper != nil && localIP != "" {
		s.syncActivePorts(mapper, localIP)
	}

	if s.logger != nil {
		s.logger.Info("router discovered",
			"service_type", result.ServiceType,
			"control_url", result.ControlURL,
			"external_ip", externalIP,
			"local_ip", localIP,
		)
	}
	return s.Status(), nil
}

// OpenPort は、提供されたポートマッピングの入力値を検証し、接続されたルーターへポート開放ルールを追加した後に、メモリ上のマッピング一覧を更新します。
func (s *Service) OpenPort(mapping application.PortMapping) (Status, error) {
	resolvedIP, err := s.resolveInternalIP(mapping.InternalIP)
	if err != nil {
		return Status{}, fmt.Errorf("failed to resolve internal IP: %w", err)
	}
	mapping.InternalIP = resolvedIP

	if err := application.ValidatePortMapping(mapping); err != nil {
		return Status{}, err
	}
	mapper, err := s.currentPortMapper()
	if err != nil {
		return Status{}, err
	}
	if err := mapper.AddPortMapping(mapping); err != nil {
		return Status{}, err
	}

	s.mu.Lock()
	s.ports = upsertMapping(s.ports, mapping)
	s.mu.Unlock()

	if s.logger != nil {
		s.logger.Info("port mapping opened",
			"protocol", normalizeProtocol(mapping.Protocol),
			"external_port", mapping.ExternalPort,
			"internal_ip", mapping.InternalIP,
			"internal_port", mapping.InternalPort,
		)
	}
	return s.Status(), nil
}

// ClosePort は、提供されたポートマッピング削除リクエスト（プロトコルおよび外部ポート）を検証し、接続されたルーターから該当転送ルールを削除してマッピング一覧から除外します。
func (s *Service) ClosePort(mapping application.PortMapping) (Status, error) {
	if err := validateDeleteRequest(mapping); err != nil {
		return Status{}, err
	}
	mapper, err := s.currentPortMapper()
	if err != nil {
		return Status{}, err
	}
	if err := mapper.DeletePortMapping(mapping.Protocol, mapping.ExternalPort); err != nil {
		return Status{}, err
	}

	s.mu.Lock()
	s.ports = removeMapping(s.ports, mapping.Protocol, mapping.ExternalPort)
	s.mu.Unlock()

	if s.logger != nil {
		s.logger.Info("port mapping closed",
			"protocol", normalizeProtocol(mapping.Protocol),
			"external_port", mapping.ExternalPort,
		)
	}
	return s.Status(), nil
}

func (s *Service) currentPortMapper() (application.PortMapper, error) {
	s.mu.RLock()
	gateway := s.gateway
	s.mu.RUnlock()
	if gateway == nil {
		return nil, errNoGateway
	}
	if s.portMapperFactory == nil {
		return nil, fmt.Errorf("port mapper factory is not configured")
	}
	mapper := s.portMapperFactory(*gateway)
	if mapper == nil {
		return nil, fmt.Errorf("port mapper factory returned nil")
	}
	return mapper, nil
}

func upsertMapping(existing []application.PortMapping, next application.PortMapping) []application.PortMapping {
	for i, current := range existing {
		if sameMappingKey(current, next) {
			existing[i] = next
			return existing
		}
	}
	return append(existing, next)
}

func removeMapping(existing []application.PortMapping, protocol string, externalPort int) []application.PortMapping {
	filtered := existing[:0]
	for _, current := range existing {
		if sameMappingIdentity(current.Protocol, current.ExternalPort, protocol, externalPort) {
			continue
		}
		filtered = append(filtered, current)
	}
	return filtered
}

func sameMappingKey(a, b application.PortMapping) bool {
	return sameMappingIdentity(a.Protocol, a.ExternalPort, b.Protocol, b.ExternalPort)
}

func sameMappingIdentity(aProtocol string, aPort int, bProtocol string, bPort int) bool {
	return normalizeProtocol(aProtocol) == normalizeProtocol(bProtocol) && aPort == bPort
}

func normalizeProtocol(protocol string) string {
	return strings.ToUpper(strings.TrimSpace(protocol))
}

func validateDeleteRequest(mapping application.PortMapping) error {
	if err := validateDeleteProtocol(mapping.Protocol); err != nil {
		return err
	}
	if mapping.ExternalPort < 1 || mapping.ExternalPort > 65535 {
		return fmt.Errorf("external port %d out of range: must be 1-65535", mapping.ExternalPort)
	}
	return nil
}

func validateDeleteProtocol(protocol string) error {
	switch normalizeProtocol(protocol) {
	case "TCP", "UDP":
		return nil
	default:
		return fmt.Errorf("invalid protocol %q: must be TCP or UDP", protocol)
	}
}

func boolValue(v *bool) bool {
	return v != nil && *v
}

func (s *Service) resolveInternalIP(providedIP string) (string, error) {
	if ip := strings.TrimSpace(providedIP); ip != "" {
		return ip, nil
	}

	s.mu.RLock()
	gateway := s.gateway
	s.mu.RUnlock()

	if gateway == nil {
		return "", errNoGateway
	}

	u, err := url.Parse(gateway.ControlURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse gateway control URL: %w", err)
	}

	// ゲートウェイのホストに向けて一時的にUDPソケットを開くことでローカルIPを特定
	conn, err := net.Dial("udp", u.Host)
	if err != nil {
		// 失敗時はフォールバックとして最初の非ループバックIPv4を探す
		return s.fallbackLocalIP()
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || localAddr.IP.IsUnspecified() {
		return s.fallbackLocalIP()
	}

	return localAddr.IP.String(), nil
}

func (s *Service) fallbackLocalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
				return ip.String(), nil
			}
		}
	}
	return "", fmt.Errorf("no suitable local IP address found")
}

func (s *Service) syncActivePorts(mapper application.PortMapper, localIP string) {
	if mapper == nil || localIP == "" {
		return
	}

	var syncedPorts []application.PortMapping
	for i := 0; ; i++ {
		entry, err := mapper.GetGenericPortMappingEntry(i)
		if err != nil {
			// インデックス範囲外など、ルーターがこれ以上エントリーを持っていない場合は走査を終了
			if s.logger != nil {
				s.logger.Debug("finished fetching port mapping entries from router", "index", i, "error", err)
			}
			break
		}

		// このローカルPCのIPアドレス宛の転送ルールのみを抽出
		if entry.InternalIP == localIP {
			syncedPorts = append(syncedPorts, entry)
		}

		// 安全のための上限 (無限ループ防止)
		if i >= 256 {
			break
		}
	}

	s.mu.Lock()
	for _, entry := range syncedPorts {
		s.ports = upsertMapping(s.ports, entry)
	}
	s.mu.Unlock()
}
