package httprest

import (
	"time"

	"github.com/gorilla/mux"
)

// Option is a http rest server functional parameter type.
type Option func(g *Server) error

// WithHTTPAddr sets the full address ( host and port ) of the server.
func WithHTTPAddr(addr string) Option {
	return func(g *Server) error {
		g.cfg.httpAddr = addr
		return nil
	}
}

// WithRouter sets the internal router of the server, this is required.
func WithRouter(r *mux.Router) Option {
	return func(g *Server) error {
		g.cfg.router = r
		return nil
	}
}

// WithTimeout allows changing the timeout value for API calls.
func WithTimeout(duration time.Duration) Option {
	return func(g *Server) error {
		g.cfg.timeout = duration
		return nil
	}
}
