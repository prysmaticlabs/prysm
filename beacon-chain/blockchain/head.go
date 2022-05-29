package blockchain

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice"
	doublylinkedtree "github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/features"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// UpdateAndSaveHeadWithBalances updates the beacon state head after getting justified balanced from cache.
// This function is only used in spec-tests, it does save the head after updating it.
func (s *Service) UpdateAndSaveHeadWithBalances(ctx context.Context) error {
	jp, err := s.store.JustifiedCheckpt()
	if err != nil {
		return err
	}

	balances, err := s.justifiedBalances.get(ctx, bytesutil.ToBytes32(jp.Root))
	if err != nil {
		msg := fmt.Sprintf("could not read balances for state w/ justified checkpoint %#x", jp.Root)
		return errors.Wrap(err, msg)
	}
	headRoot, err := s.updateHead(ctx, balances)
	if err != nil {
		return errors.Wrap(err, "could not update head")
	}
	headBlock, err := s.getBlock(ctx, headRoot)
	if err != nil {
		return err
	}
	headState, err := s.cfg.StateGen.StateByRoot(ctx, headRoot)
	if err != nil {
		return errors.Wrap(err, "could not retrieve head state in DB")
	}
	return s.saveHead(ctx, headRoot, headBlock, headState)
}

// This defines the current chain service's view of head.
type head struct {
	slot  types.Slot                   // current head slot.
	root  [32]byte                     // current head root.
	block interfaces.SignedBeaconBlock // current head block.
	state state.BeaconState            // current head state.
}

// Determined the head from the fork choice service and saves its new data
// (head root, head block, and head state) to the local service cache.
func (s *Service) updateHead(ctx context.Context, balances []uint64) ([32]byte, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.updateHead")
	defer span.End()

	// Get head from the fork choice service.
	f, err := s.store.FinalizedCheckpt()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not get finalized checkpoint")
	}
	j, err := s.store.JustifiedCheckpt()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not get justified checkpoint")
	}
	// To get head before the first justified epoch, the fork choice will start with origin root
	// instead of zero hashes.
	headStartRoot := bytesutil.ToBytes32(j.Root)
	if headStartRoot == params.BeaconConfig().ZeroHash {
		headStartRoot = s.originBlockRoot
	}

	// In order to process head, fork choice store requires justified info.
	// If the fork choice store is missing justified block info, a node should
	// re-initiate fork choice store using the latest justified info.
	// This recovers a fatal condition and should not happen in run time.
	if !s.cfg.ForkChoiceStore.HasNode(headStartRoot) {
		jb, err := s.getBlock(ctx, headStartRoot)
		if err != nil {
			return [32]byte{}, err
		}
		if features.Get().EnableForkChoiceDoublyLinkedTree {
			s.cfg.ForkChoiceStore = doublylinkedtree.New(j.Epoch, f.Epoch)
		} else {
			s.cfg.ForkChoiceStore = protoarray.New(j.Epoch, f.Epoch)
		}
		if err := s.insertBlockToForkChoiceStore(ctx, jb.Block(), headStartRoot, f, j); err != nil {
			return [32]byte{}, err
		}
	}

	return s.cfg.ForkChoiceStore.Head(ctx, headStartRoot, balances)
}

