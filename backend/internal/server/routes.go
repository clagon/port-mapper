package server

import "github.com/labstack/echo/v4"

// registerRoutes は、Echo Web フレームワークインスタンスに対して API エンドポイントとSPA静的ファイルのルートを登録します。
func registerRoutes(e *echo.Echo, svc apiService) {
	h := newAPIHandlers(svc)
	e.GET("/api/health", h.health)
	e.GET("/api/status", h.status)
	e.POST("/api/discover", h.discover)
	e.POST("/api/ports/open", h.portsOpen)
	e.POST("/api/ports/close", h.portsClose)
	e.GET("/api/settings", h.getSettings)
	e.POST("/api/settings", h.updateSettings)
	e.GET("/*", staticHandler())
}
