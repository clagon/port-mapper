package config

const defaultListenAddr = "127.0.0.1:8080"

// Config holds runtime settings for the backend.
type Config struct {
	ListenAddr   string `json:"listen_addr"`
	AutoDiscover *bool  `json:"auto_discover"`
}

// BoolPtr returns a pointer to v.
func BoolPtr(v bool) *bool { return &v }

// DefaultConfig returns the application's default configuration.
func DefaultConfig() Config {
	return Config{
		ListenAddr:   defaultListenAddr,
		AutoDiscover: BoolPtr(true),
	}
}

// WithDefaults fills in zero-value fields with defaults.
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
