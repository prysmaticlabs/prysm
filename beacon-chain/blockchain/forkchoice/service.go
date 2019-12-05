package forkchoice

import (
	"bytes"
	"context"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// ForkChoicer defines a common interface for methods useful for directly applying fork choice
// to beacon blocks to compute head.
type ForkChoicer interface {
	Head(ctx context.Context) ([]byte, error)
	OnBlock(ctx context.Context, b *ethpb.BeaconBlock) error
	OnBlockInitialSyncStateTransition(ctx context.Context, b *ethpb.BeaconBlock) error
	OnAttestation(ctx context.Context, a *ethpb.Attestation) error
	GenesisStore(ctx context.Context, justifiedCheckpoint *ethpb.Checkpoint, finalizedCheckpoint *ethpb.Checkpoint) error
	FinalizedCheckpt() *ethpb.Checkpoint
}

// Store represents a service struct that handles the forkchoice
// logic of managing the full PoS beacon chain.
type Store struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	db                   db.Database
	justifiedCheckpt     *ethpb.Checkpoint
	finalizedCheckpt     *ethpb.Checkpoint
	prevFinalizedCheckpt *ethpb.Checkpoint
	checkpointState      *cache.CheckpointStateCache
	checkpointStateLock  sync.Mutex
	seenAtts             map[[32]byte]bool
	seenAttsLock         sync.Mutex
	latestVoteMap        map[uint64]*pb.ValidatorLatestVote
	voteLock             sync.RWMutex
	initSyncState        map[[32]byte]*pb.BeaconState
}

// NewForkChoiceService instantiates a new service instance that will
// be registered into a running beacon node.
func NewForkChoiceService(ctx context.Context, db db.Database) *Store {
	ctx, cancel := context.WithCancel(ctx)
	return &Store{
		ctx:             ctx,
		cancel:          cancel,
		db:              db,
		checkpointState: cache.NewCheckpointStateCache(),
		latestVoteMap:   make(map[uint64]*pb.ValidatorLatestVote),
		seenAtts:        make(map[[32]byte]bool),
		initSyncState:   make(map[[32]byte]*pb.BeaconState),
	}
}

// GenesisStore initializes the store struct before beacon chain
// starts to advance.
//
// Spec pseudocode definition:
//   def get_genesis_store(genesis_state: BeaconState) -> Store:
//    genesis_block = BeaconBlock(state_root=hash_tree_root(genesis_state))
//    root = signing_root(genesis_block)
//    justified_checkpoint = Checkpoint(epoch=GENESIS_EPOCH, root=root)
//    finalized_checkpoint = Checkpoint(epoch=GENESIS_EPOCH, root=root)
//    return Store(
//        time=genesis_state.genesis_time,
//        justified_checkpoint=justified_checkpoint,
//        finalized_checkpoint=finalized_checkpoint,
//        blocks={root: genesis_block},
//        block_states={root: genesis_state.copy()},
//        checkpoint_states={justified_checkpoint: genesis_state.copy()},
//    )
func (s *Store) GenesisStore(
	ctx context.Context,
	justifiedCheckpoint *ethpb.Checkpoint,
	finalizedCheckpoint *ethpb.Checkpoint) error {

	s.justifiedCheckpt = proto.Clone(justifiedCheckpoint).(*ethpb.Checkpoint)
	s.finalizedCheckpt = proto.Clone(finalizedCheckpoint).(*ethpb.Checkpoint)
	s.prevFinalizedCheckpt = proto.Clone(finalizedCheckpoint).(*ethpb.Checkpoint)

	justifiedState, err := s.db.State(ctx, bytesutil.ToBytes32(s.justifiedCheckpt.Root))
	if err != nil {
		return errors.Wrap(err, "could not retrieve last justified state")
	}

	if err := s.checkpointState.AddCheckpointState(&cache.CheckpointState{
		Checkpoint: s.justifiedCheckpt,
		State:      justifiedState,
	}); err != nil {
		return errors.Wrap(err, "could not save genesis state in check point cache")
	}

	if featureconfig.Get().InitSyncCacheState {
		genesisState, err := s.db.GenesisState(ctx)
		if err != nil {
			return err
		}
		stateRoot, err := ssz.HashTreeRoot(genesisState)
		if err != nil {
			return errors.Wrap(err, "could not tree hash genesis state")
		}
		genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
		genesisBlkRoot, err := ssz.SigningRoot(genesisBlk)
		if err != nil {
			return errors.Wrap(err, "could not get genesis block root")
		}
		s.initSyncState[genesisBlkRoot] = genesisState
	}

	return nil
}

// ancestor returns the block root of an ancestry block from the input block root.
//
// Spec pseudocode definition:
//   def get_ancestor(store: Store, root: Hash, slot: Slot) -> Hash:
//    block = store.blocks[root]
//    if block.slot > slot:
//      return get_ancestor(store, block.parent_root, slot)
//    elif block.slot == slot:
//      return root
//    else:
//      return Bytes32()  # root is older than queried slot: no results.
func (s *Store) ancestor(ctx context.Context, root []byte, slot uint64) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.ancestor")
	defer span.End()

	b, err := s.db.Block(ctx, bytesutil.ToBytes32(root))
	if err != nil {
		return nil, errors.Wrap(err, "could not get ancestor block")
	}

	// If we dont have the ancestor in the DB, simply return nil so rest of fork choice
	// operation can proceed. This is not an error condition.
	if b == nil || b.Slot < slot {
		return nil, nil
	}

	if b.Slot == slot {
		return root, nil
	}

	return s.ancestor(ctx, b.ParentRoot, slot)
}

