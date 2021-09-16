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
	beaconGateway "github.com/prysmaticlabs/prysm/beacon-chain/gateway"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/shared/gateway"
	_ "github.com/prysmaticlabs/prysm/shared/maxprocs"
	"github.com/sirupsen/logrus"
)

var (
	beaconRPC               = flag.String("beacon-rpc", "localhost:4000", "Beacon chain gRPC endpoint")
	port                    = flag.Int("port", 8000, "Port to serve on")
	ethApiPort              = flag.Int("port", 8001, "Port to serve Ethereum API on")
	host                    = flag.String("host", "127.0.0.1", "Host to serve on")
	debug                   = flag.Bool("debug", false, "Enable debug logging")
	allowedOrigins          = flag.String("corsdomain", "localhost:4242", "A comma separated list of CORS domains to allow")
	enableDebugRPCEndpoints = flag.Bool("enable-debug-rpc-endpoints", false, "Enable debug rpc endpoints such as /eth/v1alpha1/beacon/state")
	grpcMaxMsgSize          = flag.Int("grpc-max-msg-size", 1<<22, "Integer to define max recieve message call size")
	disableHTTPPrysmAPI     = flag.Bool("disable-http-prysm-api", false, "Disable all HTTP endpoints of Prysm API")
	disableHTTPEthAPI       = flag.Bool("disable-http-eth-api", false, "Disable all HTTP endpoints of the official Ethereum API")
)

func init() {
	logrus.SetFormatter(joonix.NewFormatter())
}

func main() {
	flag.Parse()
	if *debug {
		log.SetLevel(logrus.DebugLevel)
	}

	gatewayConfig := beaconGateway.DefaultConfig(*enableDebugRPCEndpoints, !(*disableHTTPPrysmAPI), !(*disableHTTPEthAPI))
	muxs := make([]*gateway.PbMux, 0)
	if gatewayConfig.V1Alpha1PbMux != nil {
		muxs = append(muxs, gatewayConfig.V1Alpha1PbMux)
	}
	if gatewayConfig.V1PbMux != nil {
		muxs = append(muxs, gatewayConfig.V1PbMux)
	}

	gw := gateway.New(context.Background(), muxs, gatewayConfig.Handler, *beaconRPC, fmt.Sprintf("%s:%d", *host, *port)).
		WithAllowedOrigins(strings.Split(*allowedOrigins, ",")).
		WithMaxCallRecvMsgSize(uint64(*grpcMaxMsgSize))
	if !(*disableHTTPEthAPI) {
		gw.WithApiMiddleware(fmt.Sprintf("%s:%d", *host, *ethApiPort), &apimiddleware.BeaconEndpointFactory{})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/swagger/", gateway.SwaggerServer())
	mux.HandleFunc("/healthz", healthzServer(gw))
	gw = gw.WithMux(mux)

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
