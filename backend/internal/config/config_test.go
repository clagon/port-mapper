package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWithDefaults(t *testing.T) {
	tests := []struct {
		name           string
		in             Config
		wantListenAddr string // Config.WithDefaults().ListenAddr
	}{
		{
			name:           "empty listen addr uses default",
			in:             Config{},
			wantListenAddr: "127.0.0.1:61234",
		},
		{
			name:           "explicit listen addr is preserved",
			in:             Config{ListenAddr: "127.0.0.1:9090"},
			wantListenAddr: "127.0.0.1:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.WithDefaults()
			if got.ListenAddr != tt.wantListenAddr {
				t.Fatalf("ListenAddr = %q, want %q", got.ListenAddr, tt.wantListenAddr)
			}
		})
	}
}

func TestLoadAndSave(t *testing.T) {
	tests := []struct {
		name             string
		content          string
		wantErr          bool   // Load() error presence
		wantListenAddr   string // Load().ListenAddr
		wantAutoDiscover *bool  // Load().AutoDiscover
		wantWrite        bool   // whether Save() should be exercised
	}{
		{
			name:             "missing file returns defaults",
			content:          "",
			wantListenAddr:   DefaultConfig().ListenAddr,
			wantAutoDiscover: DefaultConfig().AutoDiscover,
			wantWrite:        true,
		},
		{
			name:    "invalid json errors",
			content: "{",
			wantErr: true,
		},
		{
			name:             "valid json loads values",
			content:          `{"listen_addr":"127.0.0.1:9090","auto_discover":true}`,
			wantListenAddr:   "127.0.0.1:9090",
			wantAutoDiscover: BoolPtr(true),
		},
		{
			name:             "explicit false is preserved",
			content:          `{"listen_addr":"127.0.0.1:9090","auto_discover":false}`,
			wantListenAddr:   "127.0.0.1:9090",
			wantAutoDiscover: BoolPtr(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.json")

			if tt.content != "" {
				if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
			}

			got, err := Load(path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got.ListenAddr != tt.wantListenAddr {
				t.Fatalf("Load().ListenAddr = %q, want %q", got.ListenAddr, tt.wantListenAddr)
			}
			if got.AutoDiscover == nil || tt.wantAutoDiscover == nil {
				t.Fatalf("Load().AutoDiscover nil mismatch: got=%v want=%v", got.AutoDiscover, tt.wantAutoDiscover)
			}
			if *got.AutoDiscover != *tt.wantAutoDiscover {
				t.Fatalf("Load().AutoDiscover = %v, want %v", *got.AutoDiscover, *tt.wantAutoDiscover)
			}

			if tt.wantWrite {
				if err := Save(path, got); err != nil {
					t.Fatalf("Save() error = %v", err)
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("ReadFile() error = %v", err)
				}
				if len(data) == 0 {
					t.Fatal("Save() wrote empty file")
				}
			}
		})
	}
}

func TestFileStoreLoadAndSave(t *testing.T) {
	tests := []struct {
		name             string
		wantListenAddr   string // FileStore.Load().ListenAddr
		wantAutoDiscover *bool  // FileStore.Load().AutoDiscover
	}{
		{
			name:             "save then load",
			wantListenAddr:   "127.0.0.1:9090",
			wantAutoDiscover: BoolPtr(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "nested", "config.json")
			store := FileStore{Path: path}

			if err := store.Save(Config{ListenAddr: tt.wantListenAddr, AutoDiscover: tt.wantAutoDiscover}); err != nil {
				t.Fatalf("FileStore.Save() error = %v", err)
			}
			got, err := store.Load()
			if err != nil {
				t.Fatalf("FileStore.Load() error = %v", err)
			}
			if got.ListenAddr != tt.wantListenAddr {
				t.Fatalf("FileStore.Load().ListenAddr = %q, want %q", got.ListenAddr, tt.wantListenAddr)
			}
			if got.AutoDiscover == nil || *got.AutoDiscover != *tt.wantAutoDiscover {
				t.Fatalf("FileStore.Load().AutoDiscover = %v, want %v", got.AutoDiscover, *tt.wantAutoDiscover)
			}
		})
	}
}
