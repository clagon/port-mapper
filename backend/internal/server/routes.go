package server

import "github.com/labstack/echo/v4"

// registerRoutes builds the HTTP routes for the application.
func registerRoutes(e *echo.Echo, svc *service) {
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
