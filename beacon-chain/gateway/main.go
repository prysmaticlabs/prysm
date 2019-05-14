package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	joonix "github.com/joonix/log"
	"github.com/sirupsen/logrus"
)

// Endpoint describes a gRPC endpoint
type Endpoint struct {
	Network, Addr string
}

// Options is a set of options to be passed to Run
type Options struct {
	// Addr is the address to listen
	Addr string

	// GRPCServer defines an endpoint of a gRPC service
	GRPCServer Endpoint

	// SwaggerDir is a path to a directory from which the server
	// serves swagger specs.
	SwaggerDir string

	// Mux is a list of options to be passed to the grpc-gateway multiplexer
	Mux []gwruntime.ServeMuxOption
}

var (
	beaconRPC = flag.String("beacon-rpc", "localhost:4000", "Beacon chain gRPC endpoint")
	port      = flag.Int("port", 8000, "Port to serve on")
	debug     = flag.Bool("debug", false, "Enable debug logging")
)

func init() {
	logrus.SetFormatter(&joonix.FluentdFormatter{})
}

var log = logrus.New()

func main() {
	flag.Parse()
	if *debug {
		log.SetLevel(logrus.DebugLevel)
	}

	opts := Options{
		GRPCServer: Endpoint{
			Network: "tcp",
			Addr:    *beaconRPC,
		},
		Addr:       fmt.Sprintf("0.0.0.0:%d", *port),
		SwaggerDir: "proto/beacon/rpc/v1/",
	}
	if err := Run(context.Background(), opts); err != nil {
		panic(err)
	}
}

// TODO: Make this a prysm service.

// Run starts a HTTP server and blocks while running if successful.
// The server will be shutdown when "ctx" is canceled.
func Run(ctx context.Context, opts Options) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	conn, err := dial(ctx, opts.GRPCServer.Network, opts.GRPCServer.Addr)
	if err != nil {
		return err
	}
	go func() {
		<-ctx.Done()
		if err := conn.Close(); err != nil {
			log.Fatalf("Failed to close a client connection to the gRPC server: %v", err)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/swagger/", swaggerServer(opts.SwaggerDir))
	mux.HandleFunc("/healthz", healthzServer(conn))

	gw, err := newGateway(ctx, conn, opts.Mux)
	if err != nil {
		return err
	}
	mux.Handle("/", gw)

	s := &http.Server{
		Addr:    opts.Addr,
		Handler: mux,
	}
	go func() {
		<-ctx.Done()
		log.Info("Shutting down the http server")
		if err := s.Shutdown(context.Background()); err != nil {
			log.Fatalf("Failed to shutdown http server: %v", err)
		}
	}()

	log.Infof("Starting listening at %s", opts.Addr)
	if err := s.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Failed to listen and serve: %v", err)
		return err
	}
	return nil
}
