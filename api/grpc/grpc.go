// Package gateway defines a grpc-gateway server that serves HTTP-JSON traffic and acts a proxy between HTTP and gRPC.
package grpc

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/api/server/middleware"
	"github.com/prysmaticlabs/prysm/v5/runtime"
)

var _ runtime.Service = (*Server)(nil)

// MuxHandler is a function that implements the mux handler functionality.
type MuxHandler func(
	h http.HandlerFunc,
	w http.ResponseWriter,
	req *http.Request,
)

// Config parameters for setting up the gateway service.
type config struct {
	gatewayAddr    string
	allowedOrigins []string
	muxHandler     MuxHandler
	router         *mux.Router
	timeout        time.Duration
}

// Server is the gRPC gateway to serve HTTP JSON traffic as a proxy and forward it to the gRPC server.
type Server struct {
	cfg          *config
	server       *http.Server
	cancel       context.CancelFunc
	ctx          context.Context
	startFailure error
}

// New returns a new instance of the Server.
func New(ctx context.Context, opts ...Option) (*Server, error) {
	g := &Server{
		ctx: ctx,
		cfg: &config{},
	}
	for _, opt := range opts {
		if err := opt(g); err != nil {
			return nil, err
		}
	}
	// TODO: this is a codesmell ( we should always have a router here)
	if g.cfg.router == nil {
		g.cfg.router = mux.NewRouter()
	}

	corsMux := middleware.CorsHandler(g.cfg.allowedOrigins).Middleware(g.cfg.router)
	// TODO: actually use the timeout config provided
	g.server = &http.Server{
		Addr:              g.cfg.gatewayAddr,
		Handler:           corsMux,
		ReadHeaderTimeout: time.Second,
	}
	if g.cfg.muxHandler != nil { // rest APIS and Web UI registration
		g.cfg.router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			g.cfg.muxHandler(corsMux.ServeHTTP, w, r)
		})
	}
	return g, nil
}

// Start the gateway service.
func (g *Server) Start() {
	_, cancel := context.WithCancel(g.ctx)
	g.cancel = cancel

	go func() {
		log.WithField("address", g.cfg.gatewayAddr).Info("Starting gRPC gateway")
		if err := g.server.ListenAndServe(); err != http.ErrServerClosed {
			log.WithError(err).Error("Failed to start gRPC gateway")
			g.startFailure = err
			return
		}
	}()
}

// Status of grpc gateway. Returns an error if this service is unhealthy.
func (g *Server) Status() error {
	if g.startFailure != nil {
		return g.startFailure
	}
	return nil
}

// Stop the gateway with a graceful shutdown.
func (g *Server) Stop() error {
	if g.server != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(g.ctx, 2*time.Second)
		defer shutdownCancel()
		if err := g.server.Shutdown(shutdownCtx); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				log.Warn("Existing connections terminated")
			} else {
				log.WithError(err).Error("Failed to gracefully shut down server")
			}
		}
	}
	if g.cancel != nil {
		g.cancel()
	}
	return nil
}
