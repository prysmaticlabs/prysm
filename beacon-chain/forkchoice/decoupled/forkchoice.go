package decoupled

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/forkchoice"
	doublylinkedtree "github.com/OffchainLabs/prysm/v7/beacon-chain/forkchoice/doubly-linked-tree"
	forkchoicetypes "github.com/OffchainLabs/prysm/v7/beacon-chain/forkchoice/types"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/state"
	consensusblocks "github.com/OffchainLabs/prysm/v7/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	prysmTime "github.com/OffchainLabs/prysm/v7/time"
	"github.com/pkg/errors"
)

var _ forkchoice.ForkChoicer = (*ForkChoice)(nil)

// ForkChoice answers from the old tree until Activate and from the Goldilocks store after, callers hold its lock.
type ForkChoice struct {
	sync.RWMutex
	legacy *doublylinkedtree.ForkChoice
	store  atomic.Pointer[Store]
	exec   map[[32]byte]*payload
	// Readers see what Prysm saved as head, computing a fresh head would tick under a read lock.
	head, confirmed [32]byte
	highestSlot     primitives.Slot
	highestRoot     [32]byte
	ancestorByDB    func(context.Context, [32]byte, primitives.Slot) ([32]byte, error)
}

// payload is a block's execution commitment, read from its bid since Gloas.
type payload struct {
	blockHash, parentHash [32]byte
	builder               primitives.BuilderIndex
	gasLimit              uint64
	preGloas, arrived     bool
}

// NewForkChoice returns a fork choice that uses the old tree until Activate.
func NewForkChoice() *ForkChoice {
	return &ForkChoice{legacy: doublylinkedtree.New()}
}

// Decoupled returns the Goldilocks store after Activate and nil before. It needs no lock.
func (f *ForkChoice) Decoupled() *Store { return f.store.Load() }

// SetPublished records the head and confirmed blocks the blockchain service saved.
func (f *ForkChoice) SetPublished(head, confirmed [32]byte) { f.head, f.confirmed = head, confirmed }

// PublishedConfirmed returns the confirmed block published with the head.
func (f *ForkChoice) PublishedConfirmed() [32]byte { return f.confirmed }

// SetAncestorByDB sets the lookup for ancestors below F, which the store has pruned.
func (f *ForkChoice) SetAncestorByDB(fn func(context.Context, [32]byte, primitives.Slot) ([32]byte, error)) {
	f.ancestorByDB = fn
}

// Activate switches to st for good. chain starts at st's finalized block with parents first, and nothing switches on error.
func (f *ForkChoice) Activate(st *Store, head [32]byte, chain []*forkchoicetypes.BlockAndCheckpoints) error {
	if len(chain) == 0 || chain[0].Block.Root() != st.finalized.Root {
		return errors.New("chain does not start at the finalized block")
	}
	root, err := payloadOf(chain[0].Block.Block())
	if err != nil {
		return err
	}
	root.arrived = root.arrived || chain[0].HasPayload
	f.exec = map[[32]byte]*payload{st.finalized.Root: root}
	f.highestSlot, f.highestRoot = st.finalized.Slot, st.finalized.Root
	for _, c := range chain[1:] {
		if err := f.add(st, c.Block, c.FG, c.HasPayload); err != nil {
			return err
		}
	}
	f.head, f.confirmed = head, st.finalized.Root
	f.store.Store(st)
	// Frees the old tree's per-validator votes and balances, an empty tree avoids nil checks.
	f.legacy = doublylinkedtree.New()
	return nil
}

// add takes F's summary for blocks before the fork, which carry no votes.
func (f *ForkChoice) add(st *Store, roblock consensusblocks.ROBlock, fg *FGSummary, arrived bool) error {
	root := roblock.Root()
	if p := f.exec[root]; p != nil {
		p.arrived = p.arrived || arrived
		return nil
	}
	blk := roblock.Block()
	b := Block{Root: root, Parent: blk.ParentRoot(), Slot: blk.Slot(), FG: st.finalized.FG}
	if roblock.Version() >= version.Decoupled {
		if fg == nil {
			return errors.Errorf("no finality summary for block %#x", root)
		}
		var err error
		if b.Votes, b.Supported, err = blockVotes(blk.Body()); err != nil {
			return err
		}
		b.FG = *fg
	}
	p, err := payloadOf(blk)
	if err != nil {
		return err
	}
	if err := st.InsertNode(b, prysmTime.Now()); err != nil {
		return err
	}
	p.arrived = p.arrived || arrived
	f.exec[root] = p
	if b.Slot > f.highestSlot {
		f.highestSlot, f.highestRoot = b.Slot, root
	}
	f.prune(st)
	return nil
}

// prune runs after any insert or removal, since inserting can move F.
func (f *ForkChoice) prune(st *Store) {
	if len(f.exec) != len(st.nodes) {
		for r := range f.exec {
			if st.nodes[r] == nil {
				delete(f.exec, r)
			}
		}
	}
	if st.nodes[f.highestRoot] != nil {
		return
	}
	f.highestSlot, f.highestRoot = st.finalized.Slot, st.finalized.Root
	for _, n := range st.nodes {
		if n.Slot > f.highestSlot {
			f.highestSlot, f.highestRoot = n.Slot, n.Root
		}
	}
}

