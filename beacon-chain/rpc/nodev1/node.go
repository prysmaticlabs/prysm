package nodev1

import (
	"context"
	"fmt"
	"runtime"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p/peers/peerdata"
	"github.com/prysmaticlabs/prysm/shared/version"
	"go.opencensus.io/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetIdentity retrieves data about the node's network presence.
func (ns *Server) GetIdentity(ctx context.Context, _ *ptypes.Empty) (*ethpb.IdentityResponse, error) {
	ctx, span := trace.StartSpan(ctx, "nodeV1.GetIdentity")
	defer span.End()

	peerId := ns.PeerManager.PeerID().Pretty()

	serializedEnr, err := p2p.SerializeENR(ns.PeerManager.ENR())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not obtain enr: %v", err)
	}
	enr := "enr:" + serializedEnr

	sourcep2p := ns.PeerManager.Host().Addrs()
	p2pAddresses := make([]string, len(sourcep2p))
	for i := range sourcep2p {
		p2pAddresses[i] = sourcep2p[i].String() + "/p2p/" + peerId
	}

	sourceDisc, err := ns.PeerManager.DiscoveryAddresses()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not obtain discovery address: %v", err)
	}
	discoveryAddresses := make([]string, len(sourceDisc))
	for i := range sourceDisc {
		discoveryAddresses[i] = sourceDisc[i].String()
	}

	metadata := &ethpb.Metadata{
		SeqNumber: ns.MetadataProvider.MetadataSeq(),
		Attnets:   ns.MetadataProvider.Metadata().Attnets,
	}

	return &ethpb.IdentityResponse{
		Data: &ethpb.Identity{
			PeerId:             peerId,
			Enr:                enr,
			P2PAddresses:       p2pAddresses,
			DiscoveryAddresses: discoveryAddresses,
			Metadata:           metadata,
		},
	}, nil
}

// GetPeer retrieves data about the given peer.
func (ns *Server) GetPeer(ctx context.Context, req *ethpb.PeerRequest) (*ethpb.PeerResponse, error) {
	ctx, span := trace.StartSpan(ctx, "nodev1.GetPeer")
	defer span.End()

	peerStatus := ns.PeersFetcher.Peers()
	id, err := peer.IDFromString(req.PeerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "Invalid peer ID: "+req.PeerId)
	}
	enr, err := peerStatus.ENR(id)
	if err != nil {
		if errors.Is(err, peerdata.ErrPeerUnknown) {
			return nil, status.Error(codes.NotFound, "Peer not found")
		}
		return nil, status.Errorf(codes.Internal, "Could not obtain ENR: %v", err)
	}
	serializedEnr, err := p2p.SerializeENR(enr)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not obtain ENR: %v", err)
	}
	p2pAddress, err := peerStatus.Address(id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not obtain address: %v", err)
	}
	state, err := peerStatus.ConnectionState(id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not obtain state: %v", err)
	}
	direction, err := peerStatus.Direction(id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Could not obtain direction: %v", err)
	}

	return &ethpb.PeerResponse{
		Data: &ethpb.Peer{
			PeerId:    req.PeerId,
			Enr:       "enr:" + serializedEnr,
			Address:   p2pAddress.String(),
			State:     ethpb.ConnectionState(state),
			Direction: ethpb.PeerDirection(direction),
		},
	}, nil
}

// ListPeers retrieves data about the node's network peers.
func (ns *Server) ListPeers(ctx context.Context, _ *ptypes.Empty) (*ethpb.PeersResponse, error) {
	ctx, span := trace.StartSpan(ctx, "nodev1.ListPeers")
	defer span.End()

	return nil, errors.New("unimplemented")
}

// GetVersion requests that the beacon node identify information about its implementation in a
// format similar to a HTTP User-Agent field.
func (ns *Server) GetVersion(ctx context.Context, _ *ptypes.Empty) (*ethpb.VersionResponse, error) {
	ctx, span := trace.StartSpan(ctx, "nodev1.GetVersion")
	defer span.End()

	v := fmt.Sprintf("Prysm/%s (%s %s)", version.GetSemanticVersion(), runtime.GOOS, runtime.GOARCH)
	return &ethpb.VersionResponse{
		Data: &ethpb.Version{
			Version: v,
		},
	}, nil
}

// GetSyncStatus requests the beacon node to describe if it's currently syncing or not, and
// if it is, what block it is up to.
func (ns *Server) GetSyncStatus(ctx context.Context, _ *ptypes.Empty) (*ethpb.SyncingResponse, error) {
	ctx, span := trace.StartSpan(ctx, "nodev1.GetSyncStatus")
	defer span.End()

	headSlot := ns.HeadFetcher.HeadSlot()
	return &ethpb.SyncingResponse{
		Data: &ethpb.SyncInfo{
			HeadSlot:     headSlot,
			SyncDistance: ns.GenesisTimeFetcher.CurrentSlot() - headSlot,
		},
	}, nil
}

// GetHealth returns node health status in http status codes. Useful for load balancers.
// Response Usage:
//    "200":
//      description: Node is ready
//    "206":
//      description: Node is syncing but can serve incomplete data
//    "503":
//      description: Node not initialized or having issues
func (ns *Server) GetHealth(ctx context.Context, _ *ptypes.Empty) (*ptypes.Empty, error) {
	ctx, span := trace.StartSpan(ctx, "nodev1.GetHealth")
	defer span.End()

	if ns.SyncChecker.Syncing() || ns.SyncChecker.Initialized() {
		return &ptypes.Empty{}, nil
	}
	return &ptypes.Empty{}, status.Error(codes.Internal, "Node not initialized or having issues")
}
