package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/clagon/port-mapper/backend/internal/config"
	"github.com/clagon/port-mapper/backend/internal/upnp"
)

type DiscoveryClient interface {
	Discover() (upnp.DiscoveryResult, error)
}

type PortMapper interface {
	GetExternalIPAddress() (string, error)
	AddPortMapping(upnp.PortMapping) error
	DeletePortMapping(protocol string, externalPort int) error
}

type PortMapperFactory func(upnp.DiscoveryResult) PortMapper

type serviceOptions struct {
	configPath        string
	cfg               config.Config
	discovery         DiscoveryClient
	portMapperFactory PortMapperFactory
	logger            *slog.Logger
}

type service struct {
	mu                sync.RWMutex
	cfg               config.Config
	configPath        string
	discovery         DiscoveryClient
	portMapperFactory PortMapperFactory
	gateway           *upnp.DiscoveryResult
	externalIP        string
	ports             []upnp.PortMapping
	logger            *slog.Logger
}

// service 内で gateway 未選択を表すエラー。UPnP discovery 自体の失敗は upnp.ErrNoGateway を使う。
var errNoGateway = errors.New("no UPnP gateway discovered")

func newService(opts serviceOptions) *service {
	logger := opts.logger
	if logger == nil {
		logger = slog.Default()
	}
	cfg := opts.cfg.WithDefaults()
	if opts.configPath == "" {
		opts.configPath = config.DefaultPath()
	}
	if opts.discovery == nil {
		opts.discovery = defaultDiscoveryClient{}
	}
	if opts.portMapperFactory == nil {
		opts.portMapperFactory = defaultPortMapperFactory
	}
	return &service{
		cfg:               cfg,
		configPath:        opts.configPath,
		discovery:         opts.discovery,
		portMapperFactory: opts.portMapperFactory,
		logger:            logger,
	}
}

func defaultPortMapperFactory(result upnp.DiscoveryResult) PortMapper {
	return &upnp.SOAPClient{
		Endpoint:    result.ControlURL,
		ServiceType: result.ServiceType,
		HTTPClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

type defaultDiscoveryClient struct{}

func (defaultDiscoveryClient) Discover() (upnp.DiscoveryResult, error) {
	return upnp.Discover()
}

func (s *service) settings() config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.WithDefaults()
}

func (s *service) updateSettings(next config.Config) (config.Config, error) {
	next = next.WithDefaults()
	if err := config.ValidateLocalListenAddr(next.ListenAddr); err != nil {
		return config.Config{}, err
	}
	if err := config.Save(s.configPath, next); err != nil {
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

func (s *service) status() StatusResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resp := StatusResponse{
		Discovered: s.gateway != nil,
		Ports:      append([]upnp.PortMapping{}, s.ports...),
	}
	if s.gateway != nil {
		resp.ServiceType = s.gateway.ServiceType
		resp.ControlURL = s.gateway.ControlURL
		resp.ExternalIP = s.externalIP
	}
	return resp
}

func (s *service) discover() (StatusResponse, error) {
	result, err := s.discovery.Discover()
	if err != nil {
		// discovery 未検出は UI 上の通常状態として扱い、現在の status を返す。
		if errors.Is(err, upnp.ErrNoGateway) {
			if s.logger != nil {
				s.logger.Info("router not discovered", "reason", err.Error())
			}
			return s.status(), nil
		}
		return StatusResponse{}, err
	}

	mapper := s.portMapperFactory(result)
	var externalIP string
	if mapper != nil {
		if ip, err := mapper.GetExternalIPAddress(); err == nil {
			externalIP = ip
		} else if s.logger != nil {
			s.logger.Warn("external IP lookup failed", "error", err)
		}
	}

	s.mu.Lock()
	resultCopy := result
	s.gateway = &resultCopy
	s.externalIP = externalIP
	s.mu.Unlock()

	if s.logger != nil {
		s.logger.Info("router discovered",
			"service_type", result.ServiceType,
			"control_url", result.ControlURL,
			"external_ip", externalIP,
		)
	}
	return s.status(), nil
}

func (s *service) openPort(mapping upnp.PortMapping) (StatusResponse, error) {
	if err := upnp.ValidatePortMapping(mapping); err != nil {
		return StatusResponse{}, err
	}
	mapper, err := s.currentPortMapper()
	if err != nil {
		return StatusResponse{}, err
	}
	if err := mapper.AddPortMapping(mapping); err != nil {
		return StatusResponse{}, err
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
	return s.status(), nil
}

func (s *service) closePort(mapping upnp.PortMapping) (StatusResponse, error) {
	if err := validateDeleteRequest(mapping); err != nil {
		return StatusResponse{}, err
	}
	mapper, err := s.currentPortMapper()
	if err != nil {
		return StatusResponse{}, err
	}
	if err := mapper.DeletePortMapping(mapping.Protocol, mapping.ExternalPort); err != nil {
		return StatusResponse{}, err
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
	return s.status(), nil
}

func (s *service) currentPortMapper() (PortMapper, error) {
	s.mu.RLock()
	gateway := s.gateway
	s.mu.RUnlock()
	if gateway == nil {
		return nil, errNoGateway
	}
	mapper := s.portMapperFactory(*gateway)
	if mapper == nil {
		return nil, fmt.Errorf("port mapper factory returned nil")
	}
	return mapper, nil
}

func upsertMapping(existing []upnp.PortMapping, next upnp.PortMapping) []upnp.PortMapping {
	for i, current := range existing {
		if sameMappingKey(current, next) {
			existing[i] = next
			return existing
		}
	}
	return append(existing, next)
}

func removeMapping(existing []upnp.PortMapping, protocol string, externalPort int) []upnp.PortMapping {
	filtered := existing[:0]
	for _, current := range existing {
		if sameMappingIdentity(current.Protocol, current.ExternalPort, protocol, externalPort) {
			continue
		}
		filtered = append(filtered, current)
	}
	return filtered
}

func sameMappingKey(a, b upnp.PortMapping) bool {
	return sameMappingIdentity(a.Protocol, a.ExternalPort, b.Protocol, b.ExternalPort)
}

func sameMappingIdentity(aProtocol string, aPort int, bProtocol string, bPort int) bool {
	return normalizeProtocol(aProtocol) == normalizeProtocol(bProtocol) && aPort == bPort
}

func normalizeProtocol(protocol string) string {
	return strings.ToUpper(strings.TrimSpace(protocol))
}

func validateDeleteRequest(mapping upnp.PortMapping) error {
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
