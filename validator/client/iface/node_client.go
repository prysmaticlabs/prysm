package iface

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type NodeClient interface {
	GetSyncStatus(ctx context.Context, in *empty.Empty) (*ethpb.SyncStatus, error)
	GetGenesis(ctx context.Context, in *empty.Empty) (*ethpb.Genesis, error)
	GetVersion(ctx context.Context, in *empty.Empty) (*ethpb.Version, error)
	ListImplementedServices(ctx context.Context, in *empty.Empty) (*ethpb.ImplementedServices, error)
	GetHost(ctx context.Context, in *empty.Empty) (*ethpb.HostData, error)
	GetPeer(ctx context.Context, in *ethpb.PeerRequest) (*ethpb.Peer, error)
	ListPeers(ctx context.Context, in *empty.Empty) (*ethpb.Peers, error)
	GetETH1ConnectionStatus(ctx context.Context, in *empty.Empty) (*ethpb.ETH1ConnectionStatus, error)
}
