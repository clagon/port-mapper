package server

import "net/http"

// Server wraps the HTTP handler used by the application.
type Server struct {
	addr    string
	handler http.Handler
}

// New constructs a server bound to the provided listen address.
func New(addr string) *Server {
	s := &Server{addr: addr}
	s.handler = NewMux()
	return s
}

// Addr returns the configured listen address.
func (s *Server) Addr() string {
	if s == nil {
		return ""
	}
	return s.addr
}

// Handler returns the server's HTTP handler.
func (s *Server) Handler() http.Handler {
	if s == nil {
		return http.NewServeMux()
	}
	if s.handler == nil {
		s.handler = NewMux()
	}
	return s.handler
}

// ListenAndServe runs the server with the provided HTTP server.
func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(s.addr, s.Handler())
}
