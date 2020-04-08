package node

import (
	"context"
	"fmt"
	"sort"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/libp2p/go-libp2p-core/network"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/prysmaticlabs/prysm/shared/version"
	"github.com/wealdtech/go-bytesutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server defines a server implementation of the gRPC Node service,
// providing RPC endpoints for verifying a beacon node's sync status, genesis and
// version information, and services the node implements and runs.
type Server struct {
	SyncChecker         sync.Checker
	Server              *grpc.Server
	BeaconDB            db.ReadOnlyDatabase
	PeersFetcher        p2p.PeersProvider
	HostFetcher         p2p.HostProvider
	GenesisTimeFetcher  blockchain.TimeFetcher
	HeadFetcher         blockchain.HeadFetcher
	FinalizationFetcher blockchain.FinalizationFetcher
}

// GetNodeInfo retrieves information about this node and its view of the beacon chain.
func (ns *Server) GetNodeInfo(ctx context.Context, _ *ptypes.Empty) (*ethpb.NodeInfo, error) {
	host := ns.HostFetcher.Host()

	addresses := make([]string, 0, len(host.Addrs()))
	for _, ma := range host.Addrs() {
		addresses = append(addresses, fmt.Sprintf("%s/p2p/%v", ma.String(), host.ID()))
	}

	peers := ns.generatePeerInfo()

	syncState := ns.calculateSyncState(ctx)

	signedHeadBlock, err := ns.HeadFetcher.HeadBlock(ctx)
	if err != nil || signedHeadBlock == nil {
		return nil, status.Error(codes.Internal, "Could not get head block")
	}
	headSlot := signedHeadBlock.Block.Slot
	headBlockRoot, err := stateutil.BlockRoot(signedHeadBlock.Block)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not get head block root: %v", err)
	}

	finalizedSlot, finalizedBlockRoot, err := ns.blockDataFromCheckpoint(ctx, ns.FinalizationFetcher.FinalizedCheckpt())
	if err != nil {
		return nil, err
	}

	justifiedSlot, justifiedBlockRoot, err := ns.blockDataFromCheckpoint(ctx, ns.FinalizationFetcher.CurrentJustifiedCheckpt())
	if err != nil {
		return nil, err
	}

	prevJustifiedSlot, prevJustifiedBlockRoot, err := ns.blockDataFromCheckpoint(ctx, ns.FinalizationFetcher.PreviousJustifiedCheckpt())
	if err != nil {
		return nil, err
	}

	return &ethpb.NodeInfo{
		NodeId:                     host.ID().String(),
		Version:                    version.GetVersion(),
		Addresses:                  addresses,
		Peers:                      peers,
		SyncState:                  syncState,
		CurrentEpoch:               headSlot / params.BeaconConfig().SlotsPerEpoch,
		CurrentSlot:                headSlot,
		CurrentBlockRoot:           headBlockRoot[:],
		FinalizedEpoch:             finalizedSlot / params.BeaconConfig().SlotsPerEpoch,
		FinalizedSlot:              finalizedSlot,
		FinalizedBlockRoot:         finalizedBlockRoot,
		JustifiedEpoch:             justifiedSlot / params.BeaconConfig().SlotsPerEpoch,
		JustifiedSlot:              justifiedSlot,
		JustifiedBlockRoot:         justifiedBlockRoot,
		PreviousJustifiedEpoch:     prevJustifiedSlot / params.BeaconConfig().SlotsPerEpoch,
		PreviousJustifiedSlot:      prevJustifiedSlot,
		PreviousJustifiedBlockRoot: prevJustifiedBlockRoot,
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

// generatePeerInfo generates the peer list for node info.
func (ns *Server) generatePeerInfo() []*ethpb.Peer {
	peers := make([]*ethpb.Peer, 0, len(ns.PeersFetcher.Peers().Connected()))
	for _, pid := range ns.PeersFetcher.Peers().Connected() {
		multiaddr, err := ns.PeersFetcher.Peers().Address(pid)
		if err != nil {
			continue
		}
		direction, err := ns.PeersFetcher.Peers().Direction(pid)
		if err != nil {
			continue
		}
		address := fmt.Sprintf("%s/p2p/%v", multiaddr.String(), pid)
		pbDirection := ethpb.PeerDirection_UNKNOWN
		switch direction {
		case network.DirInbound:
			pbDirection = ethpb.PeerDirection_INBOUND
		case network.DirOutbound:
			pbDirection = ethpb.PeerDirection_OUTBOUND
		}
		peers = append(peers, &ethpb.Peer{
			// Add ENR when available.
			Address:   address,
			Direction: pbDirection,
		})
	}
	return peers
}

// blockDataFromCheckpoint obtains block information given a checkpoint for node info.
func (ns *Server) blockDataFromCheckpoint(ctx context.Context, checkpoint *ethpb.Checkpoint) (uint64, []byte, error) {
	signedBlock, err := ns.BeaconDB.Block(ctx, bytesutil.ToBytes32(checkpoint.Root))
	if err != nil || signedBlock == nil || signedBlock.Block == nil {
		return 0, nil, status.Error(codes.Internal, "Could not get block")
	}

	return signedBlock.Block.Slot, checkpoint.Root, nil
}

func (ns *Server) calculateSyncState(ctx context.Context) ethpb.SyncState {
	if ns.SyncChecker.Syncing() {
		return ethpb.SyncState_SYNC_CATCHUP
	}
	headState, err := ns.HeadFetcher.HeadState(ctx)
	if err == nil {
		currentEpoch := uint64(roughtime.Now().Unix()-ns.GenesisTimeFetcher.GenesisTime().Unix()) / (params.BeaconConfig().SecondsPerSlot * params.BeaconConfig().SlotsPerEpoch)
		syncedEpoch := helpers.CurrentEpoch(headState)
		if currentEpoch > 0 && syncedEpoch < currentEpoch-1 {
			return ethpb.SyncState_SYNC_INACTIVE
		}
		return ethpb.SyncState_SYNC_FULL
	}
	return ethpb.SyncState_SYNC_UNKNOWN
}
