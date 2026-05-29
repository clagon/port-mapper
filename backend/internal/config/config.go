// Package config は Porto アプリケーションの設定情報のロード、セーブ、および検証を行います。
package config

import (
	"fmt"
	"net"
)

const defaultListenAddr = "127.0.0.1:8080"

// Config は、バックエンドサーバーのランタイム設定を保持する構造体です。
type Config struct {
	ListenAddr   string `json:"listen_addr"`
	AutoDiscover *bool  `json:"auto_discover"`
}

// BoolPtr は、bool 値のポインタを返す便利なユーティリティ関数です。
func BoolPtr(v bool) *bool { return &v }

// DefaultConfig は、アプリケーションのデフォルト設定値を生成して返します。
func DefaultConfig() Config {
	return Config{
		ListenAddr:   defaultListenAddr,
		AutoDiscover: BoolPtr(true),
	}
}

// WithDefaults は、ゼロ値（未設定）のフィールドをデフォルト設定値で補完した新しい Config を返します。
func (c Config) WithDefaults() Config {
	defaults := DefaultConfig()
	if c.ListenAddr == "" {
		c.ListenAddr = defaults.ListenAddr
	}
	if c.AutoDiscover == nil {
		c.AutoDiscover = defaults.AutoDiscover
	}
	return c
}

// ValidateLocalListenAddr は、セキュリティ確保のため、リスンアドレスがローカルホスト（localhost / 127.0.0.1 / ::1）にのみバインドされていることを強制検証します。
func ValidateLocalListenAddr(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid listen addr %q: %w", addr, err)
	}
	if host == "" {
		return fmt.Errorf("listen addr must be local, got %q", addr)
	}
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return nil
	}
	return fmt.Errorf("listen addr must bind to localhost, got %q", addr)
}
