package gateway

import (
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prysmaticlabs/prysm/api/gateway"
	ethpbservice "github.com/prysmaticlabs/prysm/proto/eth/service"
	ethpbalpha "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/encoding/protojson"
)

// MuxConfig contains configuration that should be used when registering the beacon node in the gateway.
type MuxConfig struct {
	Handler       gateway.MuxHandler
	EthPbMux      gateway.PbMux
	V1Alpha1PbMux gateway.PbMux
}

// DefaultConfig returns a fully configured MuxConfig with standard gateway behavior.
func DefaultConfig(enableDebugRPCEndpoints bool) MuxConfig {
	v1Alpha1Registrations := []gateway.PbHandlerRegistration{
		ethpbalpha.RegisterNodeHandler,
		ethpbalpha.RegisterBeaconChainHandler,
		ethpbalpha.RegisterBeaconNodeValidatorHandler,
		ethpbalpha.RegisterHealthHandler,
	}
	ethRegistrations := []gateway.PbHandlerRegistration{
		ethpbservice.RegisterBeaconNodeHandler,
		ethpbservice.RegisterBeaconChainHandler,
		ethpbservice.RegisterBeaconValidatorHandler,
		ethpbservice.RegisterEventsHandler,
	}
	if enableDebugRPCEndpoints {
		v1Alpha1Registrations = append(v1Alpha1Registrations, ethpbalpha.RegisterDebugHandler)
		ethRegistrations = append(ethRegistrations, ethpbservice.RegisterBeaconDebugHandler)

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
	ethMux := gwruntime.NewServeMux(
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
	v1Alpha1PbHandler := gateway.PbMux{
		Registrations: v1Alpha1Registrations,
		Patterns:      []string{"/eth/v1alpha1/"},
		Mux:           v1Alpha1Mux,
	}
	ethPbHandler := gateway.PbMux{
		Registrations: ethRegistrations,
		Patterns:      []string{"/internal/eth/v1/", "/internal/eth/v2/"},
		Mux:           ethMux,
	}

	return MuxConfig{
		EthPbMux:      ethPbHandler,
		V1Alpha1PbMux: v1Alpha1PbHandler,
	}
}
