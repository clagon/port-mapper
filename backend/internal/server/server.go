// Package server は、PortoのHTTP APIサーバーおよびSPA用静的ファイルのルーティングとハンドラーを提供します。
package server

import (
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// Server は、アプリケーションで使用される HTTP ルーターとリスナー設定をカプセル化する構造体です。
type Server struct {
	addr   string
	echo   *echo.Echo
	logger *slog.Logger
}

// New は、指定されたリスンアドレス、ロガー、および抽象サービスインスタンスを使用して HTTP サーバーインスタンスを構築します。
func New(addr string, logger *slog.Logger, svc apiService) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(loggingMiddleware(logger))
	registerRoutes(e, svc)

	return &Server{addr: addr, echo: e, logger: logger}
}

// Addr は、サーバーがバインドされている、またはバインド予定のリスンアドレスを返します。
func (s *Server) Addr() string {
	if s == nil {
		return ""
	}
	return s.addr
}

// Handler は、標準の http.Handler として機能するサーバーの HTTP ハンドラーを返します。
func (s *Server) Handler() http.Handler {
	if s == nil || s.echo == nil {
		return http.NewServeMux()
	}
	return s.echo
}

// ListenAndServe は、設定されたアドレスにバインドし、同期的に HTTP サーバーを起動します。
func (s *Server) ListenAndServe() error {
	if s == nil || s.echo == nil {
		return nil
	}
	return s.echo.Start(s.addr)
}

// Serve は、提供された既存の net.Listener オブジェクトを利用して HTTP サーバーを起動します（テストやポート自動割り当てで有用）。
func (s *Server) Serve(ln net.Listener) error {
	if s == nil || s.echo == nil {
		return nil
	}
	return (&http.Server{Handler: s.echo, ReadHeaderTimeout: 5 * time.Second}).Serve(ln)
}

func loggingMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			started := time.Now()
			err := next(c)
			status := c.Response().Status
			if status == 0 {
				status = http.StatusOK
			}
			if err != nil && status < http.StatusBadRequest {
				status = http.StatusInternalServerError
			}
			logger.Info("request",
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"status", status,
				"bytes", c.Response().Size,
				"duration", time.Since(started).String(),
			)
			return err
		}
	}
}
