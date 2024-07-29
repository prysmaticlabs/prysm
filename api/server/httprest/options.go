package httprest

import (
	"time"

	"net/http"
)

type Option func(g *Server) error

func WithMuxHandler(m httpHandler) Option {
	return func(g *Server) error {
		g.cfg.handler = m
		return nil
	}
}

func WithHTTPAddr(addr string) Option {
	return func(g *Server) error {
		g.cfg.httpAddr = addr
		return nil
	}
}

// WithRouter --.
func WithRouter(r *http.ServeMux) Option {
	return func(g *Server) error {
		g.cfg.router = r
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