// This saves head info to the local service cache, it also saves the
// new head root to the DB.
func (s *Service) saveHead(ctx context.Context, newHeadRoot [32]byte, headBlock interfaces.SignedBeaconBlock, headState state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.saveHead")
	defer span.End()

	// Do nothing if head hasn't changed.
	oldHeadroot, err := s.HeadRoot(ctx)
	if err != nil {
		return err
	}
	if newHeadRoot == bytesutil.ToBytes32(oldHeadroot) {
		return nil
	}
	if err := wrapper.BeaconBlockIsNil(headBlock); err != nil {
		return err
	}
	if headState == nil || headState.IsNil() {
		return errors.New("cannot save nil head state")
	}

	// If the head state is not available, just return nil.
	// There's nothing to cache
	if !s.cfg.BeaconDB.HasStateSummary(ctx, newHeadRoot) {
		return nil
	}

	// A chain re-org occurred, so we fire an event notifying the rest of the services.
	s.headLock.RLock()
	oldHeadRoot := s.headRoot()
	oldStateRoot := s.headBlock().Block().StateRoot()
	s.headLock.RUnlock()
	headSlot := s.HeadSlot()
	newHeadSlot := headBlock.Block().Slot()
	newStateRoot := headBlock.Block().StateRoot()
	if bytesutil.ToBytes32(headBlock.Block().ParentRoot()) != bytesutil.ToBytes32(oldHeadroot) {
		log.WithFields(logrus.Fields{
			"newSlot": fmt.Sprintf("%d", newHeadSlot),
			"oldSlot": fmt.Sprintf("%d", headSlot),
		}).Debug("Chain reorg occurred")
		absoluteSlotDifference := slots.AbsoluteValueSlotDifference(newHeadSlot, headSlot)
		isOptimistic, err := s.IsOptimistic(ctx)
		if err != nil {
			return errors.Wrap(err, "could not check if node is optimistically synced")
		}
		s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.Reorg,
			Data: &ethpbv1.EventChainReorg{
				Slot:                newHeadSlot,
				Depth:               absoluteSlotDifference,
				OldHeadBlock:        oldHeadRoot[:],
				NewHeadBlock:        newHeadRoot[:],
				OldHeadState:        oldStateRoot,
				NewHeadState:        newStateRoot,
				Epoch:               slots.ToEpoch(newHeadSlot),
				ExecutionOptimistic: isOptimistic,
			},
		})

		if err := s.saveOrphanedAtts(ctx, bytesutil.ToBytes32(oldHeadroot), newHeadRoot); err != nil {
			return err
		}
		reorgCount.Inc()
	}

	// Cache the new head info.
	s.setHead(newHeadRoot, headBlock, headState)

	// Save the new head root to DB.
	if err := s.cfg.BeaconDB.SaveHeadBlockRoot(ctx, newHeadRoot); err != nil {
		return errors.Wrap(err, "could not save head root in DB")
	}

	// Forward an event capturing a new chain head over a common event feed
	// done in a goroutine to avoid blocking the critical runtime main routine.
	go func() {
		if err := s.notifyNewHeadEvent(ctx, newHeadSlot, headState, newStateRoot, newHeadRoot[:]); err != nil {
			log.WithError(err).Error("Could not notify event feed of new chain head")
		}
	}()

	return nil
}

// This gets called to update canonical root mapping. It does not save head block
// root in DB. With the inception of initial-sync-cache-state flag, it uses finalized
// check point as anchors to resume sync therefore head is no longer needed to be saved on per slot basis.
func (s *Service) saveHeadNoDB(ctx context.Context, b interfaces.SignedBeaconBlock, r [32]byte, hs state.BeaconState) error {
	if err := wrapper.BeaconBlockIsNil(b); err != nil {
		return err
	}
	cachedHeadRoot, err := s.HeadRoot(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head root from cache")
	}
	if bytes.Equal(r[:], cachedHeadRoot) {
		return nil
	}

	s.setHeadInitialSync(r, b.Copy(), hs)
	return nil
}

// This sets head view object which is used to track the head slot, root, block and state.
func (s *Service) setHead(root [32]byte, block interfaces.SignedBeaconBlock, state state.BeaconState) {
	s.headLock.Lock()
	defer s.headLock.Unlock()

	// This does a full copy of the block and state.
	s.head = &head{
		slot:  block.Block().Slot(),
		root:  root,
		block: block.Copy(),
		state: state.Copy(),
	}
}

// This sets head view object which is used to track the head slot, root, block and state. The method
// assumes that state being passed into the method will not be modified by any other alternate
// caller which holds the state's reference.
func (s *Service) setHeadInitialSync(root [32]byte, block interfaces.SignedBeaconBlock, state state.BeaconState) {
	s.headLock.Lock()
	defer s.headLock.Unlock()

	// This does a full copy of the block only.
	s.head = &head{
		slot:  block.Block().Slot(),
		root:  root,
		block: block.Copy(),
		state: state,
	}
}

// This returns the head slot.
// This is a lock free version.
func (s *Service) headSlot() types.Slot {
	return s.head.slot
}

// This returns the head root.
// It does a full copy on head root for immutability.
// This is a lock free version.
func (s *Service) headRoot() [32]byte {
	if s.head == nil {
		return params.BeaconConfig().ZeroHash
	}

	return s.head.root
}

// This returns the head block.
// It does a full copy on head block for immutability.
// This is a lock free version.
func (s *Service) headBlock() interfaces.SignedBeaconBlock {
	return s.head.block.Copy()
}

// This returns the head state.
// It does a full copy on head state for immutability.
// This is a lock free version.
func (s *Service) headState(ctx context.Context) state.BeaconState {
	_, span := trace.StartSpan(ctx, "blockChain.headState")
	defer span.End()

	return s.head.state.Copy()
}

// This returns the genesis validators root of the head state.
// This is a lock free version.
func (s *Service) headGenesisValidatorsRoot() [32]byte {
	return bytesutil.ToBytes32(s.head.state.GenesisValidatorsRoot())
}