// Summary reads a post-state's FG pairs, root fills the zero entry root a block's own state carries.
func Summary(st state.ReadOnlyBeaconState, root [32]byte) (FGSummary, error) {
	var fg FGSummary
	height, err := st.Height()
	if err != nil {
		return fg, err
	}
	entry, err := st.Entry()
	if err != nil {
		return fg, err
	}
	fg.Target = Pair{Height: height, Root: bytesutil.ToBytes32(entry.Root)}
	if bytesutil.ZeroRoot(entry.Root) {
		fg.Target.Root = root
	}
	if fg.Justified.Height, err = st.JustifiedHeight(); err != nil {
		return fg, err
	}
	justified, err := st.Justified()
	if err != nil {
		return fg, err
	}
	fg.Justified.Root = bytesutil.ToBytes32(justified.Root)
	if fg.Finalized.Height, err = st.FinalizedHeight(); err != nil {
		return fg, err
	}
	finalized, err := st.Finalized()
	if err != nil {
		return fg, err
	}
	fg.Finalized.Root = bytesutil.ToBytes32(finalized.Root)
	return fg, nil
}

// blockVotes expands AC aggregates, bit i is validator i. A support bit selects every vote in its aggregate.
func blockVotes(body interfaces.ReadOnlyBeaconBlockBody) (votes, support []Vote, err error) {
	atts, err := body.AvailableAttestations()
	if err != nil {
		return nil, nil, err
	}
	marks, err := body.SupportVotes()
	if err != nil {
		return nil, nil, err
	}
	for i, att := range atts {
		if att.Data == nil {
			continue
		}
		root := bytesutil.ToBytes32(att.Data.BeaconBlockRoot)
		start := len(votes)
		for _, v := range att.AggregationBits.BitIndices() {
			votes = append(votes, Vote{Validator: primitives.ValidatorIndex(v), Slot: att.Data.Slot, Root: root})
		}
		if marks.BitAt(uint64(i)) {
			support = append(support, votes[start:]...)
		}
	}
	return votes, support, nil
}

// payloadOf treats a payload before Gloas as arrived, it is part of the block.
func payloadOf(blk interfaces.ReadOnlyBeaconBlock) (*payload, error) {
	if blk.Version() >= version.Gloas {
		sb, err := blk.Body().SignedExecutionPayloadBid()
		if err != nil {
			return nil, err
		}
		if sb == nil || sb.Message == nil {
			return nil, errors.New("nil execution payload bid")
		}
		bid := sb.Message
		return &payload{
			blockHash:  bytesutil.ToBytes32(bid.BlockHash),
			parentHash: bytesutil.ToBytes32(bid.ParentBlockHash),
			builder:    bid.BuilderIndex,
			gasLimit:   bid.GasLimit,
		}, nil
	}
	p := &payload{preGloas: true, arrived: true}
	if blk.Version() < version.Bellatrix {
		return p, nil
	}
	e, err := blk.Body().Execution()
	if err != nil {
		return nil, err
	}
	p.blockHash, p.parentHash, p.gasLimit = bytesutil.ToBytes32(e.BlockHash()), bytesutil.ToBytes32(e.ParentHash()), e.GasLimit()
	return p, nil
}

// Head returns the Goldilocks head without publishing it.
func (f *ForkChoice) Head(ctx context.Context) ([32]byte, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.Head(ctx)
	}
	return st.Head(prysmTime.Now()).Root, nil
}

// FullHead returns the head and its block hash.
func (f *ForkChoice) FullHead(ctx context.Context) ([32]byte, [32]byte, bool, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.FullHead(ctx)
	}
	root := st.Head(prysmTime.Now()).Root
	var hash [32]byte
	if p := f.exec[root]; p != nil {
		hash = p.blockHash
	}
	// TODO(DC): full or empty from PTC and payload arrival.
	return root, hash, true, nil
}

// InsertNode inserts a validated block, with its finality summary read from its post-state.
func (f *ForkChoice) InsertNode(ctx context.Context, bs state.BeaconState, roblock consensusblocks.ROBlock) error {
	st := f.store.Load()
	if st == nil {
		return f.legacy.InsertNode(ctx, bs, roblock)
	}
	var fg *FGSummary
	if roblock.Version() >= version.Decoupled {
		s, err := Summary(bs, roblock.Root())
		if err != nil {
			return err
		}
		fg = &s
	}
	return f.add(st, roblock, fg, false)
}

// InsertChain inserts blocks oldest first. Decoupled blocks need their FG summary.
func (f *ForkChoice) InsertChain(ctx context.Context, chain []*forkchoicetypes.BlockAndCheckpoints) error {
	st := f.store.Load()
	if st == nil {
		return f.legacy.InsertChain(ctx, chain)
	}
	for _, c := range chain {
		if err := f.add(st, c.Block, c.FG, c.HasPayload); err != nil {
			return err
		}
	}
	return nil
}

