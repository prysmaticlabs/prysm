package blockchain

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/protoarray"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	mathutil "github.com/prysmaticlabs/prysm/v3/math"
	"github.com/prysmaticlabs/prysm/v3/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"go.opencensus.io/trace"
)

// CurrentSlot returns the current slot based on time.
func (s *Service) CurrentSlot() types.Slot {
	return slots.CurrentSlot(uint64(s.genesisTime.Unix()))
}

// getBlockPreState returns the pre state of an incoming block. It uses the parent root of the block
// to retrieve the state in DB. It verifies the pre state's validity and the incoming block
// is in the correct time window.
func (s *Service) getBlockPreState(ctx context.Context, b interfaces.BeaconBlock) (state.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.getBlockPreState")
	defer span.End()

	// Verify incoming block has a valid pre state.
	if err := s.verifyBlkPreState(ctx, b); err != nil {
		return nil, err
	}

	preState, err := s.cfg.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(b.ParentRoot()))
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
func (s *Service) verifyBlkPreState(ctx context.Context, b interfaces.BeaconBlock) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.verifyBlkPreState")
	defer span.End()

	parentRoot := bytesutil.ToBytes32(b.ParentRoot())
	// Loosen the check to HasBlock because state summary gets saved in batches
	// during initial syncing. There's no risk given a state summary object is just a
	// a subset of the block object.
	if !s.cfg.BeaconDB.HasStateSummary(ctx, parentRoot) && !s.cfg.BeaconDB.HasBlock(ctx, parentRoot) {
		return errors.New("could not reconstruct parent state")
	}

	if err := s.VerifyFinalizedBlkDescendant(ctx, bytesutil.ToBytes32(b.ParentRoot())); err != nil {
		return err
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

// VerifyFinalizedBlkDescendant validates if input block root is a descendant of the
// current finalized block root.
func (s *Service) VerifyFinalizedBlkDescendant(ctx context.Context, root [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.VerifyFinalizedBlkDescendant")
	defer span.End()
	finalized := s.ForkChoicer().FinalizedCheckpoint()
	fRoot := s.ensureRootNotZeros(finalized.Root)
	fSlot, err := slots.EpochStart(finalized.Epoch)
	if err != nil {
		return err
	}
	bFinalizedRoot, err := s.ancestor(ctx, root[:], fSlot)
	if err != nil {
		return errors.Wrap(err, "could not get finalized block root")
	}
	if bFinalizedRoot == nil {
		return fmt.Errorf("no finalized block known for block %#x", bytesutil.Trunc(root[:]))
	}

	if !bytes.Equal(bFinalizedRoot, fRoot[:]) {
		err := fmt.Errorf("block %#x is not a descendant of the current finalized block slot %d, %#x != %#x",
			bytesutil.Trunc(root[:]), fSlot, bytesutil.Trunc(bFinalizedRoot),
			bytesutil.Trunc(fRoot[:]))
		tracing.AnnotateError(span, err)
		return invalidBlock{error: err}
	}
	return nil
}

// verifyBlkFinalizedSlot validates input block is not less than or equal
// to current finalized slot.
func (s *Service) verifyBlkFinalizedSlot(b interfaces.BeaconBlock) error {
	finalized := s.ForkChoicer().FinalizedCheckpoint()
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
// to cold old states and saves the last validated checkpoint to DB
func (s *Service) updateFinalized(ctx context.Context, cp *ethpb.Checkpoint) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.updateFinalized")
	defer span.End()

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
	if err != nil && err != protoarray.ErrUnknownNodeRoot && err != doublylinkedtree.ErrNilNode {
		return err
	}
	if !optimistic {
		err = s.cfg.BeaconDB.SaveLastValidatedCheckpoint(ctx, cp)
		if err != nil {
			return err
		}
	}
	if err := s.cfg.StateGen.MigrateToCold(ctx, fRoot); err != nil {
		return errors.Wrap(err, "could not migrate to cold")
	}
	return nil
}

// ancestor returns the block root of an ancestry block from the input block root.
//
// Spec pseudocode definition:
//   def get_ancestor(store: Store, root: Root, slot: Slot) -> Root:
//    block = store.blocks[root]
//    if block.slot > slot:
//        return get_ancestor(store, block.parent_root, slot)
//    elif block.slot == slot:
//        return root
//    else:
//        # root is older than queried slot, thus a skip slot. Return most recent root prior to slot
//        return root
func (s *Service) ancestor(ctx context.Context, root []byte, slot types.Slot) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.ancestor")
	defer span.End()

	r := bytesutil.ToBytes32(root)
	// Get ancestor root from fork choice store instead of recursively looking up blocks in DB.
	// This is most optimal outcome.
	ar, err := s.ancestorByForkChoiceStore(ctx, r, slot)
	if err != nil {
		// Try getting ancestor root from DB when failed to retrieve from fork choice store.
		// This is the second line of defense for retrieving ancestor root.
		ar, err = s.ancestorByDB(ctx, r, slot)
		if err != nil {
			return nil, err
		}
	}

	return ar, nil
}

// This retrieves an ancestor root using fork choice store. The look up is looping through the a flat array structure.
func (s *Service) ancestorByForkChoiceStore(ctx context.Context, r [32]byte, slot types.Slot) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.ancestorByForkChoiceStore")
	defer span.End()

	if !s.cfg.ForkChoiceStore.HasParent(r) {
		return nil, errors.New("could not find root in fork choice store")
	}
	root, err := s.cfg.ForkChoiceStore.AncestorRoot(ctx, r, slot)
	return root[:], err
}

