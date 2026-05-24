package server

import (
	"net"
	"net/http"
)

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

// ListenAndServe runs the server on its configured address.
func (s *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	return s.Serve(ln)
}

// Serve runs the server on the provided listener.
func (s *Server) Serve(ln net.Listener) error {
	return http.Serve(ln, s.Handler())
}
