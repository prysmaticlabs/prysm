package nodev1

import (
	"context"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1"
)

// GetIdentity retrieves data about the node's network presence.
func (ns *Server) GetIdentity(ctx context.Context, _ *ptypes.Empty) (*ethpb.IdentityResponse, error) {
	return nil, errors.New("unimplemented")
}

// GetPeer retrieves data about the given peer.
func (ns *Server) GetPeer(ctx context.Context, req *ethpb.PeerRequest) (*ethpb.PeerResponse, error) {
	return nil, errors.New("unimplemented")
}

// ListPeers retrieves data about the node's network peers.
func (ns *Server) ListPeers(ctx context.Context, _ *ptypes.Empty) (*ethpb.PeersResponse, error) {
	return nil, errors.New("unimplemented")
}

// GetVersion requests that the beacon node identify information about its implementation in a
// format similar to a HTTP User-Agent field.
func (ns *Server) GetVersion(ctx context.Context, _ *ptypes.Empty) (*ethpb.VersionResponse, error) {
	return nil, errors.New("unimplemented")
}

// GetSyncStatus requests the beacon node to describe if it's currently syncing or not, and
// if it is, what block it is up to.
func (ns *Server) GetSyncStatus(ctx context.Context, _ *ptypes.Empty) (*ethpb.SyncingResponse, error) {
	return nil, errors.New("unimplemented")
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
	return nil, errors.New("unimplemented")
}
