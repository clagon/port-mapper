package config

import (
	"errors"
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

func TestSaveKeepsExistingConfigWhenRenameFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	original := []byte(`{"listen_addr":"127.0.0.1:9090","auto_discover":false}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	renameErr := errors.New("rename failed")
	oldRenameFile := renameFile
	renameFile = func(_, _ string) error { return renameErr }
	t.Cleanup(func() { renameFile = oldRenameFile })

	err := Save(path, Config{ListenAddr: "127.0.0.1:61234", AutoDiscover: BoolPtr(true)})
	if !errors.Is(err, renameErr) {
		t.Fatalf("Save() error = %v, want %v", err, renameErr)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("config content = %q, want %q", got, original)
	}

	matches, err := filepath.Glob(filepath.Join(dir, ".config.json.tmp-*"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
}

func TestSavePreservesConfigSymlink(t *testing.T) {
	dir := t.TempDir()
	targetDir := filepath.Join(dir, "managed")
	if err := os.Mkdir(targetDir, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	targetPath := filepath.Join(targetDir, "config.json")
	if err := os.WriteFile(targetPath, []byte(`{"listen_addr":"127.0.0.1:9090","auto_discover":false}`), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	linkPath := filepath.Join(dir, "config.json")
	linkTarget := filepath.Join("managed", "config.json")
	if err := os.Symlink(linkTarget, linkPath); err != nil {
		t.Skipf("Symlink() error = %v", err)
	}

	if err := Save(linkPath, Config{ListenAddr: "127.0.0.1:61234", AutoDiscover: BoolPtr(true)}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("Lstat() error = %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config path mode = %v, want symlink", info.Mode())
	}

	gotTarget, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if gotTarget != linkTarget {
		t.Fatalf("symlink target = %q, want %q", gotTarget, linkTarget)
	}

	got, err := Load(targetPath)
	if err != nil {
		t.Fatalf("Load() target error = %v", err)
	}
	if got.ListenAddr != "127.0.0.1:61234" {
		t.Fatalf("target ListenAddr = %q, want %q", got.ListenAddr, "127.0.0.1:61234")
	}
	if got.AutoDiscover == nil || !*got.AutoDiscover {
		t.Fatalf("target AutoDiscover = %v, want true", got.AutoDiscover)
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
