// Package gateway defines a gRPC gateway to serve HTTP-JSON
// traffic as a proxy and forward it to a beacon node's gRPC service.
package gateway

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1_gateway"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1_gateway"
	"github.com/prysmaticlabs/prysm/shared"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

var _ = shared.Service(&Gateway{})

// Gateway is the gRPC gateway to serve HTTP JSON traffic as a proxy and forward
// it to the beacon-chain gRPC server.
type Gateway struct {
	conn                    *grpc.ClientConn
	ctx                     context.Context
	cancel                  context.CancelFunc
	gatewayAddr             string
	remoteAddr              string
	server                  *http.Server
	mux                     *http.ServeMux
	allowedOrigins          []string
	startFailure            error
	enableDebugRPCEndpoints bool
	maxCallRecvMsgSize      uint64
}

// Start the gateway service. This serves the HTTP JSON traffic on the specified
// port.
func (g *Gateway) Start() {
	ctx, cancel := context.WithCancel(g.ctx)
	g.cancel = cancel

	log.WithField("address", g.gatewayAddr).Info("Starting JSON-HTTP API")

	conn, err := g.dial(ctx, "tcp", g.remoteAddr)
	if err != nil {
		log.WithError(err).Error("Failed to connect to gRPC server")
		g.startFailure = err
		return
	}

	g.conn = conn

	gwmux := gwruntime.NewServeMux(
		gwruntime.WithMarshalerOption(
			gwruntime.MIMEWildcard,
			&gwruntime.JSONPb{OrigName: false, EmitDefaults: true},
		),
	)
	handlers := []func(context.Context, *gwruntime.ServeMux, *grpc.ClientConn) error{
		ethpb.RegisterNodeHandler,
		ethpb.RegisterBeaconChainHandler,
		ethpb.RegisterBeaconNodeValidatorHandler,
	}
	if g.enableDebugRPCEndpoints {
		handlers = append(handlers, pbrpc.RegisterDebugHandler)
	}
	for _, f := range handlers {
		if err := f(ctx, gwmux, conn); err != nil {
			log.WithError(err).Error("Failed to start gateway")
			g.startFailure = err
			return
		}
	}

	g.mux.Handle("/", gwmux)

	g.server = &http.Server{
		Addr:    g.gatewayAddr,
		Handler: newCorsHandler(g.mux, g.allowedOrigins),
	}
	go func() {
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
		if err := g.server.Shutdown(g.ctx); err != nil {
			log.WithError(err).Error("Failed to shut down server")
		}
	}

	if g.cancel != nil {
		g.cancel()
	}

	return nil
}

// New returns a new gateway server which translates HTTP into gRPC.
// Accepts a context and optional http.ServeMux.
func New(
	ctx context.Context,
	remoteAddress,
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
		remoteAddr:              remoteAddress,
		gatewayAddr:             gatewayAddress,
		ctx:                     ctx,
		mux:                     mux,
		allowedOrigins:          allowedOrigins,
		enableDebugRPCEndpoints: enableDebugRPCEndpoints,
		maxCallRecvMsgSize:      maxCallRecvMsgSize,
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
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
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
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDialer(d),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(int(g.maxCallRecvMsgSize))),
	}
	return grpc.DialContext(ctx, addr, opts...)
}
