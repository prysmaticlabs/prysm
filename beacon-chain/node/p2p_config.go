package node

import (
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/utils"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/p2p"
	"github.com/prysmaticlabs/prysm/shared/p2p/adapter/metric"
	"github.com/urfave/cli"
)

var topicMappings = map[pb.Topic]proto.Message{
	pb.Topic_BEACON_BLOCK_ANNOUNCE:               &ethpb.BeaconBlockAnnounce{},
	pb.Topic_BEACON_BLOCK_REQUEST:                &ethpb.BeaconBlockRequest{},
	pb.Topic_BEACON_BLOCK_REQUEST_BY_SLOT_NUMBER: &ethpb.BeaconBlockRequestBySlotNumber{},
	pb.Topic_BEACON_BLOCK_RESPONSE:               &ethpb.BeaconBlockResponse{},
	pb.Topic_BATCHED_BEACON_BLOCK_REQUEST:        &ethpb.BatchedBeaconBlockRequest{},
	pb.Topic_BATCHED_BEACON_BLOCK_RESPONSE:       &ethpb.BatchedBeaconBlockResponse{},
	pb.Topic_CHAIN_HEAD_REQUEST:                  &ethpb.ChainHeadRequest{},
	pb.Topic_CHAIN_HEAD_RESPONSE:                 &ethpb.ChainHeadResponse{},
	pb.Topic_BEACON_STATE_HASH_ANNOUNCE:          &pb.BeaconStateHashAnnounce{},
	pb.Topic_BEACON_STATE_REQUEST:                &pb.BeaconStateRequest{},
	pb.Topic_BEACON_STATE_RESPONSE:               &pb.BeaconStateResponse{},
	pb.Topic_ATTESTATION_ANNOUNCE:                &ethpb.AttestationAnnounce{},
	pb.Topic_ATTESTATION_REQUEST:                 &pb.AttestationRequest{},
	pb.Topic_ATTESTATION_RESPONSE:                &ethpb.AttestationResponse{},
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
	staticPeers := []string{}
	for _, entry := range ctx.GlobalStringSlice(cmd.StaticPeers.Name) {
		peers := strings.Split(entry, ",")
		staticPeers = append(staticPeers, peers...)
	}

	s, err := p2p.NewServer(&p2p.ServerConfig{
		NoDiscovery:            ctx.GlobalBool(cmd.NoDiscovery.Name),
		StaticPeers:            staticPeers,
		BootstrapNodeAddr:      ctx.GlobalString(cmd.BootstrapNode.Name),
		RelayNodeAddr:          ctx.GlobalString(cmd.RelayNode.Name),
		HostAddress:            ctx.GlobalString(cmd.P2PHost.Name),
		Port:                   ctx.GlobalInt(cmd.P2PPort.Name),
		MaxPeers:               ctx.GlobalInt(cmd.P2PMaxPeers.Name),
		PrvKey:                 ctx.GlobalString(cmd.P2PPrivKey.Name),
		DepositContractAddress: contractAddress,
		WhitelistCIDR:          ctx.GlobalString(cmd.P2PWhitelist.Name),
		EnableUPnP:             ctx.GlobalBool(cmd.EnableUPnPFlag.Name),
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
