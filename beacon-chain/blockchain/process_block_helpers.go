package blockchain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/doubly-linked-tree"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	mathutil "github.com/prysmaticlabs/prysm/v4/math"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"go.opencensus.io/trace"
)

// CurrentSlot returns the current slot based on time.
func (s *Service) CurrentSlot() primitives.Slot {
	return slots.CurrentSlot(uint64(s.genesisTime.Unix()))
}

// getBlockPreState returns the pre state of an incoming block. It uses the parent root of the block
// to retrieve the state in DB. It verifies the pre state's validity and the incoming block
// is in the correct time window.
func (s *Service) getBlockPreState(ctx context.Context, b interfaces.ReadOnlyBeaconBlock) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.getBlockPreState")
	defer span.End()

	// Verify incoming block has a valid pre state.
	if err := s.verifyBlkPreState(ctx, b); err != nil {
		return nil, err
	}

	preState, err := s.cfg.StateGen.StateByRoot(ctx, b.ParentRoot())
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pre state for slot %d", b.Slot())
	}
	if preState == nil || preState.IsNil() {
		return nil, errors.Wrapf(err, "nil pre state for slot %d", b.Slot())
	}

	// Verify block slot time is not from the future.
	if err := slots.VerifyTime(uint64(s.genesisTime.Unix()), b.Slot(), params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
		return nil, err
	}

	// Verify block is later than the finalized epoch slot.
	if err := s.verifyBlkFinalizedSlot(b); err != nil {
		return nil, err
	}

	return preState, nil
}

// verifyBlkPreState validates input block has a valid pre-state.
func (s *Service) verifyBlkPreState(ctx context.Context, b interfaces.ReadOnlyBeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.verifyBlkPreState")
	defer span.End()

	parentRoot := b.ParentRoot()
	// Loosen the check to HasBlock because state summary gets saved in batches
	// during initial syncing. There's no risk given a state summary object is just a
	// subset of the block object.
	if !s.cfg.BeaconDB.HasStateSummary(ctx, parentRoot) && !s.cfg.BeaconDB.HasBlock(ctx, parentRoot) {
		return errors.New("could not reconstruct parent state")
	}

	has, err := s.cfg.StateGen.HasState(ctx, parentRoot)
	if err != nil {
		return err
	}
	if !has {
		if err := s.cfg.BeaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
			return errors.Wrap(err, "could not save initial sync blocks")
		}
		s.clearInitSyncBlocks()
	}
	return nil
}

// verifyBlkFinalizedSlot validates input block is not less than or equal
// to current finalized slot.
func (s *Service) verifyBlkFinalizedSlot(b interfaces.ReadOnlyBeaconBlock) error {
	finalized := s.cfg.ForkChoiceStore.FinalizedCheckpoint()
	finalizedSlot, err := slots.EpochStart(finalized.Epoch)
	if err != nil {
		return err
	}
	if finalizedSlot >= b.Slot() {
		err = fmt.Errorf("block is equal or earlier than finalized block, slot %d < slot %d", b.Slot(), finalizedSlot)
		return invalidBlock{error: err}
	}
	return nil
}

// updateFinalized saves the init sync blocks, finalized checkpoint, migrates
// to cold old states and saves the last validated checkpoint to DB. It returns
// early if the new checkpoint is older than the one on db.
func (s *Service) updateFinalized(ctx context.Context, cp *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.updateFinalized")
	defer span.End()

	// return early if new checkpoint is not newer than the one in DB
	currentFinalized, err := s.cfg.BeaconDB.FinalizedCheckpoint(ctx)
	if err != nil {
		return err
	}
	if cp.Epoch <= currentFinalized.Epoch {
		return nil
	}

	// Blocks need to be saved so that we can retrieve finalized block from
	// DB when migrating states.
	if err := s.cfg.BeaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
		return err
	}
	s.clearInitSyncBlocks()

	if err := s.cfg.BeaconDB.SaveFinalizedCheckpoint(ctx, cp); err != nil {
		return err
	}

	fRoot := bytesutil.ToBytes32(cp.Root)
	optimistic, err := s.cfg.ForkChoiceStore.IsOptimistic(fRoot)
	if err != nil && !errors.Is(err, doublylinkedtree.ErrNilNode) {
		return err
	}
	if !optimistic {
		err = s.cfg.BeaconDB.SaveLastValidatedCheckpoint(ctx, cp)
		if err != nil {
			return err
		}
	}
	go func() {
		// We do not pass in the parent context from the method as this method call
		// is meant to be asynchronous and run in the background rather than being
		// tied to the execution of a block.
		if err := s.cfg.StateGen.MigrateToCold(s.ctx, fRoot); err != nil {
			log.WithError(err).Error("could not migrate to cold")
		}
	}()
	return nil
}

