package blockchain

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// BlockReceiver interface defines the methods of chain service receive and processing new blocks.
type BlockReceiver interface {
	ReceiveBlock(ctx context.Context, block *ethpb.SignedBeaconBlock, blockRoot [32]byte) error
	ReceiveBlockInitialSync(ctx context.Context, block *ethpb.SignedBeaconBlock, blockRoot [32]byte) error
	ReceiveBlockBatch(ctx context.Context, blocks []*ethpb.SignedBeaconBlock, blkRoots [][32]byte) error
	HasInitSyncBlock(root [32]byte) bool
}

// ReceiveBlock is a function that defines the the operations (minus pubsub)
// that are performed on blocks that is received from regular sync service. The operations consists of:
//   1. Validate block, apply state transition and update check points
//   2. Apply fork choice to the processed block
//   3. Save latest head info
func (s *Service) ReceiveBlock(ctx context.Context, block *ethpb.SignedBeaconBlock, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.ReceiveBlock")
	defer span.End()
	blockCopy := stateTrie.CopySignedBeaconBlock(block)

	// Apply state transition on the new block.
	if err := s.onBlock(ctx, blockCopy, blockRoot); err != nil {
		err := errors.Wrap(err, "could not process block")
		traceutil.AnnotateError(span, err)
		return err
	}

	// Send notification of the processed block to the state feed.
	s.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			Slot:      blockCopy.Block.Slot,
			BlockRoot: blockRoot,
			Verified:  true,
		},
	})

	// Handle post block operations such as attestations and exits.
	if err := s.handlePostBlockOperations(blockCopy.Block); err != nil {
		return err
	}

	// Update and save head block after fork choice.
	if err := s.updateHead(ctx, s.getJustifiedBalances()); err != nil {
		return errors.Wrap(err, "could not update head")
	}

	// Reports on block and fork choice metrics.
	reportSlotMetrics(blockCopy.Block.Slot, s.headSlot(), s.CurrentSlot(), s.finalizedCheckpt)

	// Log block sync status.
	logBlockSyncStatus(blockCopy.Block, blockRoot, s.finalizedCheckpt)

	// Log state transition data.
	logStateTransitionData(blockCopy.Block)

	return nil
}

// ReceiveBlockInitialSync processes the input block for the purpose of initial syncing.
// This method should only be used on blocks during initial syncing phase.
func (s *Service) ReceiveBlockInitialSync(ctx context.Context, block *ethpb.SignedBeaconBlock, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.ReceiveBlockNoVerify")
	defer span.End()
	blockCopy := stateTrie.CopySignedBeaconBlock(block)

	// Apply state transition on the new block.
	if err := s.onBlockInitialSyncStateTransition(ctx, blockCopy, blockRoot); err != nil {
		err := errors.Wrap(err, "could not process block")
		traceutil.AnnotateError(span, err)
		return err
	}

	cachedHeadRoot, err := s.HeadRoot(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head root from cache")
	}
	if !bytes.Equal(blockRoot[:], cachedHeadRoot) {
		if err := s.saveHeadNoDB(ctx, blockCopy, blockRoot); err != nil {
			err := errors.Wrap(err, "could not save head")
			traceutil.AnnotateError(span, err)
			return err
		}
	}

	// Send notification of the processed block to the state feed.
	s.stateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			Slot:      blockCopy.Block.Slot,
			BlockRoot: blockRoot,
			Verified:  true,
		},
	})

	// Reports on blockCopy and fork choice metrics.
	reportSlotMetrics(blockCopy.Block.Slot, s.headSlot(), s.CurrentSlot(), s.finalizedCheckpt)

	// Log state transition data.
	log.WithFields(logrus.Fields{
		"slot":         blockCopy.Block.Slot,
		"attestations": len(blockCopy.Block.Body.Attestations),
		"deposits":     len(blockCopy.Block.Body.Deposits),
	}).Debug("Finished applying state transition")

	return nil
}

// ReceiveBlockBatch processes the whole block batch at once, assuming the block batch is linear ,transitioning
// the state, performing batch verification of all collected signatures and then performing the appropriate
// actions for a block post-transition.
func (s *Service) ReceiveBlockBatch(ctx context.Context, blocks []*ethpb.SignedBeaconBlock, blkRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.ReceiveBlockBatch")
	defer span.End()

	// Apply state transition on the incoming newly received blockCopy without verifying its BLS contents.
	postState, fCheckpoints, jCheckpoints, err := s.onBlockBatch(ctx, blocks, blkRoots)
	if err != nil {
		err := errors.Wrap(err, "could not process block")
		traceutil.AnnotateError(span, err)
		return err
	}

	for i, b := range blocks {
		blockCopy := stateTrie.CopySignedBeaconBlock(b)
		if err = s.handleBlockAfterBatchVerify(ctx, blockCopy, blkRoots[i], fCheckpoints[i], jCheckpoints[i]); err != nil {
			traceutil.AnnotateError(span, err)
			return err
		}
		// Send notification of the processed block to the state feed.
		s.stateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.BlockProcessed,
			Data: &statefeed.BlockProcessedData{
				Slot:      blockCopy.Block.Slot,
				BlockRoot: blkRoots[i],
				Verified:  true,
			},
		})

		// Reports on blockCopy and fork choice metrics.
		reportSlotMetrics(blockCopy.Block.Slot, s.headSlot(), s.CurrentSlot(), s.finalizedCheckpt)
	}
	lastBlk := blocks[len(blocks)-1]
	lastRoot := blkRoots[len(blkRoots)-1]

	if err := s.stateGen.SaveState(ctx, lastRoot, postState); err != nil {
		return errors.Wrap(err, "could not save state")
	}

	cachedHeadRoot, err := s.HeadRoot(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head root from cache")
	}

	if !bytes.Equal(lastRoot[:], cachedHeadRoot) {
		if err := s.saveHeadNoDB(ctx, lastBlk, lastRoot); err != nil {
			err := errors.Wrap(err, "could not save head")
			traceutil.AnnotateError(span, err)
			return err
		}
	}

	return s.handleEpochBoundary(postState)
}

// HasInitSyncBlock returns true if the block of the input root exists in initial sync blocks cache.
func (s *Service) HasInitSyncBlock(root [32]byte) bool {
	return s.hasInitSyncBlock(root)
}

func (s *Service) handlePostBlockOperations(b *ethpb.BeaconBlock) error {
	// Delete the processed block attestations from attestation pool.
	if err := s.deletePoolAtts(b.Body.Attestations); err != nil {
		return err
	}

	// Add block attestations to the fork choice pool to compute head.
	if err := s.attPool.SaveBlockAttestations(b.Body.Attestations); err != nil {
		log.Errorf("Could not save block attestations for fork choice: %v", err)
		return nil
	}
	// Mark block exits as seen so we don't include same ones in future blocks.
	for _, e := range b.Body.VoluntaryExits {
		s.exitPool.MarkIncluded(e)
	}

	//  Mark attester slashings as seen so we don't include same ones in future blocks.
	for _, as := range b.Body.AttesterSlashings {
		s.slashingPool.MarkIncludedAttesterSlashing(as)
	}
	return nil
}
