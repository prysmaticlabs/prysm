package httprest

import (
	"time"

	"net/http"

	"github.com/prysmaticlabs/prysm/v5/api/server/middleware"
)

// Option is a http rest server functional parameter type.
type Option func(g *Server) error

func WithMiddlewares(mw []middleware.Middleware) Option {
	return func(g *Server) error {
		g.cfg.middlewares = mw
		return nil
	}
}

// WithHTTPAddr sets the full address ( host and port ) of the server.
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
