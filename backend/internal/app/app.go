package app

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/clagon/port-mapper/backend/internal/config"
	"github.com/clagon/port-mapper/backend/internal/server"
)

const defaultListenAddr = "127.0.0.1:8080"

// BrowserOpener opens a URL in the user's browser.
type BrowserOpener interface {
	Open(string) error
}

// AppOptions configures a new App.
type AppOptions struct {
	ListenAddr    string
	ConfigPath    string
	OpenBrowser   bool
	BrowserOpener BrowserOpener
}

// App is the top-level application container.
type App struct {
	cfg           config.Config
	server        *server.Server
	configPath    string
	openBrowser   bool
	browserOpener BrowserOpener
}

// New constructs a new App using the provided options.
func New(opts AppOptions) (*App, error) {
	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = config.DefaultPath()
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	if opts.ListenAddr != "" {
		cfg.ListenAddr = opts.ListenAddr
	}
	cfg = cfg.WithDefaults()
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = defaultListenAddr
	}
	if err := validateLocalListenAddr(cfg.ListenAddr); err != nil {
		return nil, err
	}

	return &App{
		cfg:           cfg,
		server:        server.New(cfg.ListenAddr),
		configPath:    configPath,
		openBrowser:   opts.OpenBrowser,
		browserOpener: opts.BrowserOpener,
	}, nil
}

// ConfigPath returns the config file path used by the application.
func (a *App) ConfigPath() string {
	if a == nil {
		return ""
	}
	return a.configPath
}

// Addr returns the configured listen address.
func (a *App) Addr() string {
	if a == nil || a.server == nil {
		return ""
	}
	return a.server.Addr()
}

// Handler returns the application's HTTP handler.
func (a *App) Handler() http.Handler {
	if a == nil || a.server == nil {
		return http.NewServeMux()
	}
	return a.server.Handler()
}

// Start performs one-time startup actions like opening the browser.
func (a *App) Start() error {
	if a == nil || !a.openBrowser || a.browserOpener == nil {
		return nil
	}
	return a.browserOpener.Open(a.browserURL())
}

// Run starts the HTTP server.
func (a *App) Run() error {
	if a == nil || a.server == nil {
		return nil
	}
	ln, err := net.Listen("tcp", a.server.Addr())
	if err != nil {
		return err
	}
	if a.openBrowser && a.browserOpener != nil {
		if err := a.browserOpener.Open(a.browserURL()); err != nil {
			// Browser launch failures should not prevent the server from starting.
			_ = err
		}
	}
	return a.server.Serve(ln)
}

func (a *App) browserURL() string {
	addr := a.Addr()
	if addr == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr + "/"
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("http://%s:%s/", host, port)
}

func validateLocalListenAddr(addr string) error {
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