// InsertPayload marks the envelope's block as having its payload.
func (f *ForkChoice) InsertPayload(pe interfaces.ROExecutionPayloadEnvelope) error {
	if f.store.Load() == nil {
		return f.legacy.InsertPayload(pe)
	}
	p := f.exec[pe.BeaconBlockRoot()]
	if p == nil {
		return errors.Wrap(doublylinkedtree.ErrNilNode, "cannot insert payload for unknown block")
	}
	p.arrived = true
	return nil
}

// MarkFullNode marks root as having its payload. The bid already fixed the gas limit.
func (f *ForkChoice) MarkFullNode(root [32]byte, gasLimit uint64) {
	if f.store.Load() == nil {
		f.legacy.MarkFullNode(root, gasLimit)
		return
	}
	if p := f.exec[root]; p != nil {
		p.arrived = true
	}
}

// ProcessAttestation does nothing after the fork, there are no Gasper votes.
func (f *ForkChoice) ProcessAttestation(ctx context.Context, validatorIndices []uint64, blockRoot [32]byte, slot primitives.Slot, payloadStatus bool) {
	if f.store.Load() == nil {
		f.legacy.ProcessAttestation(ctx, validatorIndices, blockRoot, slot, payloadStatus)
	}
}

// SetOptimisticToValid does nothing after the fork.
func (f *ForkChoice) SetOptimisticToValid(ctx context.Context, root [32]byte) error {
	if f.store.Load() == nil {
		return f.legacy.SetOptimisticToValid(ctx, root)
	}
	// TODO(DC): optimistic status.
	return nil
}

// SetOptimisticToInvalid removes root's subtree after the fork and returns every removed root.
func (f *ForkChoice) SetOptimisticToInvalid(ctx context.Context, root, parentRoot, parentHash, lastValidHash [32]byte) ([][32]byte, error) {
	st := f.store.Load()
	if st == nil {
		return f.legacy.SetOptimisticToInvalid(ctx, root, parentRoot, parentHash, lastValidHash)
	}
	// TODO(DC): walk back to the latest valid hash.
	n := st.nodes[root]
	if n == nil {
		return nil, nil
	}
	var removed [][32]byte
	blocks := []*Node{n}
	for i := 0; i < len(blocks); i++ {
		removed = append(removed, blocks[i].Root)
		blocks = append(blocks, blocks[i].Children...)
	}
	if err := st.Remove([][32]byte{root}, prysmTime.Now()); err != nil {
		return nil, err
	}
	f.prune(st)
	if !st.HasNode(f.head) {
		f.head = n.Parent.Root
	}
	if !st.HasNode(f.confirmed) {
		f.confirmed = n.Parent.Root
	}
	return removed, nil
}

// UpdateJustifiedCheckpoint does nothing after the fork.
func (f *ForkChoice) UpdateJustifiedCheckpoint(ctx context.Context, cp *forkchoicetypes.Checkpoint) error {
	if f.store.Load() == nil {
		return f.legacy.UpdateJustifiedCheckpoint(ctx, cp)
	}
	return nil
}

// UpdateFinalizedCheckpoint does nothing after the fork.
func (f *ForkChoice) UpdateFinalizedCheckpoint(cp *forkchoicetypes.Checkpoint) error {
	if f.store.Load() == nil {
		return f.legacy.UpdateFinalizedCheckpoint(cp)
	}
	return nil
}

// SetGenesisTime does nothing after the fork, the store has its genesis.
func (f *ForkChoice) SetGenesisTime(t time.Time) {
	if f.store.Load() == nil {
		f.legacy.SetGenesisTime(t)
	}
}

// NewSlot ticks the store to now after the fork.
func (f *ForkChoice) NewSlot(ctx context.Context, slot primitives.Slot) error {
	st := f.store.Load()
	if st == nil {
		return f.legacy.NewSlot(ctx, slot)
	}
	st.OnTick(prysmTime.Now())
	return nil
}

// SetBalancesByRooter sets the old tree's balance lookup.
func (f *ForkChoice) SetBalancesByRooter(handler forkchoice.BalancesByRooter) {
	f.legacy.SetBalancesByRooter(handler)
}

// InsertSlashedIndex does nothing after the fork.
func (f *ForkChoice) InsertSlashedIndex(ctx context.Context, index primitives.ValidatorIndex) {
	if f.store.Load() == nil {
		f.legacy.InsertSlashedIndex(ctx, index)
	}
}

// RecordBlockForEquivocation does nothing after the fork.
func (f *ForkChoice) RecordBlockForEquivocation(slot primitives.Slot, proposer primitives.ValidatorIndex, root [32]byte) {
	if f.store.Load() == nil {
		f.legacy.RecordBlockForEquivocation(slot, proposer, root)
	}
}

// SetPTCVote does nothing after the fork.
func (f *ForkChoice) SetPTCVote(root [32]byte, ptcIdx uint64, payloadPresent, blobDataAvailable bool) {
	if f.store.Load() == nil {
		f.legacy.SetPTCVote(root, ptcIdx, payloadPresent, blobDataAvailable)
	}
	// TODO(DC): PTC votes.
}
