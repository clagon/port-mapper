package server

import "net/http"

// NewMux builds the HTTP routes for the application.
func NewMux() http.Handler {
	h := newAPIHandlers()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", h.health)
	mux.HandleFunc("/api/status", h.status)
	mux.HandleFunc("/api/discover", h.discover)
	mux.HandleFunc("/api/ports/open", h.portsOpen)
	mux.HandleFunc("/api/ports/close", h.portsClose)
	mux.HandleFunc("/api/settings", h.settings)
	mux.Handle("/", staticHandler())
	return mux
}
