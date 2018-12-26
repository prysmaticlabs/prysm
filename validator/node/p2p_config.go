package node

import (
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/p2p/adapter/tracer"
	"github.com/urfave/cli"
)

var topicMappings = map[pb.Topic]proto.Message{
	pb.Topic_COLLATION_BODY_REQUEST:  &pb.CollationBodyRequest{},
	pb.Topic_COLLATION_BODY_RESPONSE: &pb.CollationBodyResponse{},
	pb.Topic_TRANSACTIONS:            &pb.Transaction{},
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
