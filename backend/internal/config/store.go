// Package config は Porto アプリケーションの設定情報のロード、セーブ、および検証を行います。
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var renameFile = atomicRename

// FileStore は、設定値をローカルディスク上の JSON ファイルへ永続化（保存および読み込み）するためのストア実装構造体です。
type FileStore struct {
	Path string
}

// Load は、指定されたストアパスから設定値を読み込みます。
func (s FileStore) Load() (Config, error) {
	return Load(s.Path)
}

// Save は、指定された設定値をストアパスのファイルに書き込みます。
func (s FileStore) Save(cfg Config) error {
	return Save(s.Path, cfg)
}

// Load は、指定されたパスから設定ファイルを読み込みます。ファイルが存在しない場合は、デフォルト設定値を無害に返します。
func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	return cfg.WithDefaults(), nil
}

// Save は、設定ファイルを指定されたパスに書き込みます。必要に応じて親ディレクトリを再帰的に自動作成します。
func Save(path string, cfg Config) error {
	if path == "" {
		path = DefaultPath()
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir for %q: %w", path, err)
	}

	data, err := json.MarshalIndent(cfg.WithDefaults(), "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config %q: %w", path, err)
	}
	data = append(data, '\n')

	if err := writeFileAtomically(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	return nil
}

func writeFileAtomically(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := renameFile(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

// DefaultPath は、実行可能バイナリと同一ディレクトリ内の "config.json" の絶対パスをデフォルトとして計算して返します。
func DefaultPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(filepath.Dir(exe), "config.json")
}