// latestAttestingBalance returns the staked balance of a block from the input block root.
//
// Spec pseudocode definition:
//   def get_latest_attesting_balance(store: Store, root: Hash) -> Gwei:
//    state = store.checkpoint_states[store.justified_checkpoint]
//    active_indices = get_active_validator_indices(state, get_current_epoch(state))
//    return Gwei(sum(
//        state.validators[i].effective_balance for i in active_indices
//        if (i in store.latest_messages
//            and get_ancestor(store, store.latest_messages[i].root, store.blocks[root].slot) == root)
//    ))
func (s *Store) latestAttestingBalance(ctx context.Context, root []byte) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.latestAttestingBalance")
	defer span.End()

	lastJustifiedState, err := s.checkpointState.StateByCheckpoint(s.JustifiedCheckpt())
	if err != nil {
		return 0, errors.Wrap(err, "could not retrieve cached state via last justified check point")
	}
	if lastJustifiedState == nil {
		return 0, errors.Wrapf(err, "could not get justified state at epoch %d", s.JustifiedCheckpt().Epoch)
	}

	lastJustifiedEpoch := helpers.CurrentEpoch(lastJustifiedState)
	activeIndices, err := helpers.ActiveValidatorIndices(lastJustifiedState, lastJustifiedEpoch)
	if err != nil {
		return 0, errors.Wrap(err, "could not get active indices for last justified checkpoint")
	}

	wantedBlk, err := s.db.Block(ctx, bytesutil.ToBytes32(root))
	if err != nil {
		return 0, errors.Wrap(err, "could not get target block")
	}

	balances := uint64(0)
	s.voteLock.RLock()
	defer s.voteLock.RUnlock()
	for _, i := range activeIndices {
		vote, ok := s.latestVoteMap[i]
		if !ok {
			continue
		}

		wantedRoot, err := s.ancestor(ctx, vote.Root, wantedBlk.Slot)
		if err != nil {
			return 0, errors.Wrapf(err, "could not get ancestor root for slot %d", wantedBlk.Slot)
		}
		if bytes.Equal(wantedRoot, root) {
			balances += lastJustifiedState.Validators[i].EffectiveBalance
		}
	}
	return balances, nil
}

// Head returns the head of the beacon chain.
//
// Spec pseudocode definition:
//   def get_head(store: Store) -> Hash:
//    # Execute the LMD-GHOST fork choice
//    head = store.justified_checkpoint.root
//    justified_slot = compute_start_slot_of_epoch(store.justified_checkpoint.epoch)
//    while True:
//        children = [
//            root for root in store.blocks.keys()
//            if store.blocks[root].parent_root == head and store.blocks[root].slot > justified_slot
//        ]
//        if len(children) == 0:
//            return head
//        # Sort by latest attesting balance with ties broken lexicographically
//        head = max(children, key=lambda root: (get_latest_attesting_balance(store, root), root))
func (s *Store) Head(ctx context.Context) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.head")
	defer span.End()

	head := s.JustifiedCheckpt().Root

	for {
		startSlot := s.JustifiedCheckpt().Epoch * params.BeaconConfig().SlotsPerEpoch
		filter := filters.NewFilter().SetParentRoot(head).SetStartSlot(startSlot)
		children, err := s.db.BlockRoots(ctx, filter)
		if err != nil {
			return nil, errors.Wrap(err, "could not retrieve children info")
		}

		if len(children) == 0 {
			return head, nil
		}

		// if a block has one child, then we don't have to lookup anything to
		// know that this child will be the best child.
		head = children[0][:]
		if len(children) > 1 {
			highest, err := s.latestAttestingBalance(ctx, head)
			if err != nil {
				return nil, errors.Wrap(err, "could not get latest balance")
			}
			for _, child := range children[1:] {
				balance, err := s.latestAttestingBalance(ctx, child[:])
				if err != nil {
					return nil, errors.Wrap(err, "could not get latest balance")
				}
				// When there's a tie, it's broken lexicographically to favor the higher one.
				if balance > highest ||
					balance == highest && bytes.Compare(child[:], head) > 0 {
					highest = balance
					head = child[:]
				}
			}
		}
	}
}

// JustifiedCheckpt returns the latest justified check point from fork choice store.
func (s *Store) JustifiedCheckpt() *ethpb.Checkpoint {
	return proto.Clone(s.justifiedCheckpt).(*ethpb.Checkpoint)
}

// FinalizedCheckpt returns the latest finalized check point from fork choice store.
func (s *Store) FinalizedCheckpt() *ethpb.Checkpoint {
	return proto.Clone(s.finalizedCheckpt).(*ethpb.Checkpoint)
}
