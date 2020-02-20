package node

import (
	"context"
	"fmt"
	"sort"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/libp2p/go-libp2p-core/network"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/shared/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server defines a server implementation of the gRPC Node service,
// providing RPC endpoints for verifying a beacon node's sync status, genesis and
// version information, and services the node implements and runs.
type Server struct {
	SyncChecker        sync.Checker
	Server             *grpc.Server
	BeaconDB           db.ReadOnlyDatabase
	PeersFetcher       p2p.PeersProvider
	GenesisTimeFetcher blockchain.TimeFetcher
}

// GetSyncStatus checks the current network sync status of the node.
func (ns *Server) GetSyncStatus(ctx context.Context, _ *ptypes.Empty) (*ethpb.SyncStatus, error) {
	return &ethpb.SyncStatus{
		Syncing: ns.SyncChecker.Syncing(),
	}, nil
}

// GetGenesis fetches genesis chain information of Ethereum 2.0.
func (ns *Server) GetGenesis(ctx context.Context, _ *ptypes.Empty) (*ethpb.Genesis, error) {
	contractAddr, err := ns.BeaconDB.DepositContractAddress(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve contract address from db: %v", err)
	}
	genesisTime := ns.GenesisTimeFetcher.GenesisTime()
	gt, err := ptypes.TimestampProto(genesisTime)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not convert genesis time to proto: %v", err)
	}
	return &ethpb.Genesis{
		GenesisTime:            gt,
		DepositContractAddress: contractAddr,
	}, nil
}

// GetVersion checks the version information of the beacon node.
func (ns *Server) GetVersion(ctx context.Context, _ *ptypes.Empty) (*ethpb.Version, error) {
	return &ethpb.Version{
		Version: version.GetVersion(),
	}, nil
}

// ListImplementedServices lists the services implemented and enabled by this node.
//
// Any service not present in this list may return UNIMPLEMENTED or
// PERMISSION_DENIED. The server may also support fetching services by grpc
// reflection.
func (ns *Server) ListImplementedServices(ctx context.Context, _ *ptypes.Empty) (*ethpb.ImplementedServices, error) {
	serviceInfo := ns.Server.GetServiceInfo()
	serviceNames := make([]string, 0, len(serviceInfo))
	for svc := range serviceInfo {
		serviceNames = append(serviceNames, svc)
	}
	sort.Strings(serviceNames)
	return &ethpb.ImplementedServices{
		Services: serviceNames,
	}, nil
}

// ListPeers lists the peers connected to this node.
func (ns *Server) ListPeers(ctx context.Context, _ *ptypes.Empty) (*ethpb.Peers, error) {
	res := make([]*ethpb.Peer, 0)
	for _, pid := range ns.PeersFetcher.Peers().Connected() {
		multiaddr, err := ns.PeersFetcher.Peers().Address(pid)
		if err != nil {
			continue
		}
		direction, err := ns.PeersFetcher.Peers().Direction(pid)
		if err != nil {
			continue
		}

		address := fmt.Sprintf("%s/p2p/%s", multiaddr.String(), pid.Pretty())
		pbDirection := ethpb.PeerDirection_UNKNOWN
		switch direction {
		case network.DirInbound:
			pbDirection = ethpb.PeerDirection_INBOUND
		case network.DirOutbound:
			pbDirection = ethpb.PeerDirection_OUTBOUND
		}
		res = append(res, &ethpb.Peer{
			Address:   address,
			Direction: pbDirection,
		})
	}

	return &ethpb.Peers{
		Peers: res,
	}, nil
}
