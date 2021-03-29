package gateway

import (
	"context"
	"net/http"
	"strings"
	"time"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	pb "github.com/prysmaticlabs/prysm/proto/validator/accounts/v2_gateway"
	"github.com/prysmaticlabs/prysm/validator/web"
	"github.com/rs/cors"
	"google.golang.org/grpc"
)

// Gateway is the gRPC gateway to serve HTTP JSON traffic as a
// proxy and forward it to the gRPC server.
type Gateway struct {
	ctx            context.Context
	cancel         context.CancelFunc
	gatewayAddr    string
	remoteAddr     string
	server         *http.Server
	mux            *http.ServeMux
	allowedOrigins []string
	startFailure   error
}

// New returns a new gateway server which translates HTTP into gRPC.
// Accepts a context and optional http.ServeMux.
func New(
	ctx context.Context,
	remoteAddress,
	gatewayAddress string,
	allowedOrigins []string,
) *Gateway {
	return &Gateway{
		remoteAddr:     remoteAddress,
		gatewayAddr:    gatewayAddress,
		ctx:            ctx,
		mux:            http.NewServeMux(),
		allowedOrigins: allowedOrigins,
	}
}

// Start the gateway service. This serves the HTTP JSON traffic.
func (g *Gateway) Start() {
	ctx, cancel := context.WithCancel(g.ctx)
	g.cancel = cancel

	gwmux := gwruntime.NewServeMux(
		gwruntime.WithMarshalerOption(
			gwruntime.MIMEWildcard,
			&gwruntime.JSONPb{OrigName: false},
		),
	)
	opts := []grpc.DialOption{grpc.WithInsecure()}
	handlers := []func(context.Context, *gwruntime.ServeMux, string, []grpc.DialOption) error{
		pb.RegisterAuthHandlerFromEndpoint,
		pb.RegisterWalletHandlerFromEndpoint,
		pb.RegisterHealthHandlerFromEndpoint,
		pb.RegisterAccountsHandlerFromEndpoint,
		pb.RegisterBeaconHandlerFromEndpoint,
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

	return nil
}

// Stop the gateway with a graceful shutdown.
func (g *Gateway) Stop() error {
	shutdownCtx, shutdownCancel := context.WithTimeout(g.ctx, 2*time.Second)
	defer shutdownCancel()
	if err := g.server.Shutdown(shutdownCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Warn("Existing connections terminated")
		} else {
			log.WithError(err).Error("Failed to gracefully shut down server")
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
