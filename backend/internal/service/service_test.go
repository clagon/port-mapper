package service

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/clagon/port-mapper/backend/internal/config"
	"github.com/clagon/port-mapper/backend/internal/domain"
)

type fakeDiscovery struct {
	result domain.DiscoveryResult
	err    error
	calls  int
}

func (f *fakeDiscovery) Discover() (domain.DiscoveryResult, error) {
	f.calls++
	return f.result, f.err
}

type deleteCall struct {
	protocol     string
	externalPort int
}

type fakeMapper struct {
	externalIP       string
	externalErr      error
	addErr           error
	deleteErr        error
	entryErr         error
	entries          []domain.PortMapping
	entryStarted     chan struct{}
	releaseEntry     chan struct{}
	entryStartedOnce sync.Once
	addCalls         []domain.PortMapping
	deleteCalls      []deleteCall
}

func (f *fakeMapper) GetExternalIPAddress() (string, error) {
	if f.externalErr != nil {
		return "", f.externalErr
	}
	return f.externalIP, nil
}

func (f *fakeMapper) AddPortMapping(m domain.PortMapping) error {
	f.addCalls = append(f.addCalls, m)
	return f.addErr
}

func (f *fakeMapper) DeletePortMapping(protocol string, externalPort int) error {
	f.deleteCalls = append(f.deleteCalls, deleteCall{protocol: protocol, externalPort: externalPort})
	return f.deleteErr
}

func (f *fakeMapper) GetGenericPortMappingEntry(index int) (domain.PortMapping, error) {
	if index < 0 || index >= len(f.entries) {
		if f.entryErr != nil {
			return domain.PortMapping{}, f.entryErr
		}
		return domain.PortMapping{}, domain.ErrNoPortMappingEntry
	}
	if f.entryStarted != nil {
		f.entryStartedOnce.Do(func() { close(f.entryStarted) })
	}
	if f.releaseEntry != nil {
		<-f.releaseEntry
	}
	return f.entries[index], nil
}

func newTestService(cfgPath string, discovery domain.DiscoveryClient, mapper *fakeMapper) *Service {
	return New(Options{
		ConfigPath: cfgPath,
		Config:     config.DefaultConfig(),
		Discovery:  discovery,
		PortMapperFactory: func(domain.DiscoveryResult) domain.PortMapper {
			return mapper
		},
	})
}

func TestSettingsPersistToDisk(t *testing.T) {
	tests := []struct {
		name string
		next config.Config
	}{
		{
			name: "persist updated settings",
			next: config.Config{ListenAddr: "127.0.0.1:9090", AutoDiscover: config.BoolPtr(false)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgPath := filepath.Join(t.TempDir(), "config.json")
			svc := New(Options{ConfigPath: cfgPath, Config: config.DefaultConfig()})

			if _, err := svc.UpdateSettings(tt.next); err != nil {
				t.Fatalf("UpdateSettings() error = %v", err)
			}

			loaded, err := config.Load(cfgPath)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if loaded.ListenAddr != tt.next.ListenAddr {
				t.Fatalf("listen_addr = %q, want %q", loaded.ListenAddr, tt.next.ListenAddr)
			}
			if loaded.AutoDiscover == nil || *loaded.AutoDiscover {
				t.Fatalf("auto_discover = %v, want false", loaded.AutoDiscover)
			}
		})
	}
}

