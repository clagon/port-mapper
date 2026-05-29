package service

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/clagon/port-mapper/backend/internal/config"
	"github.com/clagon/port-mapper/backend/internal/upnp"
)

func TestDefaultPortMapperFactoryUsesTimeoutClient(t *testing.T) {
	client := defaultPortMapperFactory(upnp.DiscoveryResult{ServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2", ControlURL: "http://192.168.1.1:1900/control"})
	soap, ok := client.(*upnp.SOAPClient)
	if !ok {
		t.Fatalf("defaultPortMapperFactory() type = %T, want *upnp.SOAPClient", client)
	}
	if soap.HTTPClient == nil {
		t.Fatal("HTTPClient = nil")
	}
	if soap.HTTPClient.Timeout <= 0 {
		t.Fatalf("HTTPClient timeout = %v, want > 0", soap.HTTPClient.Timeout)
	}
}

type fakeDiscovery struct {
	result upnp.DiscoveryResult
	err    error
	calls  int
}

func (f *fakeDiscovery) Discover() (upnp.DiscoveryResult, error) {
	f.calls++
	return f.result, f.err
}

type deleteCall struct {
	protocol     string
	externalPort int
}

type fakeMapper struct {
	externalIP  string
	externalErr error
	addErr      error
	deleteErr   error
	entries     []upnp.PortMapping
	addCalls    []upnp.PortMapping
	deleteCalls []deleteCall
}

func (f *fakeMapper) GetExternalIPAddress() (string, error) {
	if f.externalErr != nil {
		return "", f.externalErr
	}
	return f.externalIP, nil
}

func (f *fakeMapper) AddPortMapping(m upnp.PortMapping) error {
	f.addCalls = append(f.addCalls, m)
	return f.addErr
}

func (f *fakeMapper) DeletePortMapping(protocol string, externalPort int) error {
	f.deleteCalls = append(f.deleteCalls, deleteCall{protocol: protocol, externalPort: externalPort})
	return f.deleteErr
}

func (f *fakeMapper) GetGenericPortMappingEntry(index int) (upnp.PortMapping, error) {
	if index < 0 || index >= len(f.entries) {
		return upnp.PortMapping{}, errNoGateway
	}
	return f.entries[index], nil
}

func newTestService(cfgPath string, discovery discoveryClient, mapper *fakeMapper) *Service {
	return New(Options{
		ConfigPath: cfgPath,
		Config:     config.DefaultConfig(),
		discovery:  discovery,
		portMapperFactory: func(upnp.DiscoveryResult) portMapper {
			return mapper
		},
	})
}

func TestSettingsPersistToDisk(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	svc := New(Options{ConfigPath: cfgPath, Config: config.DefaultConfig()})

	next := config.Config{ListenAddr: "127.0.0.1:9090", AutoDiscover: config.BoolPtr(false)}
	if _, err := svc.UpdateSettings(next); err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}

	loaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.ListenAddr != next.ListenAddr {
		t.Fatalf("listen_addr = %q, want %q", loaded.ListenAddr, next.ListenAddr)
	}
	if loaded.AutoDiscover == nil || *loaded.AutoDiscover {
		t.Fatalf("auto_discover = %v, want false", loaded.AutoDiscover)
	}
}

func TestDiscoverUpdatesStatusAndSoftNoGateway(t *testing.T) {
	discovery := &fakeDiscovery{result: upnp.DiscoveryResult{
		ServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
		ControlURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn2",
	}}
	mapper := &fakeMapper{externalIP: "203.0.113.42"}
	svc := newTestService(filepath.Join(t.TempDir(), "config.json"), discovery, mapper)

	got, err := svc.Discover()
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if discovery.calls != 1 {
		t.Fatalf("discover calls = %d, want 1", discovery.calls)
	}
	if !got.Discovered || got.ControlURL != discovery.result.ControlURL || got.ExternalIP != mapper.externalIP {
		t.Fatalf("status = %+v", got)
	}

	soft := newTestService(filepath.Join(t.TempDir(), "config.json"), &fakeDiscovery{err: upnp.ErrNoGateway}, &fakeMapper{})
	got, err = soft.Discover()
	if err != nil {
		t.Fatalf("Discover() no gateway error = %v", err)
	}
	if got.Discovered {
		t.Fatal("Discovered = true, want false")
	}
}

func TestOpenAndClosePortUpdatesStatus(t *testing.T) {
	discovery := &fakeDiscovery{result: upnp.DiscoveryResult{
		ServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
		ControlURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn2",
	}}
	mapper := &fakeMapper{externalIP: "203.0.113.42"}
	svc := newTestService(filepath.Join(t.TempDir(), "config.json"), discovery, mapper)
	if _, err := svc.Discover(); err != nil {
		t.Fatalf("Discover() error = %v", err)
	}

	mapping := upnp.PortMapping{
		Protocol:             "TCP",
		ExternalPort:         8080,
		InternalIP:           "192.168.1.20",
		InternalPort:         8080,
		Description:          "test mapping",
		LeaseDurationSeconds: 3600,
	}
	got, err := svc.OpenPort(mapping)
	if err != nil {
		t.Fatalf("OpenPort() error = %v", err)
	}
	if len(mapper.addCalls) != 1 || len(got.Ports) != 1 {
		t.Fatalf("add calls = %d ports = %d", len(mapper.addCalls), len(got.Ports))
	}

	got, err = svc.ClosePort(upnp.PortMapping{Protocol: "TCP", ExternalPort: 8080})
	if err != nil {
		t.Fatalf("ClosePort() error = %v", err)
	}
	if len(mapper.deleteCalls) != 1 || len(got.Ports) != 0 {
		t.Fatalf("delete calls = %d ports = %d", len(mapper.deleteCalls), len(got.Ports))
	}
}

type recordingSettingsStore struct {
	saved []config.Config
	err   error
}

func (s *recordingSettingsStore) Save(cfg config.Config) error {
	if s.err != nil {
		return s.err
	}
	s.saved = append(s.saved, cfg)
	return nil
}

func TestUpdateSettingsUsesInjectedSettingsStore(t *testing.T) {
	store := &recordingSettingsStore{}
	svc := New(Options{
		Config:        config.DefaultConfig(),
		SettingsStore: store,
	})

	next, err := svc.UpdateSettings(config.Config{ListenAddr: "127.0.0.1:9090", AutoDiscover: config.BoolPtr(false)})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	if len(store.saved) != 1 {
		t.Fatalf("Save() calls = %d, want 1", len(store.saved))
	}
	if store.saved[0].ListenAddr != next.ListenAddr {
		t.Fatalf("saved ListenAddr = %q, want %q", store.saved[0].ListenAddr, next.ListenAddr)
	}
	if got := svc.Settings().ListenAddr; got != next.ListenAddr {
		t.Fatalf("Settings().ListenAddr = %q, want %q", got, next.ListenAddr)
	}
}

func TestUpdateSettingsDoesNotMutateConfigWhenSettingsStoreFails(t *testing.T) {
	storeErr := errors.New("save failed")
	svc := New(Options{
		Config:        config.Config{ListenAddr: "127.0.0.1:8080", AutoDiscover: config.BoolPtr(true)},
		SettingsStore: &recordingSettingsStore{err: storeErr},
	})

	_, err := svc.UpdateSettings(config.Config{ListenAddr: "127.0.0.1:9090", AutoDiscover: config.BoolPtr(false)})
	if !errors.Is(err, storeErr) {
		t.Fatalf("UpdateSettings() error = %v, want %v", err, storeErr)
	}
	if got := svc.Settings().ListenAddr; got != "127.0.0.1:8080" {
		t.Fatalf("Settings().ListenAddr = %q, want original", got)
	}
}
