package sync

import (
	"context"

	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/types"
)

// beaconBlockAndBlobsSidecarByRootRPCHandler looks up the request beacon block and blobs from the database from a given root
func (s *Service) beaconBlockAndBlobsSidecarByRootRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	return nil
}

func (s *Service) sendBlocksAndSidecarsRequest(ctx context.Context, blockRoots *types.BeaconBlockByRootsReq, id peer.ID) error {
	return nil
}