// This retrieves an ancestor root using DB. The look up is recursively looking up DB. Slower than `ancestorByForkChoiceStore`.
func (s *Service) ancestorByDB(ctx context.Context, r [32]byte, slot primitives.Slot) (root [32]byte, err error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.ancestorByDB")
	defer span.End()

	root = [32]byte{}
	// Stop recursive ancestry lookup if context is cancelled.
	if ctx.Err() != nil {
		err = ctx.Err()
		return
	}

	signed, err := s.getBlock(ctx, r)
	if err != nil {
		return root, err
	}
	b := signed.Block()
	if b.Slot() == slot || b.Slot() < slot {
		return r, nil
	}

	return s.ancestorByDB(ctx, b.ParentRoot(), slot)
}

// This retrieves missing blocks from DB (ie. the blocks that couldn't be received over sync) and inserts them to fork choice store.
// This is useful for block tree visualizer and additional vote accounting.
func (s *Service) fillInForkChoiceMissingBlocks(ctx context.Context, blk interfaces.ReadOnlyBeaconBlock,
	fCheckpoint, jCheckpoint *ethpb.Checkpoint) error {
	pendingNodes := make([]*forkchoicetypes.BlockAndCheckpoints, 0)

	// Fork choice only matters from last finalized slot.
	finalized := s.cfg.ForkChoiceStore.FinalizedCheckpoint()
	fSlot, err := slots.EpochStart(finalized.Epoch)
	if err != nil {
		return err
	}
	pendingNodes = append(pendingNodes, &forkchoicetypes.BlockAndCheckpoints{Block: blk,
		JustifiedCheckpoint: jCheckpoint, FinalizedCheckpoint: fCheckpoint})
	// As long as parent node is not in fork choice store, and parent node is in DB.
	root := blk.ParentRoot()
	for !s.cfg.ForkChoiceStore.HasNode(root) && s.cfg.BeaconDB.HasBlock(ctx, root) {
		b, err := s.getBlock(ctx, root)
		if err != nil {
			return err
		}
		if b.Block().Slot() <= fSlot {
			break
		}
		root = b.Block().ParentRoot()
		args := &forkchoicetypes.BlockAndCheckpoints{Block: b.Block(),
			JustifiedCheckpoint: jCheckpoint,
			FinalizedCheckpoint: fCheckpoint}
		pendingNodes = append(pendingNodes, args)
	}
	if len(pendingNodes) == 1 {
		return nil
	}
	if root != s.ensureRootNotZeros(finalized.Root) && !s.cfg.ForkChoiceStore.HasNode(root) {
		return ErrNotDescendantOfFinalized
	}
	return s.cfg.ForkChoiceStore.InsertChain(ctx, pendingNodes)
}

// inserts finalized deposits into our finalized deposit trie.
func (s *Service) insertFinalizedDeposits(ctx context.Context, fRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.insertFinalizedDeposits")
	defer span.End()

	// Update deposit cache.
	finalizedState, err := s.cfg.StateGen.StateByRoot(ctx, fRoot)
	if err != nil {
		return errors.Wrap(err, "could not fetch finalized state")
	}
	// We update the cache up to the last deposit index in the finalized block's state.
	// We can be confident that these deposits will be included in some block
	// because the Eth1 follow distance makes such long-range reorgs extremely unlikely.
	eth1DepositIndex, err := mathutil.Int(finalizedState.Eth1DepositIndex())
	if err != nil {
		return errors.Wrap(err, "could not cast eth1 deposit index")
	}
	// The deposit index in the state is always the index of the next deposit
	// to be included(rather than the last one to be processed). This was most likely
	// done as the state cannot represent signed integers.
	eth1DepositIndex -= 1
	s.cfg.DepositCache.InsertFinalizedDeposits(ctx, int64(eth1DepositIndex))
	// Deposit proofs are only used during state transition and can be safely removed to save space.
	if err = s.cfg.DepositCache.PruneProofs(ctx, int64(eth1DepositIndex)); err != nil {
		return errors.Wrap(err, "could not prune deposit proofs")
	}
	return nil
}

// This ensures that the input root defaults to using genesis root instead of zero hashes. This is needed for handling
// fork choice justification routine.
func (s *Service) ensureRootNotZeros(root [32]byte) [32]byte {
	if root == params.BeaconConfig().ZeroHash {
		return s.originBlockRoot
	}
	return root
}
