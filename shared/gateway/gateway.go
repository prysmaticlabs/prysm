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

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ shared.Service = (*Gateway)(nil)

// PbHandlerRegistration is a function that registers a protobuf handler.
type PbHandlerRegistration func(context.Context, *gwruntime.ServeMux, *grpc.ClientConn) error

// MuxHandler is a function that implements the mux handler functionality.
type MuxHandler func(http.Handler, http.ResponseWriter, *http.Request)

// Gateway is the gRPC gateway to serve HTTP JSON traffic as a proxy and forward it to the gRPC server.
type Gateway struct {
	conn                         *grpc.ClientConn
	v1Alpha1PbHandlers           []PbHandlerRegistration
	v1PbHandlers                 []PbHandlerRegistration
	muxHandler                   MuxHandler
	maxCallRecvMsgSize           uint64
	mux                          *http.ServeMux
	server                       *http.Server
	cancel                       context.CancelFunc
	remoteCert                   string
	gatewayAddr                  string
	apiMiddlewareAddr            string
	apiMiddlewareEndpointFactory EndpointFactory
	ctx                          context.Context
	startFailure                 error
	remoteAddr                   string
	allowedOrigins               []string
}

// New returns a new instance of the Gateway.
func New(
	ctx context.Context,
	v1Alpha1PbHandlers []PbHandlerRegistration,
	v1PbHandlers []PbHandlerRegistration,
	muxHandler MuxHandler,
	remoteAddr,
	gatewayAddress string,
) *Gateway {
	g := &Gateway{
		v1Alpha1PbHandlers: v1Alpha1PbHandlers,
		v1PbHandlers:       v1PbHandlers,
		muxHandler:         muxHandler,
		mux:                http.NewServeMux(),
		gatewayAddr:        gatewayAddress,
		ctx:                ctx,
		remoteAddr:         remoteAddr,
		allowedOrigins:     []string{},
	}
	return g
}

// WithMux allows adding a custom http.ServeMux to the gateway.
func (g *Gateway) WithMux(m *http.ServeMux) *Gateway {
	g.mux = m
	return g
}

// WithAllowedOrigins allows adding a set of allowed origins to the gateway.
func (g *Gateway) WithAllowedOrigins(origins []string) *Gateway {
	g.allowedOrigins = origins
	return g
}

// WithRemoteCert allows adding a custom certificate to the gateway,
func (g *Gateway) WithRemoteCert(cert string) *Gateway {
	g.remoteCert = cert
	return g
}

// WithMaxCallRecvMsgSize allows specifying the maximum allowed gRPC message size.
func (g *Gateway) WithMaxCallRecvMsgSize(size uint64) *Gateway {
	g.maxCallRecvMsgSize = size
	return g
}

// WithApiMiddleware allows adding API Middleware proxy to the gateway.
func (g *Gateway) WithApiMiddleware(address string, endpointFactory EndpointFactory) *Gateway {
	g.apiMiddlewareAddr = address
	g.apiMiddlewareEndpointFactory = endpointFactory
	return g
}

// Start the gateway service.
func (g *Gateway) Start() {
	ctx, cancel := context.WithCancel(g.ctx)
	g.cancel = cancel

	conn, err := g.dial(ctx, "tcp", g.remoteAddr)
	if err != nil {
		log.WithError(err).Error("Failed to connect to gRPC server")
		g.startFailure = err
		return
	}
	g.conn = conn

	v1Alpha1Mux := gwruntime.NewServeMux(
		gwruntime.WithMarshalerOption(gwruntime.MIMEWildcard, &gwruntime.HTTPBodyMarshaler{
			Marshaler: &gwruntime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					EmitUnpopulated: true,
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					DiscardUnknown: true,
				},
			},
		}),
		gwruntime.WithMarshalerOption(
			"text/event-stream", &gwruntime.EventSourceJSONPb{},
		),
	)

	v1Mux := gwruntime.NewServeMux(
		gwruntime.WithMarshalerOption(gwruntime.MIMEWildcard, &gwruntime.HTTPBodyMarshaler{
			Marshaler: &gwruntime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					UseProtoNames:   true,
					EmitUnpopulated: true,
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					DiscardUnknown: true,
				},
			},
		}),
	)

	for _, h := range g.v1Alpha1PbHandlers {
		if err := h(ctx, v1Alpha1Mux, g.conn); err != nil {
			log.WithError(err).Error("Failed to start v1alpha1 gateway")
			g.startFailure = err
			return
		}
	}
	for _, h := range g.v1PbHandlers {
		if err := h(ctx, v1Mux, g.conn); err != nil {
			log.WithError(err).Error("Failed to start v1 gateway")
			g.startFailure = err
			return
		}
	}

	g.mux.Handle("/eth/v1alpha1/", v1Alpha1Mux)
	g.mux.Handle("/eth/v1/", v1Mux)
	apiHandler := g.corsMiddleware(g.mux)

	g.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		g.muxHandler(apiHandler, w, r)
	})

	g.server = &http.Server{
		Addr:    g.gatewayAddr,
		Handler: g.mux,
	}

	go func() {
		log.WithField("address", g.gatewayAddr).Info("Starting gRPC gateway")
		if err := g.server.ListenAndServe(); err != http.ErrServerClosed {
			log.WithError(err).Error("Failed to start gRPC gateway")
			g.startFailure = err
			return
		}
	}()

	if g.apiMiddlewareAddr != "" && g.apiMiddlewareEndpointFactory != nil {
		go g.registerApiMiddleware()
	}
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
		AllowedOrigins:   g.allowedOrigins,
		AllowedMethods:   []string{http.MethodPost, http.MethodGet, http.MethodOptions},
		AllowCredentials: true,
		MaxAge:           600,
		AllowedHeaders:   []string{"*"},
	})
	return c.Handler(h)
}

const swaggerDir = "proto/beacon/rpc/v1/"

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
	if len(g.remoteCert) > 0 {
		creds, err := credentials.NewClientTLSFromFile(g.remoteCert, "")
		if err != nil {
			return nil, err
		}
		security = grpc.WithTransportCredentials(creds)
	}
	opts := []grpc.DialOption{
		security,
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(int(g.maxCallRecvMsgSize))),
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
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(int(g.maxCallRecvMsgSize))),
	}
	return grpc.DialContext(ctx, addr, opts...)
}

func (g *Gateway) registerApiMiddleware() {
	proxy := &ApiProxyMiddleware{
		GatewayAddress:  g.gatewayAddr,
		ProxyAddress:    g.apiMiddlewareAddr,
		EndpointCreator: g.apiMiddlewareEndpointFactory,
	}
	log.WithField("API middleware address", g.apiMiddlewareAddr).Info("Starting API middleware")
	if err := proxy.Run(); err != http.ErrServerClosed {
		log.WithError(err).Error("Failed to start API middleware")
		g.startFailure = err
		return
	}
}
