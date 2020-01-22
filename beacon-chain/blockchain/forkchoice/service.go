package forkchoice

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"

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
	"go.opencensus.io/trace"
)

// ForkChoicer defines a common interface for methods useful for directly applying fork choice
// to beacon blocks to compute head.
type ForkChoicer interface {
	Head(ctx context.Context) ([]byte, error)
	OnBlock(ctx context.Context, b *ethpb.SignedBeaconBlock) error
	OnBlockCacheFilteredTree(ctx context.Context, b *ethpb.SignedBeaconBlock) error
	OnBlockInitialSyncStateTransition(ctx context.Context, b *ethpb.SignedBeaconBlock) error
	OnAttestation(ctx context.Context, a *ethpb.Attestation) error
	GenesisStore(ctx context.Context, justifiedCheckpoint *ethpb.Checkpoint, finalizedCheckpoint *ethpb.Checkpoint) error
	FinalizedCheckpt() *ethpb.Checkpoint
}

// Store represents a service struct that handles the forkchoice
// logic of managing the full PoS beacon chain.
type Store struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	db                    db.HeadAccessDatabase
	justifiedCheckpt      *ethpb.Checkpoint
	finalizedCheckpt      *ethpb.Checkpoint
	prevFinalizedCheckpt  *ethpb.Checkpoint
	checkpointState       *cache.CheckpointStateCache
	checkpointStateLock   sync.Mutex
	genesisTime           uint64
	bestJustifiedCheckpt  *ethpb.Checkpoint
	latestVoteMap         map[uint64]*pb.ValidatorLatestVote
	voteLock              sync.RWMutex
	initSyncState         map[[32]byte]*stateTrie.BeaconState
	initSyncStateLock     sync.RWMutex
	nextEpochBoundarySlot uint64
	filteredBlockTree     map[[32]byte]*ethpb.BeaconBlock
	filteredBlockTreeLock sync.RWMutex
}