// This retrieves an ancestor root using DB. The look up is recursively looking up DB. Slower than `ancestorByForkChoiceStore`.
func (s *Service) ancestorByDB(ctx context.Context, r [32]byte, slot types.Slot) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.ancestorByDB")
	defer span.End()

	// Stop recursive ancestry lookup if context is cancelled.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	signed, err := s.getBlock(ctx, r)
	if err != nil {
		return nil, err
	}
	b := signed.Block()
	if b.Slot() == slot || b.Slot() < slot {
		return r[:], nil
	}

	return s.ancestorByDB(ctx, bytesutil.ToBytes32(b.ParentRoot()), slot)
}

// This retrieves missing blocks from DB (ie. the blocks that couldn't be received over sync) and inserts them to fork choice store.
// This is useful for block tree visualizer and additional vote accounting.
func (s *Service) fillInForkChoiceMissingBlocks(ctx context.Context, blk interfaces.BeaconBlock,
	fCheckpoint, jCheckpoint *ethpb.Checkpoint) error {
	pendingNodes := make([]*forkchoicetypes.BlockAndCheckpoints, 0)

	// Fork choice only matters from last finalized slot.
	finalized := s.ForkChoicer().FinalizedCheckpoint()
	fSlot, err := slots.EpochStart(finalized.Epoch)
	if err != nil {
		return err
	}
	pendingNodes = append(pendingNodes, &forkchoicetypes.BlockAndCheckpoints{Block: blk,
		JustifiedCheckpoint: jCheckpoint, FinalizedCheckpoint: fCheckpoint})
	// As long as parent node is not in fork choice store, and parent node is in DB.
	root := bytesutil.ToBytes32(blk.ParentRoot())
	for !s.cfg.ForkChoiceStore.HasNode(root) && s.cfg.BeaconDB.HasBlock(ctx, root) {
		b, err := s.getBlock(ctx, root)
		if err != nil {
			return err
		}
		if b.Block().Slot() <= fSlot {
			break
		}
		root = bytesutil.ToBytes32(b.Block().ParentRoot())
		args := &forkchoicetypes.BlockAndCheckpoints{Block: b.Block(),
			JustifiedCheckpoint: jCheckpoint,
			FinalizedCheckpoint: fCheckpoint}
		pendingNodes = append(pendingNodes, args)
	}
	if len(pendingNodes) == 1 {
		return nil
	}
	if root != s.ensureRootNotZeros(finalized.Root) {
		return errNotDescendantOfFinalized
	}
	return s.cfg.ForkChoiceStore.InsertOptimisticChain(ctx, pendingNodes)
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

// The deletes input attestations from the attestation pool, so proposers don't include them in a block for the future.
func (s *Service) deletePoolAtts(atts []*ethpb.Attestation) error {
	for _, att := range atts {
		if helpers.IsAggregated(att) {
			if err := s.cfg.AttPool.DeleteAggregatedAttestation(att); err != nil {
				return err
			}
		} else {
			if err := s.cfg.AttPool.DeleteUnaggregatedAttestation(att); err != nil {
				return err
			}
		}
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
