package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FileStore persists config values to a JSON file.
type FileStore struct {
	Path string
}

// Load reads config values from the store path.
func (s FileStore) Load() (Config, error) {
	return Load(s.Path)
}

// Save writes config values to the store path.
func (s FileStore) Save(cfg Config) error {
	return Save(s.Path, cfg)
}

// Load reads a config file from path. Missing files return defaults.
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

// Save writes the config file to path, creating parent directories as needed.
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

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	return nil
}

// DefaultPath returns the config path beside the executable.
func DefaultPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(filepath.Dir(exe), "config.json")
}