// NewForkChoiceService instantiates a new service instance that will
// be registered into a running beacon node.
func NewForkChoiceService(ctx context.Context, db db.HeadAccessDatabase) *Store {
	ctx, cancel := context.WithCancel(ctx)
	return &Store{
		ctx:             ctx,
		cancel:          cancel,
		db:              db,
		checkpointState: cache.NewCheckpointStateCache(),
		latestVoteMap:   make(map[uint64]*pb.ValidatorLatestVote),
		initSyncState:   make(map[[32]byte]*stateTrie.BeaconState),
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
	s.bestJustifiedCheckpt = proto.Clone(justifiedCheckpoint).(*ethpb.Checkpoint)
	s.finalizedCheckpt = proto.Clone(finalizedCheckpoint).(*ethpb.Checkpoint)
	s.prevFinalizedCheckpt = proto.Clone(finalizedCheckpoint).(*ethpb.Checkpoint)

	justifiedState, err := s.db.State(ctx, bytesutil.ToBytes32(s.justifiedCheckpt.Root))
	if err != nil {
		return errors.Wrap(err, "could not retrieve last justified state")
	}

	if err := s.checkpointState.AddCheckpointState(&cache.CheckpointState{
		Checkpoint: s.justifiedCheckpt,
		State:      justifiedState.Clone(),
	}); err != nil {
		return errors.Wrap(err, "could not save genesis state in check point cache")
	}

	s.genesisTime = justifiedState.GenesisTime()
	if err := s.cacheGenesisState(ctx); err != nil {
		return errors.Wrap(err, "could not cache initial sync state")
	}

	return nil
}

// This sets up gensis for initial sync state cache.
func (s *Store) cacheGenesisState(ctx context.Context) error {
	if !featureconfig.Get().InitSyncCacheState {
		return nil
	}

	genesisState, err := s.db.GenesisState(ctx)
	if err != nil {
		return err
	}
	stateRoot := genesisState.HashTreeRoot()
	if err != nil {
		return errors.Wrap(err, "could not tree hash genesis state")
	}
	genesisBlk := blocks.NewGenesisBlock(stateRoot[:])
	genesisBlkRoot, err := ssz.HashTreeRoot(genesisBlk.Block)
	if err != nil {
		return errors.Wrap(err, "could not get genesis block root")
	}
	s.initSyncState[genesisBlkRoot] = genesisState

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

	// Stop recursive ancestry lookup if context is cancelled.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	signed, err := s.db.Block(ctx, bytesutil.ToBytes32(root))
	if err != nil {
		return nil, errors.Wrap(err, "could not get ancestor block")
	}
	if signed == nil || signed.Block == nil {
		return nil, errors.New("nil block")
	}
	b := signed.Block

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
	justfiedState, err := stateTrie.InitializeFromProto(lastJustifiedState)
	if err != nil {
		return 0, err
	}

	lastJustifiedEpoch := helpers.CurrentEpoch(justfiedState)
	activeIndices, err := helpers.ActiveValidatorIndices(justfiedState, lastJustifiedEpoch)
	if err != nil {
		return 0, errors.Wrap(err, "could not get active indices for last justified checkpoint")
	}

	wantedBlkSigned, err := s.db.Block(ctx, bytesutil.ToBytes32(root))
	if err != nil {
		return 0, errors.Wrap(err, "could not get target block")
	}
	if wantedBlkSigned == nil || wantedBlkSigned.Block == nil {
		return 0, errors.New("nil wanted block")
	}
	wantedBlk := wantedBlkSigned.Block

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
//   def get_head(store: Store) -> Root:
//    # Get filtered block tree that only includes viable branches
//    blocks = get_filtered_block_tree(store)
//    # Execute the LMD-GHOST fork choice
//    head = store.justified_checkpoint.root
//    justified_slot = compute_start_slot_at_epoch(store.justified_checkpoint.epoch)
//    while True:
//        children = [
//            root for root in blocks.keys()
//            if blocks[root].parent_root == head and blocks[root].slot > justified_slot
//        ]
//        if len(children) == 0:
//            return head
//        # Sort by latest attesting balance with ties broken lexicographically
//        head = max(children, key=lambda root: (get_latest_attesting_balance(store, root), root))
func (s *Store) Head(ctx context.Context) ([]byte, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.head")
	defer span.End()

	head := s.JustifiedCheckpt().Root
	filteredBlocks := make(map[[32]byte]*ethpb.BeaconBlock)
	var err error
	if featureconfig.Get().EnableBlockTreeCache {
		s.filteredBlockTreeLock.RLock()
		filteredBlocks = s.filteredBlockTree
		s.filteredBlockTreeLock.RUnlock()
	} else {
		filteredBlocks, err = s.getFilterBlockTree(ctx)
		if err != nil {
			return nil, err
		}
	}

	justifiedSlot := helpers.StartSlot(s.justifiedCheckpt.Epoch)
	for {
		children := make([][32]byte, 0, len(filteredBlocks))
		for root, block := range filteredBlocks {
			if bytes.Equal(block.ParentRoot, head) && block.Slot > justifiedSlot {
				children = append(children, root)
			}
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

// getFilterBlockTree retrieves a filtered block tree from store, it only returns branches
// whose leaf state's justified and finalized info agrees with what's in the store.
// Rationale: https://notes.ethereum.org/Fj-gVkOSTpOyUx-zkWjuwg?view
//
// Spec pseudocode definition:
//   def get_filtered_block_tree(store: Store) -> Dict[Root, BeaconBlock]:
//    """
//    Retrieve a filtered block true from ``store``, only returning branches
//    whose leaf state's justified/finalized info agrees with that in ``store``.
//    """
//    base = store.justified_checkpoint.root
//    blocks: Dict[Root, BeaconBlock] = {}
//    filter_block_tree(store, base, blocks)
//    return blocks
func (s *Store) getFilterBlockTree(ctx context.Context) (map[[32]byte]*ethpb.BeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.getFilterBlockTree")
	defer span.End()

	baseRoot := bytesutil.ToBytes32(s.justifiedCheckpt.Root)
	filteredBlocks := make(map[[32]byte]*ethpb.BeaconBlock)
	if _, err := s.filterBlockTree(ctx, baseRoot, filteredBlocks); err != nil {
		return nil, err
	}

	return filteredBlocks, nil
}

// filterBlockTree filters for branches that see latest finalized and justified info as correct on-chain
// before running Head.
//
// Spec pseudocode definition:
//   def filter_block_tree(store: Store, block_root: Root, blocks: Dict[Root, BeaconBlock]) -> bool:
//    block = store.blocks[block_root]
//    children = [
//        root for root in store.blocks.keys()
//        if store.blocks[root].parent_root == block_root
//    ]
//    # If any children branches contain expected finalized/justified checkpoints,
//    # add to filtered block-tree and signal viability to parent.
//    if any(children):
//        filter_block_tree_result = [filter_block_tree(store, child, blocks) for child in children]
//        if any(filter_block_tree_result):
//            blocks[block_root] = block
//            return True
//        return False
//    # If leaf block, check finalized/justified checkpoints as matching latest.
//    head_state = store.block_states[block_root]
//    correct_justified = (
//        store.justified_checkpoint.epoch == GENESIS_EPOCH
//        or head_state.current_justified_checkpoint == store.justified_checkpoint
//    )
//    correct_finalized = (
//        store.finalized_checkpoint.epoch == GENESIS_EPOCH
//        or head_state.finalized_checkpoint == store.finalized_checkpoint
//    )
//    # If expected finalized/justified, add to viable block-tree and signal viability to parent.
//    if correct_justified and correct_finalized:
//        blocks[block_root] = block
//        return True
//    # Otherwise, branch not viable
//    return False
func (s *Store) filterBlockTree(ctx context.Context, blockRoot [32]byte, filteredBlocks map[[32]byte]*ethpb.BeaconBlock) (bool, error) {
	if !s.db.HasState(ctx, blockRoot) {
		return false, nil
	}

	ctx, span := trace.StartSpan(ctx, "forkchoice.filterBlockTree")
	defer span.End()
	signed, err := s.db.Block(ctx, blockRoot)
	if err != nil {
		return false, err
	}
	if signed == nil || signed.Block == nil {
		return false, errors.New("nil block")
	}
	block := signed.Block

	filter := filters.NewFilter().SetParentRoot(blockRoot[:])
	childrenRoots, err := s.db.BlockRoots(ctx, filter)
	if err != nil {
		return false, err
	}

	if len(childrenRoots) != 0 {
		var filtered bool
		for _, childRoot := range childrenRoots {
			didFilter, err := s.filterBlockTree(ctx, childRoot, filteredBlocks)
			if err != nil {
				return false, err
			}
			if didFilter {
				filtered = true
			}
		}
		if filtered {
			filteredBlocks[blockRoot] = block
			return true, nil
		}
		return false, nil
	}

	headState, err := s.db.State(ctx, blockRoot)
	if err != nil {
		return false, err
	}

	if headState == nil {
		return false, fmt.Errorf("no state matching block root %v", hex.EncodeToString(blockRoot[:]))
	}

	correctJustified := s.justifiedCheckpt.Epoch == 0 ||
		proto.Equal(s.justifiedCheckpt, headState.CurrentJustifiedCheckpoint())
	correctFinalized := s.finalizedCheckpt.Epoch == 0 ||
		proto.Equal(s.finalizedCheckpt, headState.FinalizedCheckpoint())
	if correctJustified && correctFinalized {
		filteredBlocks[blockRoot] = block
		return true, nil
	}

	return false, nil
}

// JustifiedCheckpt returns the latest justified check point from fork choice store.
func (s *Store) JustifiedCheckpt() *ethpb.Checkpoint {
	return proto.Clone(s.justifiedCheckpt).(*ethpb.Checkpoint)
}

// FinalizedCheckpt returns the latest finalized check point from fork choice store.
func (s *Store) FinalizedCheckpt() *ethpb.Checkpoint {
	return proto.Clone(s.finalizedCheckpt).(*ethpb.Checkpoint)
}