// This returns the validator referenced by the provided index in
// the head state.
// This is a lock free version.
func (s *Service) headValidatorAtIndex(index types.ValidatorIndex) (state.ReadOnlyValidator, error) {
	return s.head.state.ValidatorAtIndexReadOnly(index)
}

// This returns the validator index referenced by the provided pubkey in
// the head state.
// This is a lock free version.
func (s *Service) headValidatorIndexAtPubkey(pubKey [fieldparams.BLSPubkeyLength]byte) (types.ValidatorIndex, bool) {
	return s.head.state.ValidatorIndexByPubkey(pubKey)
}

// Returns true if head state exists.
// This is the lock free version.
func (s *Service) hasHeadState() bool {
	return s.head != nil && s.head.state != nil
}

// Notifies a common event feed of a new chain head event. Called right after a new
// chain head is determined, set, and saved to disk.
func (s *Service) notifyNewHeadEvent(
	ctx context.Context,
	newHeadSlot types.Slot,
	newHeadState state.BeaconState,
	newHeadStateRoot,
	newHeadRoot []byte,
) error {
	previousDutyDependentRoot := s.originBlockRoot[:]
	currentDutyDependentRoot := s.originBlockRoot[:]

	var previousDutyEpoch types.Epoch
	currentDutyEpoch := slots.ToEpoch(newHeadSlot)
	if currentDutyEpoch > 0 {
		previousDutyEpoch = currentDutyEpoch.Sub(1)
	}
	currentDutySlot, err := slots.EpochStart(currentDutyEpoch)
	if err != nil {
		return errors.Wrap(err, "could not get duty slot")
	}
	previousDutySlot, err := slots.EpochStart(previousDutyEpoch)
	if err != nil {
		return errors.Wrap(err, "could not get duty slot")
	}
	if currentDutySlot > 0 {
		currentDutyDependentRoot, err = helpers.BlockRootAtSlot(newHeadState, currentDutySlot-1)
		if err != nil {
			return errors.Wrap(err, "could not get duty dependent root")
		}
	}
	if previousDutySlot > 0 {
		previousDutyDependentRoot, err = helpers.BlockRootAtSlot(newHeadState, previousDutySlot-1)
		if err != nil {
			return errors.Wrap(err, "could not get duty dependent root")
		}
	}
	isOptimistic, err := s.IsOptimistic(ctx)
	if err != nil {
		return errors.Wrap(err, "could not check if node is optimistically synced")
	}
	s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.NewHead,
		Data: &ethpbv1.EventHead{
			Slot:                      newHeadSlot,
			Block:                     newHeadRoot,
			State:                     newHeadStateRoot,
			EpochTransition:           slots.IsEpochStart(newHeadSlot),
			PreviousDutyDependentRoot: previousDutyDependentRoot,
			CurrentDutyDependentRoot:  currentDutyDependentRoot,
			ExecutionOptimistic:       isOptimistic,
		},
	})
	return nil
}

// This saves the attestations between `orphanedRoot` and the common ancestor root that is derived using `newHeadRoot`.
// It also filters out the attestations that is one epoch older as a defense so invalid attestations don't flow into the attestation pool.
func (s *Service) saveOrphanedAtts(ctx context.Context, orphanedRoot [32]byte, newHeadRoot [32]byte) error {
	commonAncestorRoot, err := s.ForkChoicer().CommonAncestorRoot(ctx, newHeadRoot, orphanedRoot)
	switch {
	// Exit early if there's no common ancestor as there would be nothing to save.
	case errors.Is(err, forkchoice.ErrUnknownCommonAncestor):
		return nil
	case err != nil:
		return err
	}
	for orphanedRoot != commonAncestorRoot {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		orphanedBlk, err := s.getBlock(ctx, orphanedRoot)
		if err != nil {
			return err
		}
		// If the block is an epoch older, break out of the loop since we can't include atts anyway.
		// This prevents stuck within this for loop longer than necessary.
		if orphanedBlk.Block().Slot()+params.BeaconConfig().SlotsPerEpoch <= s.CurrentSlot() {
			break
		}
		for _, a := range orphanedBlk.Block().Body().Attestations() {
			// if the attestation is one epoch older, it wouldn't been useful to save it.
			if a.Data.Slot+params.BeaconConfig().SlotsPerEpoch < s.CurrentSlot() {
				continue
			}
			if helpers.IsAggregated(a) {
				if err := s.cfg.AttPool.SaveAggregatedAttestation(a); err != nil {
					return err
				}
			} else {
				if err := s.cfg.AttPool.SaveUnaggregatedAttestation(a); err != nil {
					return err
				}
			}
			saveOrphanedAttCount.Inc()
		}
		orphanedRoot = bytesutil.ToBytes32(orphanedBlk.Block().ParentRoot())
	}
	return nil
}
