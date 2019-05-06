package node

import (
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/p2p/adapter/metric"
	"github.com/urfave/cli"
)

var topicMappings = map[pb.Topic]proto.Message{
	pb.Topic_BEACON_BLOCK_ANNOUNCE:               &pb.BeaconBlockAnnounce{},
	pb.Topic_BEACON_BLOCK_REQUEST:                &pb.BeaconBlockRequest{},
	pb.Topic_BEACON_BLOCK_REQUEST_BY_SLOT_NUMBER: &pb.BeaconBlockRequestBySlotNumber{},
	pb.Topic_BEACON_BLOCK_RESPONSE:               &pb.BeaconBlockResponse{},
	pb.Topic_BATCHED_BEACON_BLOCK_REQUEST:        &pb.BatchedBeaconBlockRequest{},
	pb.Topic_BATCHED_BEACON_BLOCK_RESPONSE:       &pb.BatchedBeaconBlockResponse{},
	pb.Topic_CHAIN_HEAD_REQUEST:                  &pb.ChainHeadRequest{},
	pb.Topic_CHAIN_HEAD_RESPONSE:                 &pb.ChainHeadResponse{},
	pb.Topic_BEACON_STATE_HASH_ANNOUNCE:          &pb.BeaconStateHashAnnounce{},
	pb.Topic_BEACON_STATE_REQUEST:                &pb.BeaconStateRequest{},
	pb.Topic_BEACON_STATE_RESPONSE:               &pb.BeaconStateResponse{},
	pb.Topic_ATTESTATION_ANNOUNCE:                &pb.AttestationAnnounce{},
	pb.Topic_ATTESTATION_REQUEST:                 &pb.AttestationRequest{},
	pb.Topic_ATTESTATION_RESPONSE:                &pb.AttestationResponse{},
}

func configureP2P(ctx *cli.Context) (*p2p.Server, error) {
	contractAddress := ctx.GlobalString(utils.DepositContractFlag.Name)
	if contractAddress == "" {
		var err error
		contractAddress, err = fetchDepositContract()
		if err != nil {
			return nil, err
		}
	}

	s, err := p2p.NewServer(&p2p.ServerConfig{
		NoDiscovery:            ctx.GlobalBool(cmd.NoDiscovery.Name),
		BootstrapNodeAddr:      ctx.GlobalString(cmd.BootstrapNode.Name),
		RelayNodeAddr:          ctx.GlobalString(cmd.RelayNode.Name),
		HostAddress:            ctx.GlobalString(cmd.P2PHost.Name),
		Port:                   ctx.GlobalInt(cmd.P2PPort.Name),
		DepositContractAddress: contractAddress,
	})
	if err != nil {
		return nil, err
	}

	adapters := []p2p.Adapter{}
	if !ctx.GlobalBool(cmd.DisableMonitoringFlag.Name) {
		adapters = append(adapters, metric.New())
	}

	for k, v := range topicMappings {
		s.RegisterTopic(k.String(), v, adapters...)
	}

	return s, nil
}
