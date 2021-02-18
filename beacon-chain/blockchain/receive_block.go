package blockchain

import (
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// This defines how many epochs since finality the run time will begin to save hot state on to the DB.
var epochsSinceFinalitySaveHotStateDB = types.Epoch(100)

// BlockReceiver interface defines the methods of chain service receive and processing new blocks.
type BlockReceiver interface {
	ReceiveBlock(ctx context.Context, block *ethpb.SignedBeaconBlock, blockRoot [32]byte) error
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
	receivedTime := timeutils.Now()
	blockCopy := stateTrie.CopySignedBeaconBlock(block)

	// Apply state transition on the new block.
	if err := s.onBlock(ctx, blockCopy, blockRoot); err != nil {
		err := errors.Wrap(err, "could not process block")
		traceutil.AnnotateError(span, err)
		return err
	}

	// Update and save head block after fork choice.
	if !featureconfig.Get().UpdateHeadTimely {
		if err := s.updateHead(ctx, s.getJustifiedBalances()); err != nil {
			log.WithError(err).Warn("Could not update head")
		}
		// Send notification of the processed block to the state feed.
		s.stateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.BlockProcessed,
			Data: &statefeed.BlockProcessedData{
				Slot:        blockCopy.Block.Slot,
				BlockRoot:   blockRoot,
				SignedBlock: blockCopy,
				Verified:    true,
			},
		})
	}

	// Handle post block operations such as attestations and exits.
	if err := s.handlePostBlockOperations(blockCopy.Block); err != nil {
		return err
	}

	// Have we been finalizing? Should we start saving hot states to db?
	if err := s.checkSaveHotStateDB(ctx); err != nil {
		return err
	}

	// Reports on block and fork choice metrics.
	reportSlotMetrics(blockCopy.Block.Slot, s.HeadSlot(), s.CurrentSlot(), s.finalizedCheckpt)

	// Log block sync status.
	if err := logBlockSyncStatus(blockCopy.Block, blockRoot, s.finalizedCheckpt, receivedTime, uint64(s.genesisTime.Unix())); err != nil {
		return err
	}
	// Log state transition data.
	logStateTransitionData(blockCopy.Block)

	return nil
}

// ReceiveBlockBatch processes the whole block batch at once, assuming the block batch is linear ,transitioning
// the state, performing batch verification of all collected signatures and then performing the appropriate
// actions for a block post-transition.
func (s *Service) ReceiveBlockBatch(ctx context.Context, blocks []*ethpb.SignedBeaconBlock, blkRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.ReceiveBlockBatch")
	defer span.End()

	// Apply state transition on the incoming newly received blockCopy without verifying its BLS contents.
	fCheckpoints, jCheckpoints, err := s.onBlockBatch(ctx, blocks, blkRoots)
	if err != nil {
		err := errors.Wrap(err, "could not process block in batch")
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
				Slot:        blockCopy.Block.Slot,
				BlockRoot:   blkRoots[i],
				SignedBlock: blockCopy,
				Verified:    true,
			},
		})

		// Reports on blockCopy and fork choice metrics.
		reportSlotMetrics(blockCopy.Block.Slot, s.HeadSlot(), s.CurrentSlot(), s.finalizedCheckpt)
	}

	if err := s.VerifyWeakSubjectivityRoot(s.ctx); err != nil {
		// log.Fatalf will prevent defer from being called
		span.End()
		// Exit run time if the node failed to verify weak subjectivity checkpoint.
		log.Fatalf("Could not verify weak subjectivity checkpoint: %v", err)
	}

	return nil
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

// This checks whether it's time to start saving hot state to DB.
// It's time when there's `epochsSinceFinalitySaveHotStateDB` epochs of non-finality.
func (s *Service) checkSaveHotStateDB(ctx context.Context) error {
	currentEpoch := helpers.SlotToEpoch(s.CurrentSlot())
	// Prevent `sinceFinality` going underflow.
	var sinceFinality types.Epoch
	if currentEpoch > s.finalizedCheckpt.Epoch {
		sinceFinality = currentEpoch - s.finalizedCheckpt.Epoch
	}

	if sinceFinality >= epochsSinceFinalitySaveHotStateDB {
		s.stateGen.EnableSaveHotStateToDB(ctx)
		return nil
	}

	return s.stateGen.DisableSaveHotStateToDB(ctx)
}
