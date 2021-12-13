// Package gateway defines a grpc-gateway server that serves HTTP-JSON traffic and acts a proxy between HTTP and gRPC.
package gateway

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gorilla/mux"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/api/gateway/apimiddleware"
	"github.com/prysmaticlabs/prysm/runtime"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
)

var _ runtime.Service = (*Gateway)(nil)

// PbMux serves grpc-gateway requests for selected patterns using registered protobuf handlers.
type PbMux struct {
	Registrations []PbHandlerRegistration // Protobuf registrations to be registered in Mux.
	Patterns      []string                // URL patterns that will be handled by Mux.
	Mux           *gwruntime.ServeMux     // The router that will be used for grpc-gateway requests.
}

// PbHandlerRegistration is a function that registers a protobuf handler.
type PbHandlerRegistration func(context.Context, *gwruntime.ServeMux, *grpc.ClientConn) error

// MuxHandler is a function that implements the mux handler functionality.
type MuxHandler func(
	apiMiddlewareHandler *apimiddleware.ApiProxyMiddleware,
	h http.HandlerFunc,
	w http.ResponseWriter,
	req *http.Request,
)

// Config parameters for setting up the gateway service.
type config struct {
	maxCallRecvMsgSize           uint64
	remoteCert                   string
	gatewayAddr                  string
	remoteAddr                   string
	allowedOrigins               []string
	apiMiddlewareEndpointFactory apimiddleware.EndpointFactory
	muxHandler                   MuxHandler
	pbHandlers                   []*PbMux
	router                       *mux.Router
}

// Gateway is the gRPC gateway to serve HTTP JSON traffic as a proxy and forward it to the gRPC server.
type Gateway struct {
	cfg          *config
	conn         *grpc.ClientConn
	server       *http.Server
	cancel       context.CancelFunc
	proxy        *apimiddleware.ApiProxyMiddleware
	ctx          context.Context
	startFailure error
}

// New returns a new instance of the Gateway.
func New(ctx context.Context, opts ...Option) (*Gateway, error) {
	g := &Gateway{
		ctx: ctx,
		cfg: &config{
			router: mux.NewRouter(),
		},
	}
	for _, opt := range opts {
		if err := opt(g); err != nil {
			return nil, err
		}
	}
	return g, nil
}

// Start the gateway service.
func (g *Gateway) Start() {
	ctx, cancel := context.WithCancel(g.ctx)
	g.cancel = cancel

	conn, err := g.dial(ctx, "tcp", g.cfg.remoteAddr)
	if err != nil {
		log.WithError(err).Error("Failed to connect to gRPC server")
		g.startFailure = err
		return
	}
	g.conn = conn

	for _, h := range g.cfg.pbHandlers {
		for _, r := range h.Registrations {
			if err := r(ctx, h.Mux, g.conn); err != nil {
				log.WithError(err).Error("Failed to register handler")
				g.startFailure = err
				return
			}
		}
		for _, p := range h.Patterns {
			g.cfg.router.PathPrefix(p).Handler(h.Mux)
		}
	}

	corsMux := g.corsMiddleware(g.cfg.router)

	if g.cfg.apiMiddlewareEndpointFactory != nil && !g.cfg.apiMiddlewareEndpointFactory.IsNil() {
		g.registerApiMiddleware()
	}

	if g.cfg.muxHandler != nil {
		g.cfg.router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			g.cfg.muxHandler(g.proxy, corsMux.ServeHTTP, w, r)
		})
	}

	g.server = &http.Server{
		Addr:    g.cfg.gatewayAddr,
		Handler: corsMux,
	}

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
func (g *Gateway) Status() error {
	if g.startFailure != nil {
		return g.startFailure
	}
	if s := g.conn.GetState(); s != connectivity.Ready {
		return fmt.Errorf("grpc server is %s", s)
	}
	return nil
}

// Stop the gateway with a graceful shutdown.
func (g *Gateway) Stop() error {
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

func (g *Gateway) corsMiddleware(h http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   g.cfg.allowedOrigins,
		AllowedMethods:   []string{http.MethodPost, http.MethodGet, http.MethodDelete, http.MethodOptions},
		AllowCredentials: true,
		MaxAge:           600,
		AllowedHeaders:   []string{"*"},
	})
	return c.Handler(h)
}

const swaggerDir = "proto/prysm/v1alpha1/"

// SwaggerServer returns swagger specification files located under "/swagger/"
func SwaggerServer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ".swagger.json") {
			log.Debugf("Not found: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		log.Debugf("Serving %s\n", r.URL.Path)
		p := strings.TrimPrefix(r.URL.Path, "/swagger/")
		p = path.Join(swaggerDir, p)
		http.ServeFile(w, r, p)
	}
}

// dial the gRPC server.
func (g *Gateway) dial(ctx context.Context, network, addr string) (*grpc.ClientConn, error) {
	switch network {
	case "tcp":
		return g.dialTCP(ctx, addr)
	case "unix":
		return g.dialUnix(ctx, addr)
	default:
		return nil, fmt.Errorf("unsupported network type %q", network)
	}
}

// dialTCP creates a client connection via TCP.
// "addr" must be a valid TCP address with a port number.
func (g *Gateway) dialTCP(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	security := grpc.WithInsecure()
	if len(g.cfg.remoteCert) > 0 {
		creds, err := credentials.NewClientTLSFromFile(g.cfg.remoteCert, "")
		if err != nil {
			return nil, err
		}
		security = grpc.WithTransportCredentials(creds)
	}
	opts := []grpc.DialOption{
		security,
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(int(g.cfg.maxCallRecvMsgSize))),
	}

	return grpc.DialContext(ctx, addr, opts...)
}

// dialUnix creates a client connection via a unix domain socket.
// "addr" must be a valid path to the socket.
func (g *Gateway) dialUnix(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	d := func(addr string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout("unix", addr, timeout)
	}
	f := func(ctx context.Context, addr string) (net.Conn, error) {
		if deadline, ok := ctx.Deadline(); ok {
			return d(addr, time.Until(deadline))
		}
		return d(addr, 0)
	}
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithContextDialer(f),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(int(g.cfg.maxCallRecvMsgSize))),
	}
	return grpc.DialContext(ctx, addr, opts...)
}

func (g *Gateway) registerApiMiddleware() {
	g.proxy = &apimiddleware.ApiProxyMiddleware{
		GatewayAddress:  g.cfg.gatewayAddr,
		EndpointCreator: g.cfg.apiMiddlewareEndpointFactory,
	}
	log.Info("Starting API middleware")
	g.proxy.Run(g.cfg.router)
}
