// Package main allows for creation of an HTTP-JSON to gRPC
// gateway as a binary go process.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strings"

	joonix "github.com/joonix/log"
	"github.com/prysmaticlabs/prysm/beacon-chain/gateway"
	"github.com/sirupsen/logrus"
	_ "github.com/prysmaticlabs/prysm/shared/maxprocs"
)

var (
	beaconRPC               = flag.String("beacon-rpc", "localhost:4000", "Beacon chain gRPC endpoint")
	port                    = flag.Int("port", 8000, "Port to serve on")
	host                    = flag.String("host", "127.0.0.1", "Host to serve on")
	debug                   = flag.Bool("debug", false, "Enable debug logging")
	allowedOrigins          = flag.String("corsdomain", "", "A comma separated list of CORS domains to allow")
	enableDebugRPCEndpoints = flag.Bool("enable-debug-rpc-endpoints", false, "Enable debug rpc endpoints such as /eth/v1alpha1/beacon/state")
	grpcMaxMsgSize          = flag.Int("grpc-max-msg-size", 1<<22, "Integer to define max recieve message call size")
)

func init() {
	logrus.SetFormatter(joonix.NewFormatter())
}

var log = logrus.New()

func main() {
	flag.Parse()
	if *debug {
		log.SetLevel(logrus.DebugLevel)
	}

	mux := http.NewServeMux()
	gw := gateway.New(
		context.Background(),
		*beaconRPC,
		fmt.Sprintf("%s:%d", *host, *port),
		mux,
		strings.Split(*allowedOrigins, ","),
		*enableDebugRPCEndpoints,
		uint64(*grpcMaxMsgSize),
	)
	mux.HandleFunc("/swagger/", gateway.SwaggerServer())
	mux.HandleFunc("/healthz", healthzServer(gw))
	gw.Start()

	select {}
}

// healthzServer returns a simple health handler which returns ok.
func healthzServer(gw *gateway.Gateway) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if err := gw.Status(); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if _, err := fmt.Fprintln(w, "ok"); err != nil {
			log.WithError(err).Error("failed to respond to healthz")
		}
	}
}
