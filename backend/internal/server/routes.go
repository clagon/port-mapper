package server

import "github.com/labstack/echo/v4"

// registerRoutes は、Echo Web フレームワークインスタンスに対して API エンドポイントとSPA静的ファイルのルートを登録します。
func registerRoutes(e *echo.Echo, svc apiService, browserToken, listenAddr string) {
	h := newAPIHandlers(svc)
	requireBrowserToken := browserTokenMiddleware(browserToken, listenAddr)
	e.GET("/api/health", h.health)
	e.GET("/api/status", h.status)
	e.POST("/api/discover", h.discover, requireBrowserToken)
	e.POST("/api/ports/open", h.portsOpen, requireBrowserToken)
	e.POST("/api/ports/close", h.portsClose, requireBrowserToken)
	e.GET("/api/settings", h.getSettings)
	e.POST("/api/settings", h.updateSettings, requireBrowserToken)
	e.GET("/*", staticHandler(browserToken))
}
