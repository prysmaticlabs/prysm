package blockchain

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"go.opencensus.io/trace"
)

// This defines how many epochs since finality the run time will begin to save hot state on to the DB.
var epochsSinceFinalitySaveHotStateDB = types.Epoch(100)

// BlockReceiver interface defines the methods of chain service for receiving and processing new blocks.
type BlockReceiver interface {
	ReceiveBlock(ctx context.Context, block interfaces.SignedBeaconBlock, blockRoot [32]byte) error
	ReceiveBlockBatch(ctx context.Context, blocks []interfaces.SignedBeaconBlock, blkRoots [][32]byte) error
	HasBlock(ctx context.Context, root [32]byte) bool
}

// SlashingReceiver interface defines the methods of chain service for receiving validated slashing over the wire.
type SlashingReceiver interface {
	ReceiveAttesterSlashing(ctx context.Context, slashings *ethpb.AttesterSlashing)
}

// ReceiveBlock is a function that defines the operations (minus pubsub)
// that are performed on a received block. The operations consist of:
//   1. Validate block, apply state transition and update checkpoints
//   2. Apply fork choice to the processed block
//   3. Save latest head info
func (s *Service) ReceiveBlock(ctx context.Context, block interfaces.SignedBeaconBlock, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.ReceiveBlock")
	defer span.End()
	receivedTime := time.Now()
	blockCopy, err := block.Copy()
	if err != nil {
		return err
	}

	// Apply state transition on the new block.
	if err := s.onBlock(ctx, blockCopy, blockRoot); err != nil {
		err := errors.Wrap(err, "could not process block")
		tracing.AnnotateError(span, err)
		return err
	}

	// Handle post block operations such as attestations and exits.
	if err := s.handlePostBlockOperations(blockCopy.Block()); err != nil {
		return err
	}

	// Have we been finalizing? Should we start saving hot states to db?
	if err := s.checkSaveHotStateDB(ctx); err != nil {
		return err
	}

	// Reports on block and fork choice metrics.
	finalized := s.FinalizedCheckpt()
	reportSlotMetrics(blockCopy.Block().Slot(), s.HeadSlot(), s.CurrentSlot(), finalized)

	// Log block sync status.
	justified := s.CurrentJustifiedCheckpt()
	if err := logBlockSyncStatus(blockCopy.Block(), blockRoot, justified, finalized, receivedTime, uint64(s.genesisTime.Unix())); err != nil {
		log.WithError(err).Error("Unable to log block sync status")
	}
	// Log payload data
	if err := logPayload(blockCopy.Block()); err != nil {
		log.WithError(err).Error("Unable to log debug block payload data")
	}
	// Log state transition data.
	if err := logStateTransitionData(blockCopy.Block()); err != nil {
		log.WithError(err).Error("Unable to log state transition data")
	}

	return nil
}

// ReceiveBlockBatch processes the whole block batch at once, assuming the block batch is linear ,transitioning
// the state, performing batch verification of all collected signatures and then performing the appropriate
// actions for a block post-transition.
func (s *Service) ReceiveBlockBatch(ctx context.Context, blocks []interfaces.SignedBeaconBlock, blkRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.ReceiveBlockBatch")
	defer span.End()

	// Apply state transition on the incoming newly received block batches, one by one.
	if err := s.onBlockBatch(ctx, blocks, blkRoots); err != nil {
		err := errors.Wrap(err, "could not process block in batch")
		tracing.AnnotateError(span, err)
		return err
	}

	for i, b := range blocks {
		blockCopy, err := b.Copy()
		if err != nil {
			return err
		}
		// Send notification of the processed block to the state feed.
		s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.BlockProcessed,
			Data: &statefeed.BlockProcessedData{
				Slot:        blockCopy.Block().Slot(),
				BlockRoot:   blkRoots[i],
				SignedBlock: blockCopy,
				Verified:    true,
			},
		})

		// Reports on blockCopy and fork choice metrics.
		finalized := s.FinalizedCheckpt()
		reportSlotMetrics(blockCopy.Block().Slot(), s.HeadSlot(), s.CurrentSlot(), finalized)
	}

	if err := s.cfg.BeaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
		return err
	}
	finalized := s.FinalizedCheckpt()
	if finalized == nil {
		return errNilFinalizedInStore
	}
	if err := s.wsVerifier.VerifyWeakSubjectivity(s.ctx, finalized.Epoch); err != nil {
		// log.Fatalf will prevent defer from being called
		span.End()
		// Exit run time if the node failed to verify weak subjectivity checkpoint.
		log.WithError(err).Fatal("Could not verify weak subjectivity checkpoint")
	}

	return nil
}

// HasBlock returns true if the block of the input root exists in initial sync blocks cache or DB.
func (s *Service) HasBlock(ctx context.Context, root [32]byte) bool {
	return s.hasBlockInInitSyncOrDB(ctx, root)
}

// ReceiveAttesterSlashing receives an attester slashing and inserts it to forkchoice
func (s *Service) ReceiveAttesterSlashing(ctx context.Context, slashing *ethpb.AttesterSlashing) {
	s.InsertSlashingsToForkChoiceStore(ctx, []*ethpb.AttesterSlashing{slashing})
}

func (s *Service) handlePostBlockOperations(b interfaces.BeaconBlock) error {
	// Delete the processed block attestations from attestation pool.
	if err := s.deletePoolAtts(b.Body().Attestations()); err != nil {
		return err
	}

	// Mark block exits as seen so we don't include same ones in future blocks.
	for _, e := range b.Body().VoluntaryExits() {
		s.cfg.ExitPool.MarkIncluded(e)
	}

	//  Mark attester slashings as seen so we don't include same ones in future blocks.
	for _, as := range b.Body().AttesterSlashings() {
		s.cfg.SlashingPool.MarkIncludedAttesterSlashing(as)
	}
	return nil
}

// This checks whether it's time to start saving hot state to DB.
// It's time when there's `epochsSinceFinalitySaveHotStateDB` epochs of non-finality.
func (s *Service) checkSaveHotStateDB(ctx context.Context) error {
	currentEpoch := slots.ToEpoch(s.CurrentSlot())
	// Prevent `sinceFinality` going underflow.
	var sinceFinality types.Epoch
	finalized := s.FinalizedCheckpt()
	if finalized == nil {
		return errNilFinalizedInStore
	}
	if currentEpoch > finalized.Epoch {
		sinceFinality = currentEpoch - finalized.Epoch
	}

	if sinceFinality >= epochsSinceFinalitySaveHotStateDB {
		s.cfg.StateGen.EnableSaveHotStateToDB(ctx)
		return nil
	}

	return s.cfg.StateGen.DisableSaveHotStateToDB(ctx)
}
