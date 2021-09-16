package gateway

import (
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
	ethpbalpha "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/gateway"
	"google.golang.org/protobuf/encoding/protojson"
)

// MuxConfig contains configuration that should be used when registering the beacon node in the gateway.
type MuxConfig struct {
	Handler       gateway.MuxHandler
	V1PbMux       *gateway.PbMux
	V1Alpha1PbMux *gateway.PbMux
}

// DefaultConfig returns a fully configured MuxConfig with standard gateway behavior.
func DefaultConfig(enableDebugRPCEndpoints, enableHTTPPrysmAPI, enableHTTPEthAPI bool) MuxConfig {
	var v1Alpha1PbHandler, v1PbHandler *gateway.PbMux
	if enableHTTPPrysmAPI {
		v1Alpha1Registrations := []gateway.PbHandlerRegistration{
			ethpbalpha.RegisterNodeHandler,
			ethpbalpha.RegisterBeaconChainHandler,
			ethpbalpha.RegisterBeaconNodeValidatorHandler,
			ethpbalpha.RegisterHealthHandler,
		}
		if enableDebugRPCEndpoints {
			v1Alpha1Registrations = append(v1Alpha1Registrations, ethpbalpha.RegisterDebugHandler)
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
		v1Alpha1PbHandler = &gateway.PbMux{
			Registrations: v1Alpha1Registrations,
			Patterns:      []string{"/eth/v1alpha1/"},
			Mux:           v1Alpha1Mux,
		}
	}
	if enableHTTPEthAPI {
		v1Registrations := []gateway.PbHandlerRegistration{
			ethpbservice.RegisterBeaconNodeHandler,
			ethpbservice.RegisterBeaconChainHandler,
			ethpbservice.RegisterBeaconValidatorHandler,
			ethpbservice.RegisterEventsHandler,
		}
		if enableDebugRPCEndpoints {
			v1Registrations = append(v1Registrations, ethpbservice.RegisterBeaconDebugHandler)
		}
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
		v1PbHandler = &gateway.PbMux{
			Registrations: v1Registrations,
			Patterns:      []string{"/eth/v1/"},
			Mux:           v1Mux,
		}
	}

	return MuxConfig{
		V1PbMux:       v1PbHandler,
		V1Alpha1PbMux: v1Alpha1PbHandler,
	}
}
