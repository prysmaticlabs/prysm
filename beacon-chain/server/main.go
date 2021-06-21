// Package main allows for creation of an HTTP-JSON to gRPC
// gateway as a binary go process.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strings"

	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	joonix "github.com/joonix/log"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/apimiddleware"
	pbrpc "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/gateway"
	_ "github.com/prysmaticlabs/prysm/shared/maxprocs"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	beaconRPC               = flag.String("beacon-rpc", "localhost:4000", "Beacon chain gRPC endpoint")
	port                    = flag.Int("port", 8000, "Port to serve on")
	apiMiddlewarePort       = flag.Int("port", 8001, "Port to serve API middleware on")
	host                    = flag.String("host", "127.0.0.1", "Host to serve on")
	debug                   = flag.Bool("debug", false, "Enable debug logging")
	allowedOrigins          = flag.String("corsdomain", "localhost:4242", "A comma separated list of CORS domains to allow")
	enableDebugRPCEndpoints = flag.Bool("enable-debug-rpc-endpoints", false, "Enable debug rpc endpoints such as /eth/v1alpha1/beacon/state")
	grpcMaxMsgSize          = flag.Int("grpc-max-msg-size", 1<<22, "Integer to define max recieve message call size")
)

func init() {
	logrus.SetFormatter(joonix.NewFormatter())
}

func main() {
	flag.Parse()
	if *debug {
		log.SetLevel(logrus.DebugLevel)
	}

	v1Alpha1Registrations := []gateway.PbHandlerRegistration{
		ethpb.RegisterNodeHandler,
		ethpb.RegisterBeaconChainHandler,
		ethpb.RegisterBeaconNodeValidatorHandler,
		ethpbv1.RegisterEventsHandler,
		pbrpc.RegisterHealthHandler,
	}
	v1Registrations := []gateway.PbHandlerRegistration{
		ethpbv1.RegisterBeaconNodeHandler,
		ethpbv1.RegisterBeaconChainHandler,
		ethpbv1.RegisterBeaconValidatorHandler,
	}
	if *enableDebugRPCEndpoints {
		v1Alpha1Registrations = append(v1Alpha1Registrations, pbrpc.RegisterDebugHandler)
		v1Registrations = append(v1Registrations, ethpbv1.RegisterBeaconDebugHandler)

	}
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
	muxHandler := func(h http.Handler, w http.ResponseWriter, req *http.Request) {
		h.ServeHTTP(w, req)
	}
	v1Alpha1PbHandler := gateway.PbMux{
		Registrations: v1Alpha1Registrations,
		Patterns:      []string{"/eth/v1alpha1/"},
		Mux:           v1Alpha1Mux,
	}
	v1PbHandler := gateway.PbMux{
		Registrations: v1Registrations,
		Patterns:      []string{"/eth/v1/"},
		Mux:           v1Mux,
	}

	gw := gateway.New(
		context.Background(),
		[]gateway.PbMux{v1Alpha1PbHandler, v1PbHandler},
		muxHandler,
		*beaconRPC,
		fmt.Sprintf("%s:%d", *host, *port),
	).WithAllowedOrigins(strings.Split(*allowedOrigins, ",")).
		WithMaxCallRecvMsgSize(uint64(*grpcMaxMsgSize)).
		WithApiMiddleware(fmt.Sprintf("%s:%d", *host, *apiMiddlewarePort), &apimiddleware.BeaconEndpointFactory{})

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
