// Package node defines a gRPC node service implementation, providing
// useful endpoints for checking a node's sync status, peer info,
// genesis data, and version information.
package node

import (
	"context"
	"fmt"
	"sort"
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
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
	PeerManager        p2p.PeerManager
	GenesisTimeFetcher blockchain.TimeFetcher
	GenesisFetcher     blockchain.GenesisFetcher
}

// GetSyncStatus checks the current network sync status of the node.
func (ns *Server) GetSyncStatus(ctx context.Context, _ *ptypes.Empty) (*ethpb.SyncStatus, error) {
	return &ethpb.SyncStatus{
		Syncing: ns.SyncChecker.Syncing(),
	}, nil
}

// GetGenesis fetches genesis chain information of Ethereum 2.0. Returns unix timestamp 0
// if a genesis time has yet to be determined.
func (ns *Server) GetGenesis(ctx context.Context, _ *ptypes.Empty) (*ethpb.Genesis, error) {
	contractAddr, err := ns.BeaconDB.DepositContractAddress(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not retrieve contract address from db: %v", err)
	}
	genesisTime := ns.GenesisTimeFetcher.GenesisTime()
	var defaultGenesisTime time.Time
	var gt *ptypes.Timestamp
	if genesisTime == defaultGenesisTime {
		gt, err = ptypes.TimestampProto(time.Unix(0, 0))
	} else {
		gt, err = ptypes.TimestampProto(genesisTime)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not convert genesis time to proto: %v", err)
	}

	genValRoot := ns.GenesisFetcher.GenesisValidatorRoot()
	return &ethpb.Genesis{
		GenesisTime:            gt,
		DepositContractAddress: contractAddr,
		GenesisValidatorsRoot:  genValRoot[:],
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

// GetHost returns the p2p data on the current local and host peer.
func (ns *Server) GetHost(ctx context.Context, _ *ptypes.Empty) (*ethpb.HostData, error) {
	stringAddr := []string{}
	for _, addr := range ns.PeerManager.Host().Addrs() {
		stringAddr = append(stringAddr, addr.String())
	}
	record := ns.PeerManager.ENR()
	enr := ""
	err := error(nil)
	if record != nil {
		enr, err = p2p.SerializeENR(record)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Unable to serialize enr: %v", err)
		}
	}

	return &ethpb.HostData{
		Addresses: stringAddr,
		PeerId:    ns.PeerManager.PeerID().String(),
		Enr:       enr,
	}, nil
}

// GetPeer returns the data known about the peer defined by the provided peer id.
func (ns *Server) GetPeer(ctx context.Context, peerReq *ethpb.PeerRequest) (*ethpb.Peer, error) {
	pid, err := peer.Decode(peerReq.PeerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Unable to parse provided peer id: %v", err)
	}
	addr, err := ns.PeersFetcher.Peers().Address(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	dir, err := ns.PeersFetcher.Peers().Direction(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	pbDirection := ethpb.PeerDirection_UNKNOWN
	switch dir {
	case network.DirInbound:
		pbDirection = ethpb.PeerDirection_INBOUND
	case network.DirOutbound:
		pbDirection = ethpb.PeerDirection_OUTBOUND
	}
	connState, err := ns.PeersFetcher.Peers().ConnectionState(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	record, err := ns.PeersFetcher.Peers().ENR(pid)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "Requested peer does not exist: %v", err)
	}
	enr := ""
	if record != nil {
		enr, err = p2p.SerializeENR(record)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Unable to serialize enr: %v", err)
		}
	}
	return &ethpb.Peer{
		Address:         addr.String(),
		Direction:       pbDirection,
		ConnectionState: ethpb.ConnectionState(connState),
		PeerId:          peerReq.PeerId,
		Enr:             enr,
	}, nil
}

// ListPeers lists the peers connected to this node.
func (ns *Server) ListPeers(ctx context.Context, _ *ptypes.Empty) (*ethpb.Peers, error) {
	peers := ns.PeersFetcher.Peers().Connected()
	res := make([]*ethpb.Peer, 0, len(peers))
	for _, pid := range peers {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		multiaddr, err := ns.PeersFetcher.Peers().Address(pid)
		if err != nil {
			continue
		}
		direction, err := ns.PeersFetcher.Peers().Direction(pid)
		if err != nil {
			continue
		}
		record, err := ns.PeersFetcher.Peers().ENR(pid)
		if err != nil {
			continue
		}
		enr := ""
		if record != nil {
			enr, err = p2p.SerializeENR(record)
			if err != nil {
				continue
			}
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
			Address:         address,
			Direction:       pbDirection,
			ConnectionState: ethpb.ConnectionState_CONNECTED,
			PeerId:          pid.String(),
			Enr:             enr,
		})
	}

	return &ethpb.Peers{
		Peers: res,
	}, nil
}
