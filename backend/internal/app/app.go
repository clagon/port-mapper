// Package app は、Porto アプリケーション全体のコンテナ、ライフサイクル管理、および起動処理（DIなど）を提供します。
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
	"github.com/clagon/port-mapper/backend/internal/upnp"
)

const defaultListenAddr = "127.0.0.1:61234"

// BrowserOpener は、指定された URL をユーザーのブラウザで開くためのインターフェースです。
type BrowserOpener interface {
	// Open はブラウザでURLを開きます。
	Open(string) error
}

// AppOptions は、App の初期化を設定するためのオプションです。
type AppOptions struct {
	ListenAddr    string
	ConfigPath    string
	OpenBrowser   bool
	BrowserOpener BrowserOpener
	Logger        *slog.Logger
}

// App は、アプリケーション全体の最上位コンテナであり、サーバーとサービスのライフサイクルを統括します。
type App struct {
	cfg           config.Config
	server        *server.Server
	service       *service.Service
	configPath    string
	openBrowser   bool
	browserOpener BrowserOpener
	logger        *slog.Logger
}

// New は、指定されたオプションを使用して新しい App インスタンスを構築します。
func New(opts AppOptions) (*App, error) {
	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = config.DefaultPath()
	}

	settingsStore := config.FileStore{Path: configPath}
	cfg, err := settingsStore.Load()
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
		ConfigPath:        configPath,
		Config:            cfg,
		Logger:            logger,
		SettingsStore:     settingsStore,
		Discovery:         upnp.NewDiscoveryClient(),
		PortMapperFactory: upnp.NewSOAPPortMapper,
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

// ConfigPath は、アプリケーションが使用している設定ファイルのパスを返します。
func (a *App) ConfigPath() string {
	if a == nil {
		return ""
	}
	return a.configPath
}

// Addr は、サーバーがバインドされている、またはバインド予定のリスンアドレスを返します。
func (a *App) Addr() string {
	if a == nil || a.server == nil {
		return ""
	}
	return a.server.Addr()
}

// Handler は、アプリケーションの HTTP ハンドラー（Router）を返します。
func (a *App) Handler() http.Handler {
	if a == nil || a.server == nil {
		return http.NewServeMux()
	}
	return a.server.Handler()
}

// Start は、ブラウザの自動起動など、サーバー実行前の初期起動処理を行います。
func (a *App) Start() error {
	if a == nil || !a.openBrowser || a.browserOpener == nil {
		return nil
	}
	if a.logger != nil {
		a.logger.Info("opening browser", "url", a.browserURL())
	}
	return a.browserOpener.Open(a.browserURL())
}

// Run は、HTTP サーバーのリスナーを起動し、バックグラウンドでのルーター自動検出を開始してリクエストの待機を開始します。
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
