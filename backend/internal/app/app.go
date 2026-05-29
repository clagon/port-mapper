package app

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/clagon/port-mapper/backend/internal/config"
	"github.com/clagon/port-mapper/backend/internal/server"
	"github.com/clagon/port-mapper/backend/internal/service"
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
	Logger        *slog.Logger
}

// App is the top-level application container.
type App struct {
	cfg           config.Config
	server        *server.Server
	service       *service.Service
	configPath    string
	openBrowser   bool
	browserOpener BrowserOpener
	logger        *slog.Logger
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
	if err := config.ValidateLocalListenAddr(cfg.ListenAddr); err != nil {
		return nil, err
	}

	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	svc := service.New(service.Options{
		ConfigPath: configPath,
		Config:     cfg,
		Logger:     logger,
	})

	return &App{
		cfg:           cfg,
		server:        server.New(cfg.ListenAddr, logger, svc),
		service:       svc,
		configPath:    configPath,
		openBrowser:   opts.OpenBrowser,
		browserOpener: opts.BrowserOpener,
		logger:        logger,
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
	if a.logger != nil {
		a.logger.Info("opening browser", "url", a.browserURL())
	}
	return a.browserOpener.Open(a.browserURL())
}

// Run starts the HTTP server.
func (a *App) Run() error {
	if a == nil || a.server == nil {
		return nil
	}
	if a.logger != nil {
		a.logger.Info("starting backend",
			"config_path", a.configPath,
			"listen_addr", a.server.Addr(),
			"auto_discover", boolValue(a.cfg.AutoDiscover),
			"browser_open", a.openBrowser,
		)
	}
	ln, err := net.Listen("tcp", a.server.Addr())
	if err != nil {
		if a.logger != nil {
			a.logger.Error("failed to bind listen address", "listen_addr", a.server.Addr(), "error", err)
		}
		return err
	}
	if a.logger != nil {
		a.logger.Info("listening", "listen_addr", a.server.Addr())
	}
	if boolValue(a.cfg.AutoDiscover) {
		go func() {
			if a.logger != nil {
				a.logger.Info("auto discovering router")
			}
			if _, err := a.service.Discover(); err != nil && a.logger != nil {
				a.logger.Warn("auto discovery failed", "error", err)
			}
		}()
	}
	if a.openBrowser && a.browserOpener != nil {
		if a.logger != nil {
			a.logger.Info("opening browser", "url", a.browserURL())
		}
		if err := a.browserOpener.Open(a.browserURL()); err != nil && a.logger != nil {
			a.logger.Warn("browser open failed", "url", a.browserURL(), "error", err)
		}
	}
	if a.logger != nil {
		a.logger.Info("server running")
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

func boolValue(v *bool) bool {
	return v != nil && *v
}
