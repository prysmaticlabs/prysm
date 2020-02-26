package blockchain

import (
	"context"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// This defines the current chain service's view of head.
type head struct {
	slot  uint64                   // current head slot.
	root  [32]byte                 // current head root.
	block *ethpb.SignedBeaconBlock // current head block.
	state *state.BeaconState       // current head state.
}

// This gets head from the fork choice service and saves head related items
// (ie root, block, state) to the local service cache.
func (s *Service) updateHead(ctx context.Context, balances []uint64) error {
	ctx, span := trace.StartSpan(ctx, "blockchain.updateHead")
	defer span.End()

	// To get the proper head update, a node first checks its best justified
	// can become justified. This is designed to prevent bounce attack and
	// ensure head gets its best justified info.
	if s.bestJustifiedCheckpt.Epoch > s.justifiedCheckpt.Epoch {
		s.justifiedCheckpt = s.bestJustifiedCheckpt
	}

	// Get head from the fork choice service.
	f := s.finalizedCheckpt
	j := s.justifiedCheckpt
	headRoot, err := s.forkChoiceStore.Head(ctx, j.Epoch, bytesutil.ToBytes32(j.Root), balances, f.Epoch)
	if err != nil {
		return err
	}

	// Save head to the local service cache.
	return s.saveHead(ctx, headRoot)
}

// This saves head info to the local service cache, it also saves the
// new head root to the DB.
func (s *Service) saveHead(ctx context.Context, headRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockchain.saveHead")
	defer span.End()

	// Do nothing if head hasn't changed.
	if headRoot == s.headRoot() {
		return nil
	}

	// If the head state is not available, just return nil.
	// There's nothing to cache
	_, cached := s.initSyncState[headRoot]
	if !cached && !s.beaconDB.HasState(ctx, headRoot) {
		return nil
	}

	// Get the new head block from DB.
	newHeadBlock, err := s.beaconDB.Block(ctx, headRoot)
	if err != nil {
		return err
	}
	if newHeadBlock == nil || newHeadBlock.Block == nil {
		return errors.New("cannot save nil head block")
	}

	// Get the new head state from cached state or DB.
	var newHeadState *state.BeaconState
	var exists bool
	newHeadState, exists = s.initSyncState[headRoot]
	if !exists {
		newHeadState, err = s.beaconDB.State(ctx, headRoot)
		if err != nil {
			return errors.Wrap(err, "could not retrieve head state in DB")
		}
		if newHeadState == nil {
			return errors.New("cannot save nil head state")
		}
	}
	if newHeadState == nil {
		return errors.New("cannot save nil head state")
	}

	// Cache the new head info.
	s.setHead(headRoot, newHeadBlock, newHeadState)

	// Save the new head root to DB.
	if err := s.beaconDB.SaveHeadBlockRoot(ctx, headRoot); err != nil {
		return errors.Wrap(err, "could not save head root in DB")
	}

	return nil
}

// This gets called to update canonical root mapping. It does not save head block
// root in DB. With the inception of inital-sync-cache-state flag, it uses finalized
// check point as anchors to resume sync therefore head is no longer needed to be saved on per slot basis.
func (s *Service) saveHeadNoDB(ctx context.Context, b *ethpb.SignedBeaconBlock, r [32]byte) error {
	if b == nil || b.Block == nil {
		return errors.New("cannot save nil head block")
	}

	headState, err := s.beaconDB.State(ctx, r)
	if err != nil {
		return errors.Wrap(err, "could not retrieve head state in DB")
	}
	if headState == nil {
		s.initSyncStateLock.RLock()
		cachedHeadState, ok := s.initSyncState[r]
		if ok {
			headState = cachedHeadState
		}
		s.initSyncStateLock.RUnlock()
	}

	if headState == nil {
		return errors.New("nil head state")
	}

	s.setHead(r, stateTrie.CopySignedBeaconBlock(b), headState)

	return nil
}

// This sets head view object which is used to track the head slot, root, block and state.
func (s *Service) setHead(root [32]byte, block *ethpb.SignedBeaconBlock, state *state.BeaconState) {
	s.headLock.Lock()
	defer s.headLock.Unlock()

	// This does a full copy of the block and state.
	s.head = &head{
		slot:  block.Block.Slot,
		root:  root,
		block: stateTrie.CopySignedBeaconBlock(block),
		state: state.Copy(),
	}
}

// This returns the head slot.
func (s *Service) headSlot() uint64 {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return s.head.slot
}

// This returns the head root.
// It does a full copy on head root for immutability.
func (s *Service) headRoot() [32]byte {
	if s.head == nil {
		return params.BeaconConfig().ZeroHash
	}

	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return s.head.root
}

// This returns the head block.
// It does a full copy on head block for immutability.
func (s *Service) headBlock() *ethpb.SignedBeaconBlock {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return stateTrie.CopySignedBeaconBlock(s.head.block)
}

// This returns the head state.
// It does a full copy on head state for immutability.
func (s *Service) headState() *state.BeaconState {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return s.head.state.Copy()
}

// Returns true if head state exists.
func (s *Service) hasHeadState() bool {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return s.head != nil && s.head.state != nil
}
