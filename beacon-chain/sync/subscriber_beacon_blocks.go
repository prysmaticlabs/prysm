package sync

import (
	"context"

	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition/interop"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	eth "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

func (s *Service) beaconBlockSubscriber(ctx context.Context, msg proto.Message) error {
	signed, err := wrapper.WrappedSignedBeaconBlock(msg)
	if err != nil {
		return err
	}
	if err := wrapper.BeaconBlockIsNil(signed); err != nil {
		return err
	}

	s.setSeenBlockIndexSlot(signed.Block().Slot(), signed.Block().ProposerIndex())

	block := signed.Block()

	root, err := block.HashTreeRoot()
	if err != nil {
		return err
	}

	var sidecar *eth.BlobsSidecar
	if blockRequiresSidecar(block) {
		slot := block.Slot()
		s.pendingQueueLock.RLock()
		sidecars := s.pendingSidecarsInCache(slot)
		s.pendingQueueLock.RUnlock()

		queuedSidecar := findSidecarForBlock(block, root, sidecars)
		if queuedSidecar == nil {
			// re-schedule block to be processed later.
			// TODO(XXX): This is a bit inefficient as the block will be validated again
			s.pendingQueueLock.Lock()
			s.insertBlockToPendingQueue(slot, signed, root)
			s.pendingQueueLock.Unlock()
			return nil
		}
		sidecar = queuedSidecar.s
	}

	if err := s.cfg.chain.ReceiveBlock(ctx, signed, root, sidecar); err != nil {
		if blockchain.IsInvalidBlock(err) {
			interop.WriteBlockToDisk(signed, true /*failed*/)
			s.setBadBlock(ctx, root)
		}
		return err
	}
	return err
}

// The input attestations are seen by the network, this deletes them from pool
// so proposers don't include them in a block for the future.
func (s *Service) deleteAttsInPool(atts []*ethpb.Attestation) error {
	for _, att := range atts {
		if helpers.IsAggregated(att) {
			if err := s.cfg.attPool.DeleteAggregatedAttestation(att); err != nil {
				return err
			}
		} else {
			// Ideally there's shouldn't be any unaggregated attestation in the block.
			if err := s.cfg.attPool.DeleteUnaggregatedAttestation(att); err != nil {
				return err
			}
		}
	}
	return nil
}

func blockRequiresSidecar(b interfaces.BeaconBlock) bool {
	if blocks.IsPreEIP4844Version(b.Version()) {
		return false
	}
	blobKzgs, _ := b.Body().BlobKzgs()
	return len(blobKzgs) != 0
}