func TestDiscoverUpdatesStatusAndSoftNoGateway(t *testing.T) {
	tests := []struct {
		name           string
		discovery      *fakeDiscovery
		mapper         *fakeMapper
		wantDiscovered bool   // Status.Discovered
		wantCalls      int    // fakeDiscovery.Discover call count
		wantControlURL string // Status.ControlURL
		wantExternalIP string // Status.ExternalIP
	}{
		{
			name: "updates discovered status",
			discovery: &fakeDiscovery{result: domain.DiscoveryResult{
				ServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
				ControlURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn2",
			}},
			mapper:         &fakeMapper{externalIP: "203.0.113.42"},
			wantDiscovered: true,
			wantCalls:      1,
			wantControlURL: "http://192.168.1.1:1900/upnp/control/WANIPConn2",
			wantExternalIP: "203.0.113.42",
		},
		{
			name:           "soft no gateway",
			discovery:      &fakeDiscovery{err: domain.ErrNoGateway},
			mapper:         &fakeMapper{},
			wantDiscovered: false,
			wantCalls:      1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService(filepath.Join(t.TempDir(), "config.json"), tt.discovery, tt.mapper)

			got, err := svc.Discover()
			if err != nil {
				t.Fatalf("Discover() error = %v", err)
			}
			if tt.discovery.calls != tt.wantCalls {
				t.Fatalf("discover calls = %d, want %d", tt.discovery.calls, tt.wantCalls)
			}
			if got.Discovered != tt.wantDiscovered {
				t.Fatalf("Discovered = %v, want %v", got.Discovered, tt.wantDiscovered)
			}
			if tt.wantControlURL != "" && got.ControlURL != tt.wantControlURL {
				t.Fatalf("ControlURL = %q, want %q", got.ControlURL, tt.wantControlURL)
			}
			if tt.wantExternalIP != "" && got.ExternalIP != tt.wantExternalIP {
				t.Fatalf("ExternalIP = %q, want %q", got.ExternalIP, tt.wantExternalIP)
			}
		})
	}
}

func TestSyncActivePortsReplacesLocalMappingsAfterCompleteFetch(t *testing.T) {
	localIP := "192.168.1.20"
	svc := New(Options{Config: config.DefaultConfig()})
	svc.ports = []domain.PortMapping{
		{Protocol: "TCP", ExternalPort: 25565, InternalIP: localIP, InternalPort: 25565, Description: "manual old", LeaseDurationSeconds: 3600},
		{Protocol: "UDP", ExternalPort: 19132, InternalIP: localIP, InternalPort: 19132, Description: "expired", LeaseDurationSeconds: 60},
		{Protocol: "TCP", ExternalPort: 9000, InternalIP: "192.168.1.30", InternalPort: 9000, Description: "other host", LeaseDurationSeconds: 0},
	}
	mapper := &fakeMapper{entries: []domain.PortMapping{
		{Protocol: "TCP", ExternalPort: 25565, InternalIP: localIP, InternalPort: 25565, Description: "manual current", LeaseDurationSeconds: 7200},
		{Protocol: "TCP", ExternalPort: 8080, InternalIP: "192.168.1.30", InternalPort: 8080, Description: "other router entry", LeaseDurationSeconds: 0},
	}}

	svc.syncActivePorts(mapper, localIP)

	got := svc.Status().Ports
	if len(got) != 2 {
		t.Fatalf("ports = %#v, want 2 entries", got)
	}
	assertHasMapping(t, got, domain.PortMapping{Protocol: "TCP", ExternalPort: 25565, InternalIP: localIP, InternalPort: 25565, Description: "manual current", LeaseDurationSeconds: 7200})
	assertHasMapping(t, got, domain.PortMapping{Protocol: "TCP", ExternalPort: 9000, InternalIP: "192.168.1.30", InternalPort: 9000, Description: "other host", LeaseDurationSeconds: 0})
	assertMissingMapping(t, got, "UDP", 19132)
}

