// Package main allows for creation of an HTTP-JSON to gRPC
// gateway as a binary go process.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	joonix "github.com/joonix/log"
	"github.com/prysmaticlabs/prysm/v3/api/gateway"
	beaconGateway "github.com/prysmaticlabs/prysm/v3/beacon-chain/gateway"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/v3/cmd/beacon-chain/flags"
	_ "github.com/prysmaticlabs/prysm/v3/runtime/maxprocs"
	"github.com/sirupsen/logrus"
)

var (
	beaconRPC               = flag.String("beacon-rpc", "localhost:4000", "Beacon chain gRPC endpoint")
	port                    = flag.Int("port", 8000, "Port to serve on")
	host                    = flag.String("host", "127.0.0.1", "Host to serve on")
	debug                   = flag.Bool("debug", false, "Enable debug logging")
	allowedOrigins          = flag.String("corsdomain", "localhost:4242", "A comma separated list of CORS domains to allow")
	enableDebugRPCEndpoints = flag.Bool("enable-debug-rpc-endpoints", false, "Enable debug rpc endpoints such as /eth/v1alpha1/beacon/state")
	grpcMaxMsgSize          = flag.Int("grpc-max-msg-size", 1<<22, "Integer to define max recieve message call size")
	httpModules             = flag.String(
		"http-modules",
		strings.Join([]string{flags.PrysmAPIModule, flags.EthAPIModule}, ","),
		"Comma-separated list of API module names. Possible values: `"+flags.PrysmAPIModule+`,`+flags.EthAPIModule+"`.",
	)
)

func init() {
	logrus.SetFormatter(joonix.NewFormatter())
}

func main() {
	flag.Parse()
	if *debug {
		log.SetLevel(logrus.DebugLevel)
	}

	gatewayConfig := beaconGateway.DefaultConfig(*enableDebugRPCEndpoints, *httpModules)
	muxs := make([]*gateway.PbMux, 0)
	if gatewayConfig.V1AlphaPbMux != nil {
		muxs = append(muxs, gatewayConfig.V1AlphaPbMux)
	}
	if gatewayConfig.EthPbMux != nil {
		muxs = append(muxs, gatewayConfig.EthPbMux)
	}
	opts := []gateway.Option{
		gateway.WithPbHandlers(muxs),
		gateway.WithMuxHandler(gatewayConfig.Handler),
		gateway.WithRemoteAddr(*beaconRPC),
		gateway.WithGatewayAddr(fmt.Sprintf("%s:%d", *host, *port)),
		gateway.WithAllowedOrigins(strings.Split(*allowedOrigins, ",")),
		gateway.WithMaxCallRecvMsgSize(uint64(*grpcMaxMsgSize)),
	}

	if flags.EnableHTTPEthAPI(*httpModules) {
		opts = append(opts, gateway.WithApiMiddleware(&apimiddleware.BeaconEndpointFactory{}))
	}

	gw, err := gateway.New(context.Background(), opts...)
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/swagger/", gateway.SwaggerServer())
	r.HandleFunc("/healthz", healthzServer(gw))
	gw.SetRouter(r)

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
