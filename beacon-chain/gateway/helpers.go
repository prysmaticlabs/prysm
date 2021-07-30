package gateway

import (
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	pbrpc "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/gateway"
	"google.golang.org/protobuf/encoding/protojson"
)

// MuxConfig contains configuration that should be used when registering the beacon node in the gateway.
type MuxConfig struct {
	Handler       gateway.MuxHandler
	V1PbMux       gateway.PbMux
	V1Alpha1PbMux gateway.PbMux
}

// DefaultConfig returns a fully configured MuxConfig with standard gateway behavior.
func DefaultConfig(enableDebugRPCEndpoints bool) MuxConfig {
	v1Alpha1Registrations := []gateway.PbHandlerRegistration{
		ethpb.RegisterNodeHandler,
		ethpb.RegisterBeaconChainHandler,
		ethpb.RegisterBeaconNodeValidatorHandler,
		pbrpc.RegisterHealthHandler,
	}
	v1Registrations := []gateway.PbHandlerRegistration{
		ethpbv1.RegisterBeaconNodeHandler,
		ethpbv1.RegisterBeaconChainHandler,
		ethpbv1.RegisterBeaconValidatorHandler,
		ethpbv1.RegisterEventsHandler,
	}
	if enableDebugRPCEndpoints {
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

	return MuxConfig{
		V1PbMux:       v1PbHandler,
		V1Alpha1PbMux: v1Alpha1PbHandler,
	}
}
