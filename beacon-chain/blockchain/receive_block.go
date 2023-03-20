package blockchain

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
)

// This defines how many epochs since finality the run time will begin to save hot state on to the DB.
var epochsSinceFinalitySaveHotStateDB = primitives.Epoch(100)

// BlockReceiver interface defines the methods of chain service for receiving and processing new blocks.
type BlockReceiver interface {
	ReceiveBlock(ctx context.Context, block interfaces.ReadOnlySignedBeaconBlock, blockRoot [32]byte) error
	ReceiveBlockBatch(ctx context.Context, blocks []interfaces.ReadOnlySignedBeaconBlock, blkRoots [][32]byte) error
	HasBlock(ctx context.Context, root [32]byte) bool
}

// SlashingReceiver interface defines the methods of chain service for receiving validated slashing over the wire.
type SlashingReceiver interface {
	ReceiveAttesterSlashing(ctx context.Context, slashings *ethpb.AttesterSlashing)
}

// ReceiveBlock is a function that defines the operations (minus pubsub)
// that are performed on a received block. The operations consist of:
//  1. Validate block, apply state transition and update checkpoints
//  2. Apply fork choice to the processed block
//  3. Save latest head info
func (s *Service) ReceiveBlock(ctx context.Context, block interfaces.ReadOnlySignedBeaconBlock, blockRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.ReceiveBlock")
	defer span.End()
	receivedTime := time.Now()
	blockCopy, err := block.Copy()
	if err != nil {
		return err
	}

	s.cfg.ForkChoiceStore.Lock()
	defer s.cfg.ForkChoiceStore.Unlock()
	// Apply state transition on the new block.
	if err := s.onBlock(ctx, blockCopy, blockRoot); err != nil {
		err := errors.Wrap(err, "could not process block")
		tracing.AnnotateError(span, err)
		return err
	}

	// Handle post block operations such as pruning exits and bls messages if incoming block is the head
	if err := s.prunePostBlockOperationPools(ctx, blockCopy, blockRoot); err != nil {
		log.WithError(err).Error("Could not prune canonical objects from pool ")
	}

	// Have we been finalizing? Should we start saving hot states to db?
	if err := s.checkSaveHotStateDB(ctx); err != nil {
		return err
	}

	// Reports on block and fork choice metrics.
	cp := s.ForkChoicer().FinalizedCheckpoint()
	finalized := &ethpb.Checkpoint{Epoch: cp.Epoch, Root: bytesutil.SafeCopyBytes(cp.Root[:])}
	reportSlotMetrics(blockCopy.Block().Slot(), s.HeadSlot(), s.CurrentSlot(), finalized)

	// Log block sync status.
	cp = s.ForkChoicer().JustifiedCheckpoint()
	justified := &ethpb.Checkpoint{Epoch: cp.Epoch, Root: bytesutil.SafeCopyBytes(cp.Root[:])}
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
func (s *Service) ReceiveBlockBatch(ctx context.Context, blocks []interfaces.ReadOnlySignedBeaconBlock, blkRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.ReceiveBlockBatch")
	defer span.End()

	s.cfg.ForkChoiceStore.Lock()
	defer s.cfg.ForkChoiceStore.Unlock()

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
		cp := s.ForkChoicer().FinalizedCheckpoint()
		finalized := &ethpb.Checkpoint{Epoch: cp.Epoch, Root: bytesutil.SafeCopyBytes(cp.Root[:])}
		reportSlotMetrics(blockCopy.Block().Slot(), s.HeadSlot(), s.CurrentSlot(), finalized)
	}

	if err := s.cfg.BeaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
		return err
	}
	finalized := s.ForkChoicer().FinalizedCheckpoint()
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
	s.ForkChoicer().Lock()
	defer s.ForkChoicer().Unlock()
	s.InsertSlashingsToForkChoiceStore(ctx, []*ethpb.AttesterSlashing{slashing})
}

// prunePostBlockOperationPools only runs on new head otherwise should return a nil.
func (s *Service) prunePostBlockOperationPools(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock, root [32]byte) error {
	headRoot, err := s.HeadRoot(ctx)
	if err != nil {
		return err
	}
	// By comparing the current headroot, that has already gone through forkchoice,
	// we can assume that if equal the current block root is canonical.
	if !bytes.Equal(headRoot, root[:]) {
		return nil
	}

	// Mark block exits as seen so we don't include same ones in future blocks.
	for _, e := range blk.Block().Body().VoluntaryExits() {
		s.cfg.ExitPool.MarkIncluded(e)
	}

	// Mark block BLS changes as seen so we don't include same ones in future blocks.
	if err := s.markIncludedBlockBLSToExecChanges(blk.Block()); err != nil {
		return errors.Wrap(err, "could not process BLSToExecutionChanges")
	}

	//  Mark attester slashings as seen so we don't include same ones in future blocks.
	for _, as := range blk.Block().Body().AttesterSlashings() {
		s.cfg.SlashingPool.MarkIncludedAttesterSlashing(as)
	}
	return nil
}

func (s *Service) markIncludedBlockBLSToExecChanges(headBlock interfaces.ReadOnlyBeaconBlock) error {
	if headBlock.Version() < version.Capella {
		return nil
	}
	changes, err := headBlock.Body().BLSToExecutionChanges()
	if err != nil {
		return errors.Wrap(err, "could not get BLSToExecutionChanges")
	}
	for _, change := range changes {
		s.cfg.BLSToExecPool.MarkIncluded(change)
	}
	return nil
}

// This checks whether it's time to start saving hot state to DB.
// It's time when there's `epochsSinceFinalitySaveHotStateDB` epochs of non-finality.
// Requires a read lock on forkchoice
func (s *Service) checkSaveHotStateDB(ctx context.Context) error {
	currentEpoch := slots.ToEpoch(s.CurrentSlot())
	// Prevent `sinceFinality` going underflow.
	var sinceFinality primitives.Epoch
	finalized := s.ForkChoicer().FinalizedCheckpoint()
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
