// Package gateway defines a gRPC gateway to serve HTTP-JSON
// traffic as a proxy and forward it to a beacon or validator node's gRPC service.
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
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2"
	"github.com/prysmaticlabs/prysm/shared"
	"github.com/prysmaticlabs/prysm/validator/web"
	"github.com/rs/cors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ shared.Service = (*Gateway)(nil)

// CallerId defines whether the caller node is a beacon
// or a validator node. This helps register the handlers accordingly.
type CallerId uint8

const (
	Beacon CallerId = iota
	Validator
)

// Gateway is the gRPC gateway to serve HTTP JSON traffic as a
// proxy and forward it to the gRPC server.
type Gateway struct {
	conn                    *grpc.ClientConn
	enableDebugRPCEndpoints bool
	callerId                CallerId
	maxCallRecvMsgSize      uint64
	mux                     *http.ServeMux
	server                  *http.Server
	cancel                  context.CancelFunc
	remoteCert              string
	gatewayAddr             string
	ctx                     context.Context
	startFailure            error
	remoteAddr              string
	allowedOrigins          []string
}

// NewValidator returns a new gateway server which translates HTTP into gRPC.
// Accepts a context.
func NewValidator(
	ctx context.Context,
	remoteAddress,
	gatewayAddress string,
	allowedOrigins []string,
) *Gateway {
	return &Gateway{
		callerId:       Validator,
		remoteAddr:     remoteAddress,
		gatewayAddr:    gatewayAddress,
		ctx:            ctx,
		mux:            http.NewServeMux(),
		allowedOrigins: allowedOrigins,
	}
}

// NewBeacon returns a new gateway server which translates HTTP into gRPC.
// Accepts a context and optional http.ServeMux.
func NewBeacon(
	ctx context.Context,
	remoteAddress,
	remoteCert,
	gatewayAddress string,
	mux *http.ServeMux,
	allowedOrigins []string,
	enableDebugRPCEndpoints bool,
	maxCallRecvMsgSize uint64,
) *Gateway {
	if mux == nil {
		mux = http.NewServeMux()
	}

	return &Gateway{
		callerId:                Beacon,
		remoteAddr:              remoteAddress,
		remoteCert:              remoteCert,
		gatewayAddr:             gatewayAddress,
		ctx:                     ctx,
		mux:                     mux,
		allowedOrigins:          allowedOrigins,
		enableDebugRPCEndpoints: enableDebugRPCEndpoints,
		maxCallRecvMsgSize:      maxCallRecvMsgSize,
	}
}

// Start the gateway service. This serves the HTTP JSON traffic.
// The beacon node supports TCP and Unix domain socket communications.
// Beacon and validator have different handlers.
func (g *Gateway) Start() {
	ctx, cancel := context.WithCancel(g.ctx)
	g.cancel = cancel

	if g.callerId == Beacon {
		conn, err := g.dial(ctx, "tcp", g.remoteAddr)
		if err != nil {
			log.WithError(err).Error("Failed to connect to gRPC server")
			g.startFailure = err
			return
		}

		g.conn = conn
	}
	gwmux := gwruntime.NewServeMux(
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
	if g.callerId == Beacon {
		handlers := []func(context.Context, *gwruntime.ServeMux, *grpc.ClientConn) error{
			ethpb.RegisterNodeHandler,
			ethpb.RegisterBeaconChainHandler,
			ethpb.RegisterBeaconNodeValidatorHandler,
			pbrpc.RegisterHealthHandler,
		}
		if g.enableDebugRPCEndpoints {
			handlers = append(handlers, pbrpc.RegisterDebugHandler)
		}
		for _, f := range handlers {
			if err := f(ctx, gwmux, g.conn); err != nil {
				log.WithError(err).Error("Failed to start gateway")
				g.startFailure = err
				return
			}
		}

		g.mux.Handle("/", gwmux)
		g.server = &http.Server{
			Addr:    g.gatewayAddr,
			Handler: g.corsMiddleware(g.mux),
		}

	} else {
		opts := []grpc.DialOption{grpc.WithInsecure()}
		handlers := []func(context.Context, *gwruntime.ServeMux, string, []grpc.DialOption) error{
			pb.RegisterAuthHandlerFromEndpoint,
			pb.RegisterWalletHandlerFromEndpoint,
			pb.RegisterHealthHandlerFromEndpoint,
			pb.RegisterAccountsHandlerFromEndpoint,
			pb.RegisterBeaconHandlerFromEndpoint,
			pb.RegisterSlashingProtectionHandlerFromEndpoint,
		}
		for _, h := range handlers {
			if err := h(ctx, gwmux, g.remoteAddr, opts); err != nil {
				log.Fatalf("Could not register API handler with grpc endpoint: %v", err)
			}
		}
		apiHandler := g.corsMiddleware(gwmux)
		g.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api") {
				http.StripPrefix("/api", apiHandler).ServeHTTP(w, r)
			} else {
				web.Handler(w, r)
			}
		})
		g.server = &http.Server{
			Addr:    g.gatewayAddr,
			Handler: g.mux,
		}
	}

	go func() {
		log.WithField("address", g.gatewayAddr).Info("Starting gRPC gateway")
		if err := g.server.ListenAndServe(); err != http.ErrServerClosed {
			log.WithError(err).Error("Failed to listen and serve")
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

	return grpc.DialContext(
		ctx,
		addr,
		opts...,
	)
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
