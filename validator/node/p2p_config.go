package node

import (
	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/p2p/adapter/tracer"
	"github.com/urfave/cli"

	pb "github.com/prysmaticlabs/prysm/proto/sharding/p2p/v1"
)

var topicMappings = map[pb.Topic]proto.Message{
	pb.Topic_COLLATION_BODY_REQUEST:  &pb.CollationBodyRequest{},
	pb.Topic_COLLATION_BODY_RESPONSE: &pb.CollationBodyResponse{},
	pb.Topic_TRANSACTIONS:            &pb.Transaction{},
}

func configureP2P(ctx *cli.Context) (*p2p.Server, error) {
	s, err := p2p.NewServer()
	if err != nil {
		return nil, err
	}

	traceAdapter, err := tracer.New("validator", ctx.GlobalString(cmd.TracingEndpointFlag.Name),
		ctx.GlobalBool(cmd.DisableTracingFlag.Name))
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