func TestSyncActivePortsCommitsWhenRouterHasExactlySafetyLimitEntries(t *testing.T) {
	localIP := "192.168.1.20"
	svc := New(Options{Config: config.DefaultConfig()})
	svc.ports = []domain.PortMapping{
		{Protocol: "UDP", ExternalPort: 19132, InternalIP: localIP, InternalPort: 19132, Description: "expired", LeaseDurationSeconds: 60},
	}

	entries := make([]domain.PortMapping, 256)
	for i := range entries {
		entries[i] = domain.PortMapping{
			Protocol:             "TCP",
			ExternalPort:         10000 + i,
			InternalIP:           localIP,
			InternalPort:         10000 + i,
			Description:          "router entry",
			LeaseDurationSeconds: 3600,
		}
	}
	mapper := &fakeMapper{entries: entries}

	svc.syncActivePorts(mapper, localIP)

	got := svc.Status().Ports
	if len(got) != len(entries) {
		t.Fatalf("ports = %d, want %d", len(got), len(entries))
	}
	assertHasMapping(t, got, entries[0])
	assertHasMapping(t, got, entries[len(entries)-1])
	assertMissingMapping(t, got, "UDP", 19132)
}

func TestSyncActivePortsPreservesConcurrentOpenPort(t *testing.T) {
	localIP := "192.168.1.20"
	entryStarted := make(chan struct{})
	releaseEntry := make(chan struct{})
	mapper := &fakeMapper{
		entries: []domain.PortMapping{
			{Protocol: "TCP", ExternalPort: 25565, InternalIP: localIP, InternalPort: 25565, Description: "router current", LeaseDurationSeconds: 3600},
		},
		entryStarted: entryStarted,
		releaseEntry: releaseEntry,
	}
	svc := New(Options{
		Config: config.DefaultConfig(),
		PortMapperFactory: func(domain.DiscoveryResult) domain.PortMapper {
			return mapper
		},
	})
	svc.gateway = &domain.DiscoveryResult{
		ServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
		ControlURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn2",
	}
	svc.ports = []domain.PortMapping{
		{Protocol: "UDP", ExternalPort: 19132, InternalIP: localIP, InternalPort: 19132, Description: "expired", LeaseDurationSeconds: 60},
	}

	syncDone := make(chan struct{})
	go func() {
		svc.syncActivePorts(mapper, localIP)
		close(syncDone)
	}()
	<-entryStarted

	opened := domain.PortMapping{
		Protocol:             "TCP",
		ExternalPort:         8080,
		InternalIP:           localIP,
		InternalPort:         8080,
		Description:          "opened concurrently",
		LeaseDurationSeconds: 3600,
	}
	openDone := make(chan error, 1)
	go func() {
		_, err := svc.OpenPort(opened)
		openDone <- err
	}()

	close(releaseEntry)
	<-syncDone
	if err := <-openDone; err != nil {
		t.Fatalf("OpenPort() error = %v", err)
	}

	got := svc.Status().Ports
	assertHasMapping(t, got, mapper.entries[0])
	assertHasMapping(t, got, opened)
	assertMissingMapping(t, got, "UDP", 19132)
}

func TestSyncActivePortsDoesNotMutateOnFetchFailure(t *testing.T) {
	localIP := "192.168.1.20"
	fetchErr := errors.New("temporary router failure")
	svc := New(Options{Config: config.DefaultConfig()})
	svc.ports = []domain.PortMapping{
		{Protocol: "TCP", ExternalPort: 25565, InternalIP: localIP, InternalPort: 25565, Description: "manual old", LeaseDurationSeconds: 3600},
	}
	mapper := &fakeMapper{
		entries: []domain.PortMapping{
			{Protocol: "TCP", ExternalPort: 8080, InternalIP: localIP, InternalPort: 8080, Description: "partial", LeaseDurationSeconds: 3600},
		},
		entryErr: fetchErr,
	}

	svc.syncActivePorts(mapper, localIP)

	got := svc.Status().Ports
	if len(got) != 1 {
		t.Fatalf("ports = %#v, want original entry only", got)
	}
	assertHasMapping(t, got, domain.PortMapping{Protocol: "TCP", ExternalPort: 25565, InternalIP: localIP, InternalPort: 25565, Description: "manual old", LeaseDurationSeconds: 3600})
	assertMissingMapping(t, got, "TCP", 8080)
}

