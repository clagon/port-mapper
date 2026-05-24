package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWithDefaults(t *testing.T) {
	tests := []struct {
		name string
		in   Config
		want string
	}{
		{
			name: "empty listen addr uses default",
			in:   Config{},
			want: "127.0.0.1:8080",
		},
		{
			name: "explicit listen addr is preserved",
			in:   Config{ListenAddr: "127.0.0.1:9090"},
			want: "127.0.0.1:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in.WithDefaults()
			if got.ListenAddr != tt.want {
				t.Fatalf("ListenAddr = %q, want %q", got.ListenAddr, tt.want)
			}
		})
	}
}

func TestLoadAndSave(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantErr   bool
		wantCfg   Config
		wantWrite bool
	}{
		{
			name:      "missing file returns defaults",
			content:   "",
			wantCfg:   DefaultConfig(),
			wantWrite: true,
		},
		{
			name:    "invalid json errors",
			content: "{",
			wantErr: true,
		},
		{
			name:    "valid json loads values",
			content: `{"listen_addr":"127.0.0.1:9090","auto_discover":true}`,
			wantCfg: Config{ListenAddr: "127.0.0.1:9090", AutoDiscover: BoolPtr(true)},
		},
		{
			name:    "explicit false is preserved",
			content: `{"listen_addr":"127.0.0.1:9090","auto_discover":false}`,
			wantCfg: Config{ListenAddr: "127.0.0.1:9090", AutoDiscover: BoolPtr(false)},
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
			if got.ListenAddr != tt.wantCfg.ListenAddr {
				t.Fatalf("Load().ListenAddr = %q, want %q", got.ListenAddr, tt.wantCfg.ListenAddr)
			}
			if got.AutoDiscover == nil || tt.wantCfg.AutoDiscover == nil {
				t.Fatalf("Load().AutoDiscover nil mismatch: got=%v want=%v", got.AutoDiscover, tt.wantCfg.AutoDiscover)
			}
			if *got.AutoDiscover != *tt.wantCfg.AutoDiscover {
				t.Fatalf("Load().AutoDiscover = %v, want %v", *got.AutoDiscover, *tt.wantCfg.AutoDiscover)
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
