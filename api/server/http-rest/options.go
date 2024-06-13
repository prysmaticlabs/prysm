package http_rest

import (
	"time"

	"github.com/gorilla/mux"
)

type Option func(g *Server) error

func WithMuxHandler(m restHandler) Option {
	return func(g *Server) error {
		g.cfg.muxHandler = m
		return nil
	}
}

func WithHTTPAddr(addr string) Option {
	return func(g *Server) error {
		g.cfg.httpAddr = addr
		return nil
	}
}

// WithRouter allows adding a custom mux router to the gateway.
func WithRouter(r *mux.Router) Option {
	return func(g *Server) error {
		g.cfg.router = r
		return nil
	}
}

// WithAllowedOrigins allows adding a set of allowed origins to the gateway.
func WithAllowedOrigins(origins []string) Option {
	return func(g *Server) error {
		g.cfg.allowedOrigins = origins
		return nil
	}
}

// WithTimeout allows changing the timeout value for API calls.
func WithTimeout(seconds uint64) Option {
	return func(g *Server) error {
		g.cfg.timeout = time.Second * time.Duration(seconds)
		return nil
	}
}