func assertHasMapping(t *testing.T, mappings []domain.PortMapping, want domain.PortMapping) {
	t.Helper()
	for _, got := range mappings {
		if got == want {
			return
		}
	}
	t.Fatalf("mapping %#v not found in %#v", want, mappings)
}

func assertMissingMapping(t *testing.T, mappings []domain.PortMapping, protocol string, externalPort int) {
	t.Helper()
	for _, got := range mappings {
		if sameMappingIdentity(got.Protocol, got.ExternalPort, protocol, externalPort) {
			t.Fatalf("mapping %s/%d unexpectedly found in %#v", protocol, externalPort, mappings)
		}
	}
}

func TestOpenAndClosePortUpdatesStatus(t *testing.T) {
	tests := []struct {
		name    string
		mapping domain.PortMapping
	}{
		{
			name: "open then close port",
			mapping: domain.PortMapping{
				Protocol:             "TCP",
				ExternalPort:         8080,
				InternalIP:           "192.168.1.20",
				InternalPort:         8080,
				Description:          "test mapping",
				LeaseDurationSeconds: 3600,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discovery := &fakeDiscovery{result: domain.DiscoveryResult{
				ServiceType: "urn:schemas-upnp-org:service:WANIPConnection:2",
				ControlURL:  "http://192.168.1.1:1900/upnp/control/WANIPConn2",
			}}
			mapper := &fakeMapper{externalIP: "203.0.113.42"}
			svc := newTestService(filepath.Join(t.TempDir(), "config.json"), discovery, mapper)
			if _, err := svc.Discover(); err != nil {
				t.Fatalf("Discover() error = %v", err)
			}

			got, err := svc.OpenPort(tt.mapping)
			if err != nil {
				t.Fatalf("OpenPort() error = %v", err)
			}
			if len(mapper.addCalls) != 1 || len(got.Ports) != 1 {
				t.Fatalf("add calls = %d ports = %d", len(mapper.addCalls), len(got.Ports))
			}

			got, err = svc.ClosePort(domain.PortMapping{Protocol: tt.mapping.Protocol, ExternalPort: tt.mapping.ExternalPort})
			if err != nil {
				t.Fatalf("ClosePort() error = %v", err)
			}
			if len(mapper.deleteCalls) != 1 || len(got.Ports) != 0 {
				t.Fatalf("delete calls = %d ports = %d", len(mapper.deleteCalls), len(got.Ports))
			}
		})
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
	tests := []struct {
		name string
		next config.Config
	}{
		{
			name: "uses injected settings store",
			next: config.Config{ListenAddr: "127.0.0.1:9090", AutoDiscover: config.BoolPtr(false)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &recordingSettingsStore{}
			svc := New(Options{
				Config:        config.DefaultConfig(),
				SettingsStore: store,
			})

			next, err := svc.UpdateSettings(tt.next)
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
		})
	}
}

func TestUpdateSettingsDoesNotMutateConfigWhenSettingsStoreFails(t *testing.T) {
	tests := []struct {
		name           string
		initial        config.Config
		next           config.Config
		wantListenAddr string // Service.Settings().ListenAddr
	}{
		{
			name:           "settings unchanged on save error",
			initial:        config.Config{ListenAddr: "127.0.0.1:8080", AutoDiscover: config.BoolPtr(true)},
			next:           config.Config{ListenAddr: "127.0.0.1:9090", AutoDiscover: config.BoolPtr(false)},
			wantListenAddr: "127.0.0.1:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storeErr := errors.New("save failed")
			svc := New(Options{
				Config:        tt.initial,
				SettingsStore: &recordingSettingsStore{err: storeErr},
			})

			_, err := svc.UpdateSettings(tt.next)
			if !errors.Is(err, storeErr) {
				t.Fatalf("UpdateSettings() error = %v, want %v", err, storeErr)
			}
			if got := svc.Settings().ListenAddr; got != tt.wantListenAddr {
				t.Fatalf("Settings().ListenAddr = %q, want original", got)
			}
		})
	}
}
