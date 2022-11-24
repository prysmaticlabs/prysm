package sync

import (
	"context"

	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

// beaconBlockAndBlobsSidecarByRootRPCHandler looks up the request beacon block and blobs from the database from a given root
func (s *Service) beaconBlockAndBlobsSidecarByRootRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	return nil
}

func (s *Service) sendBlocksAndSidecarsRequest(ctx context.Context, blockRoots *types.BeaconBlockByRootsReq, id peer.ID) error {
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()

	_, err := SendBlocksAndSidecarsByRootRequest(ctx, s.cfg.chain, s.cfg.p2p, id, blockRoots, func(blkAndSidecar *ethpb.SignedBeaconBlockAndBlobsSidecar) error {
		blk, err := blocks.NewSignedBeaconBlock(blkAndSidecar.BeaconBlock)
		if err != nil {
			return err
		}
		blkRoot, err := blk.Block().HashTreeRoot()
		if err != nil {
			return err
		}
		s.pendingQueueLock.Lock()
		defer s.pendingQueueLock.Unlock()
		if err := s.insertBlockToPendingQueue(blk.Block().Slot(), blk, blkRoot); err != nil {
			return err
		}
		err = s.cfg.beaconDB.SaveBlobsSidecar(ctx, blkAndSidecar.BlobsSidecar)
		if err != nil {
			return err
		}
		return nil
	})
	return err
}
