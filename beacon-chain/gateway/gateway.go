package gateway

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/prysmaticlabs/prysm/shared"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

var _ = shared.Service(&Gateway{})

// Gateway is the gRPC gateway to serve HTTP JSON traffic as a proxy and forward
// it to the beacon-chain gRPC server.
type Gateway struct {
	conn        *grpc.ClientConn
	ctx         context.Context
	cancel      context.CancelFunc
	gatewayAddr string
	remoteAddr  string
	server      *http.Server
	mux         *http.ServeMux

	startFailure error
}

// Start the gateway service. This serves the HTTP JSON traffic on the specified
// port.
func (g *Gateway) Start() {
	ctx, cancel := context.WithCancel(g.ctx)
	g.cancel = cancel

	log.WithField("address", g.gatewayAddr).Info("Starting gRPC gateway.")

	conn, err := dial(ctx, "tcp", g.remoteAddr)
	if err != nil {
		log.WithError(err).Error("Failed to connect to gRPC server")
		g.startFailure = err
		return
	}

	g.conn = conn

	gwmux := gwruntime.NewServeMux()
	for _, f := range []func(context.Context, *gwruntime.ServeMux, *grpc.ClientConn) error{} {
		if err := f(ctx, gwmux, conn); err != nil {
			log.WithError(err).Error("Failed to start gateway")
			g.startFailure = err
			return
		}
	}

	g.mux.Handle("/", gwmux)

	g.server = &http.Server{
		Addr:    g.gatewayAddr,
		Handler: g.mux,
	}
	go func() {
		if err := g.server.ListenAndServe(); err != http.ErrServerClosed {
			log.WithError(err).Error("Failed to listen and serve")
			g.startFailure = err
			return
		}
	}()

	return
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
	if err := g.server.Shutdown(g.ctx); err != nil {
		log.WithError(err).Error("Failed to shut down server")
	}

	if g.cancel != nil {
		g.cancel()
	}

	return nil
}

// New returns a new gateway server which translates HTTP into gRPC.
// Accepts a context and optional http.ServeMux.
func New(ctx context.Context, remoteAddress, gatewayAddress string, mux *http.ServeMux) *Gateway {
	if mux == nil {
		mux = http.NewServeMux()
	}

	return &Gateway{
		remoteAddr:  remoteAddress,
		gatewayAddr: gatewayAddress,
		ctx:         ctx,
		mux:         mux,
	}
}

// dial the gRPC server.
func dial(ctx context.Context, network, addr string) (*grpc.ClientConn, error) {
	switch network {
	case "tcp":
		return dialTCP(ctx, addr)
	case "unix":
		return dialUnix(ctx, addr)
	default:
		return nil, fmt.Errorf("unsupported network type %q", network)
	}
}

// dialTCP creates a client connection via TCP.
// "addr" must be a valid TCP address with a port number.
func dialTCP(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, addr, grpc.WithInsecure())
}

// dialUnix creates a client connection via a unix domain socket.
// "addr" must be a valid path to the socket.
func dialUnix(ctx context.Context, addr string) (*grpc.ClientConn, error) {
	d := func(addr string, timeout time.Duration) (net.Conn, error) {
		return net.DialTimeout("unix", addr, timeout)
	}
	return grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithDialer(d))
}
