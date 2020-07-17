package blockchain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/forkchoice/protoarray"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// This defines the current chain service's view of head.
type head struct {
	slot  uint64                   // current head slot.
	root  [32]byte                 // current head root.
	block *ethpb.SignedBeaconBlock // current head block.
	state *state.BeaconState       // current head state.
}

// Determined the head from the fork choice service and saves its new data
// (head root, head block, and head state) to the local service cache.
func (s *Service) updateHead(ctx context.Context, balances []uint64) error {
	ctx, span := trace.StartSpan(ctx, "blockchain.updateHead")
	defer span.End()

	// To get the proper head update, a node first checks its best justified
	// can become justified. This is designed to prevent bounce attack and
	// ensure head gets its best justified info.
	if s.bestJustifiedCheckpt.Epoch > s.justifiedCheckpt.Epoch {
		s.justifiedCheckpt = s.bestJustifiedCheckpt
		if err := s.cacheJustifiedStateBalances(ctx, bytesutil.ToBytes32(s.justifiedCheckpt.Root)); err != nil {
			return err
		}
	}

	// Get head from the fork choice service.
	f := s.finalizedCheckpt
	j := s.justifiedCheckpt
	// To get head before the first justified epoch, the fork choice will start with genesis root
	// instead of zero hashes.
	headStartRoot := bytesutil.ToBytes32(j.Root)
	if headStartRoot == params.BeaconConfig().ZeroHash {
		headStartRoot = s.genesisRoot
	}
	headRoot, err := s.forkChoiceStore.Head(ctx, j.Epoch, headStartRoot, balances, f.Epoch)
	if err != nil {
		return err
	}

	if err := s.updateRecentCanonicalBlocks(ctx, headRoot); err != nil {
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
	if !s.stateGen.StateSummaryExists(ctx, headRoot) {
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
	newHeadState, err := s.stateGen.StateByRoot(ctx, headRoot)
	if err != nil {
		return errors.Wrap(err, "could not retrieve head state in DB")
	}
	if newHeadState == nil {
		return errors.New("cannot save nil head state")
	}

	// A chain re-org occurred, so we fire an event notifying the rest of the services.
	if bytesutil.ToBytes32(newHeadBlock.Block.ParentRoot) != s.headRoot() {
		log.WithFields(logrus.Fields{
			"newSlot": fmt.Sprintf("%d", newHeadBlock.Block.Slot),
			"oldSlot": fmt.Sprintf("%d", s.headSlot()),
		}).Debug("Chain reorg occurred")
		s.stateNotifier.StateFeed().Send(&feed.Event{
			Type: statefeed.Reorg,
			Data: &statefeed.ReorgData{
				NewSlot: newHeadBlock.Block.Slot,
				OldSlot: s.headSlot(),
			},
		})

		reorgCount.Inc()
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
// root in DB. With the inception of initial-sync-cache-state flag, it uses finalized
// check point as anchors to resume sync therefore head is no longer needed to be saved on per slot basis.
func (s *Service) saveHeadNoDB(ctx context.Context, b *ethpb.SignedBeaconBlock, r [32]byte) error {
	if b == nil || b.Block == nil {
		return errors.New("cannot save nil head block")
	}

	headState, err := s.stateGen.StateByRootInitialSync(ctx, r)
	if err != nil {
		return errors.Wrap(err, "could not retrieve head state in DB")
	}
	if headState == nil {
		return errors.New("nil head state")
	}

	s.setHeadInitialSync(r, stateTrie.CopySignedBeaconBlock(b), headState)

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

// This sets head view object which is used to track the head slot, root, block and state. The method
// assumes that state being passed into the method will not be modified by any other alternate
// caller which holds the state's reference.
func (s *Service) setHeadInitialSync(root [32]byte, block *ethpb.SignedBeaconBlock, state *state.BeaconState) {
	s.headLock.Lock()
	defer s.headLock.Unlock()

	// This does a full copy of the block only.
	s.head = &head{
		slot:  block.Block.Slot,
		root:  root,
		block: stateTrie.CopySignedBeaconBlock(block),
		state: state,
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
func (s *Service) headState(ctx context.Context) *stateTrie.BeaconState {
	ctx, span := trace.StartSpan(ctx, "blockchain.headState")
	defer span.End()

	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return s.head.state.Copy()
}

// This returns the genesis validator root of the head state.
func (s *Service) headGenesisValidatorRoot() [32]byte {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return bytesutil.ToBytes32(s.head.state.GenesisValidatorRoot())
}

// Returns true if head state exists.
func (s *Service) hasHeadState() bool {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return s.head != nil && s.head.state != nil
}

// This updates recent canonical block mapping. It uses input head root and retrieves
// all the canonical block roots that are ancestor of the input head block root.
func (s *Service) updateRecentCanonicalBlocks(ctx context.Context, headRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockchain.updateRecentCanonicalBlocks")
	defer span.End()

	s.recentCanonicalBlocksLock.Lock()
	defer s.recentCanonicalBlocksLock.Unlock()

	s.recentCanonicalBlocks = make(map[[32]byte]bool)
	s.recentCanonicalBlocks[headRoot] = true
	nodes := s.forkChoiceStore.Nodes()
	node := s.forkChoiceStore.Node(headRoot)
	if node == nil {
		return nil
	}

	for node.Parent != protoarray.NonExistentNode {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		node = nodes[node.Parent]
		s.recentCanonicalBlocks[node.Root] = true
	}

	return nil
}

// This caches justified state balances to be used for fork choice.
func (s *Service) cacheJustifiedStateBalances(ctx context.Context, justifiedRoot [32]byte) error {
	if err := s.beaconDB.SaveBlocks(ctx, s.getInitSyncBlocks()); err != nil {
		return err
	}

	s.clearInitSyncBlocks()
	justifiedState, err := s.stateGen.StateByRoot(ctx, justifiedRoot)
	if err != nil {
		return err
	}

	epoch := helpers.CurrentEpoch(justifiedState)
	validators := justifiedState.Validators()
	justifiedBalances := make([]uint64, len(validators))
	for i, validator := range validators {
		if helpers.IsActiveValidator(validator, epoch) {
			justifiedBalances[i] = validator.EffectiveBalance
		} else {
			justifiedBalances[i] = 0
		}
	}

	s.justifiedBalancesLock.Lock()
	defer s.justifiedBalancesLock.Unlock()
	s.justifiedBalances = justifiedBalances
	return nil
}

func (s *Service) getJustifiedBalances() []uint64 {
	s.justifiedBalancesLock.RLock()
	defer s.justifiedBalancesLock.RUnlock()
	return s.justifiedBalances
}
