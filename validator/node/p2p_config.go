package node

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	beaconpb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/p2p/adapter/tracer"
	"github.com/urfave/cli"
)

var topicMappings = map[fmt.Stringer]proto.Message{
	// Beacon chain topics
	beaconpb.Topic_BEACON_BLOCK_ANNOUNCE: &beaconpb.BeaconBlockAnnounce{},
	beaconpb.Topic_BEACON_BLOCK_REQUEST:  &beaconpb.BeaconBlockRequest{},
	beaconpb.Topic_BEACON_BLOCK_RESPONSE: &beaconpb.BeaconBlockResponse{},
	beaconpb.Topic_ATTESTATION_ANNOUNCE:  &beaconpb.AttestationAnnounce{},
	beaconpb.Topic_ATTESTATION_REQUEST:   &beaconpb.AttestationRequest{},
	beaconpb.Topic_ATTESTATION_RESPONSE:  &beaconpb.AttestationResponse{},

	// Shard chain topics
}

func configureP2P(ctx *cli.Context) (*p2p.Server, error) {
	s, err := p2p.NewServer(&p2p.ServerConfig{
		BootstrapNodeAddr: ctx.GlobalString(cmd.BootstrapNode.Name),
		RelayNodeAddr:     ctx.GlobalString(cmd.RelayNode.Name),
		Port:              ctx.GlobalInt(cmd.P2PPort.Name),
	})
	if err != nil {
		return nil, err
	}

	traceAdapter, err := tracer.New("validator",
		ctx.GlobalString(cmd.TracingEndpointFlag.Name),
		ctx.GlobalFloat64(cmd.TraceSampleFractionFlag.Name),
		ctx.GlobalBool(cmd.EnableTracingFlag.Name))
	if err != nil {
		return nil, err
	}

	// TODO(437): Define default adapters for logging, monitoring, etc.
	adapters := []p2p.Adapter{traceAdapter}
	for k, v := range topicMappings {
		s.RegisterTopic(k.String(), v, adapters...)
	}

	return s, nil
}
