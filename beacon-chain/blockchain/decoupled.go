package blockchain

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/feed"
	statefeed "github.com/OffchainLabs/prysm/v7/beacon-chain/core/feed/state"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/core/transition"
	"github.com/OffchainLabs/prysm/v7/beacon-chain/forkchoice/decoupled"
	forkchoicetypes "github.com/OffchainLabs/prysm/v7/beacon-chain/forkchoice/types"
	"github.com/OffchainLabs/prysm/v7/config/params"
	consensusblocks "github.com/OffchainLabs/prysm/v7/consensus-types/blocks"
	"github.com/OffchainLabs/prysm/v7/consensus-types/interfaces"
	payloadattribute "github.com/OffchainLabs/prysm/v7/consensus-types/payload-attribute"
	"github.com/OffchainLabs/prysm/v7/consensus-types/primitives"
	"github.com/OffchainLabs/prysm/v7/encoding/bytesutil"
	enginev1 "github.com/OffchainLabs/prysm/v7/proto/engine/v1"
	ethpb "github.com/OffchainLabs/prysm/v7/proto/prysm/v1alpha1"
	"github.com/OffchainLabs/prysm/v7/runtime/version"
	prysmTime "github.com/OffchainLabs/prysm/v7/time"
	"github.com/OffchainLabs/prysm/v7/time/slots"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

var errDecoupledUnjoined = errors.New("decoupled fork choice has not joined")

// decoupledProposal caches the parent and vote lists selected together for a slot.
type decoupledProposal struct {
	slot           primitives.Slot
	parent         [32]byte
	votes, support []decoupled.Vote
	ok             bool
}

type decoupledFCU struct {
	head, safe, finalized [32]byte
	attr                  payloadattribute.Attributer
}

// DecoupledReceiver accepts gossip AC votes.
type DecoupledReceiver interface {
	ReceiveAvailableAttestation(context.Context, *ethpb.AvailableAttestationMessage) error
}

// logDecoupledFG logs height, justification and finalization changes reported
// by the block's state relative to its parent. These may differ from the
// checkpoints accepted by fork choice.
func logDecoupledFG(parent, n *decoupled.Node) {
	if parent == nil || n == nil {
		return
	}
	if n.FG.Target.Height > parent.FG.Target.Height {
		reason := "progress quorum"
		if n.FG.Justified.Height > parent.FG.Justified.Height {
			reason = "target quorum"
		}
		log.WithFields(logrus.Fields{"slot": n.Slot, "height": n.FG.Target.Height, "entry": rootRef(n.FG.Target.Root), "reason": reason}).Info("Decoupled height advanced")
	}
	if n.FG.Justified.Height > parent.FG.Justified.Height {
		log.WithFields(logrus.Fields{"slot": n.Slot, "height": n.FG.Justified.Height, "root": rootRef(n.FG.Justified.Root)}).Info("Decoupled height justified")
	}
	if n.FG.Finalized.Height > parent.FG.Finalized.Height {
		log.WithFields(logrus.Fields{"slot": n.Slot, "height": n.FG.Finalized.Height, "root": rootRef(n.FG.Finalized.Root)}).Info("Decoupled height finalized")
	}
}

func rootRef(root [32]byte) string { return fmt.Sprintf("%#x", bytesutil.Trunc(root[:])) }

// decoupledFinalityData builds the validator's FG inputs: height and entry
// from the source, justification and finalized height from the head.
// It returns nil if either block is missing.
func decoupledFinalityData(source, head *decoupled.Node) *ethpb.FinalityAttestationData {
	if source == nil || head == nil {
		return nil
	}
	return &ethpb.FinalityAttestationData{
		Height:          source.FG.Target.Height,
		Entry:           bytesutil.SafeCopyBytes(source.FG.Target.Root[:]),
		JustifiedHeight: head.FG.Justified.Height,
		Justified:       bytesutil.SafeCopyBytes(head.FG.Justified.Root[:]),
		FinalizedHeight: head.FG.Finalized.Height,
	}
}

// goldilocks returns nil until fork choice switches to Goldilocks. It needs no lock.
func (s *Service) goldilocks() *decoupled.Store {
	if fc, ok := s.cfg.ForkChoiceStore.(*decoupled.ForkChoice); ok {
		return fc.Decoupled()
	}
	return nil
}

// onDecoupledSyncComplete marks initial sync complete and restarts observation for
// an existing store. Voting and proposing must still pass the joined check.
func (s *Service) onDecoupledSyncComplete() {
	s.cfg.ForkChoiceStore.Lock()
	defer s.cfg.ForkChoiceStore.Unlock()
	s.decoupledSynced = true
	st := s.goldilocks()
	if st == nil {
		return
	}
	st.Restart(prysmTime.Now())
	s.decoupledProposal = decoupledProposal{}
	log.WithField("slot", s.CurrentSlot()).Info("Restarted the decoupled fork choice at sync")
}

// ReceiveAvailableAttestation records a verified AC vote at the current time
// and saves its signature for block inclusion. The caller validates the message.
func (s *Service) ReceiveAvailableAttestation(_ context.Context, m *ethpb.AvailableAttestationMessage) error {
	if m == nil || m.Data == nil {
		return errors.New("nil available attestation")
	}
	root := bytesutil.ToBytes32(m.Data.BeaconBlockRoot)
	// TODO(DC): Queue votes with their receipt times instead of taking the fork choice write lock per vote.
	s.cfg.ForkChoiceStore.Lock()
	if st := s.goldilocks(); st != nil {
		st.ProcessVote([]primitives.ValidatorIndex{m.ValidatorIndex}, root, m.Data.Slot, prysmTime.Now())
	}
	s.cfg.ForkChoiceStore.Unlock()
	if s.cfg.DecoupledVotePool != nil {
		s.cfg.DecoupledVotePool.Insert(m.ValidatorIndex, m.Data.Slot, root, m.Signature)
	}
	return nil
}

// DecoupledVote returns the head captured at Δ for the requested slot.
// It returns false until sync and the observation wait are complete, or when
// no valid vote head is retained for that slot.
func (s *Service) DecoupledVote(slot primitives.Slot) ([32]byte, bool) {
	s.cfg.ForkChoiceStore.Lock()
	defer s.cfg.ForkChoiceStore.Unlock()
	st := s.goldilocks()
	if st == nil || !s.decoupledSynced {
		return [32]byte{}, false
	}
	n := st.Vote(slot, prysmTime.Now())
	if n == nil {
		return [32]byte{}, false
	}
	return n.Root, true
}

// DecoupledFinalityData returns the source and head data captured for an FG
// duty in slot. It returns false if the node is not ready or no valid FG
// inputs are retained for that slot.
func (s *Service) DecoupledFinalityData(slot primitives.Slot) (*ethpb.FinalityAttestationData, bool) {
	s.cfg.ForkChoiceStore.Lock()
	defer s.cfg.ForkChoiceStore.Unlock()
	st := s.goldilocks()
	if st == nil || !s.decoupledSynced {
		return nil, false
	}
	d := decoupledFinalityData(st.FGInputs(slot, prysmTime.Now()))
	return d, d != nil
}

// prepareDecoupledBlock converts a validated block and its post-state, then
// prepares any ancestors needed for insertion. The caller must hold the
// legacy fork choice lock.
func (s *Service) prepareDecoupledBlock(ctx context.Context, st state.BeaconState, roblock consensusblocks.ROBlock) ([]decoupled.Block, error) {
	b, err := decoupledBlock(st, roblock)
	if err != nil {
		return nil, err
	}
	return s.prepareDecoupled(ctx, b.FG.Finalized, uint64(st.NumValidators()), b)
}

// Spec: ac_on_block after fg_on_block. Everything fallible happens here, the lookups for the missing ancestors, with no engine mutation beyond creating it, so an import that fails afterwards leaves the engine as it was. claimed is the finalized pair of the first block's post-state. The caller holds the legacy fork choice lock.
func (s *Service) prepareDecoupled(ctx context.Context, claimed decoupled.Pair, validators uint64, blocks ...decoupled.Block) ([]decoupled.Block, error) {
	s.decoupledLock.Lock()
	defer s.decoupledLock.Unlock()
	if s.decoupledStore == nil {
		if err := s.startDecoupled(ctx, stateFinalized, validators, prysmTime.Now()); err != nil {
			return nil, err
		}
	}
	prepared := make(map[[32]byte]bool, len(blocks))
	var out []decoupled.Block
	for _, b := range blocks {
		ancestors, err := s.missingDecoupledAncestors(ctx, b.Parent, prepared)
		if err != nil {
			return nil, err
		}
		for _, a := range ancestors {
			prepared[a.Root] = true
		}
		prepared[b.Root] = true
		out = append(append(out, ancestors...), b)
	}
	return out, nil
}

// commitDecoupled inserts prepared blocks and copies the accepted checkpoints
// to the legacy store. The caller must first insert the blocks into the legacy
// store and hold its lock. Cancellation is disabled so pruning finishes once
// insertion begins; an insertion or checkpoint error is fatal.
func (s *Service) commitDecoupled(ctx context.Context, blocks []decoupled.Block) {
	ctx = context.WithoutCancel(ctx)
	s.decoupledLock.Lock()
	defer s.decoupledLock.Unlock()
	now := prysmTime.Now()
	for _, b := range blocks {
		if err := s.decoupledStore.InsertNode(b, now); err != nil {
			log.WithError(err).WithField("root", fmt.Sprintf("%#x", b.Root)).Fatal("Decoupled fork choice rejected a prepared block")
		}
		if s.inRegularSync() {
			logDecoupledFG(s.decoupledStore.Node(b.Parent), b)
		}
	}
	if err := s.syncDecoupledCheckpoints(ctx); err != nil {
		log.WithError(err).Fatal("Legacy fork choice could not follow the decoupled checkpoints")
	}
}

// poolDecoupledAttestations saves included FG attestations for reuse on other
// branches. It then prunes attestations below the accepted finalized height.
// The caller holds the fork choice lock.
func (s *Service) poolDecoupledAttestations(blocks ...consensusblocks.ROBlock) {
	pool := s.cfg.DecoupledFGPool
	st := s.goldilocks()
	if pool == nil || st == nil {
		return
	}
	for _, b := range blocks {
		if atts, err := b.Block().Body().AttestationsDecoupled(); err == nil {
			pool.InsertAggregates(atts)
		}
	}
	pool.Prune(st.FinalizedHeight())
}

// insertChain switches to Goldilocks before the first Decoupled block, zero validators means read them from its saved state.
func (s *Service) insertChain(ctx context.Context, chain []*forkchoicetypes.BlockAndCheckpoints, validators uint64) error {
	first := -1
	for i, c := range chain {
		if c.Block.Version() < version.Decoupled {
			continue
		}
		if first < 0 {
			first = i
		}
		if c.FG != nil {
			continue
		}
		st, err := s.cfg.StateGen.StateByRoot(ctx, c.Block.Root())
		if err != nil {
			return errors.Wrapf(err, "could not get state for block %#x", c.Block.Root())
		}
		fg, err := decoupled.Summary(st, c.Block.Root())
		if err != nil {
			return err
		}
		c.FG = &fg
		if i == first && validators == 0 {
			validators = uint64(st.NumValidators())
		}
	}
	if first >= 0 {
		if err := s.cfg.ForkChoiceStore.InsertChain(ctx, chain[:first]); err != nil {
			return err
		}
		if err := s.startDecoupled(ctx, chain[first].Block.Block().ParentRoot(), chain[first].FG.Finalized.Root, validators, prysmTime.Now()); err != nil {
			return err
		}
		chain = chain[first:]
	}
	return s.cfg.ForkChoiceStore.InsertChain(ctx, chain)
}

// startDecoupled switches fork choice to Goldilocks with tip as the head, anchored at the later of the old tree's F and finalized.
// The caller holds the fork choice write lock.
//
// TODO(DC): Update the committee size when the validator registry grows.
// TODO(DC): Sampled committees need a denominator per slot and must account for
// validators selected for multiple seats.
func (s *Service) startDecoupled(ctx context.Context, tip, finalized [32]byte, validators uint64, now time.Time) error {
	fc, ok := s.cfg.ForkChoiceStore.(*decoupled.ForkChoice)
	if !ok || fc.Decoupled() != nil {
		return nil
	}
	if params.BeaconConfig().DecoupledSlotsPerRound < 2 {
		return errors.New("decoupled slots per round must be at least 2")
	}
	var chain []*forkchoicetypes.BlockAndCheckpoints
	have := make(map[[32]byte]bool)
	add := func(root [32]byte, blk interfaces.ReadOnlySignedBeaconBlock) error {
		ro, err := consensusblocks.NewROBlockWithRoot(blk, root)
		if err != nil {
			return err
		}
		chain = append(chain, &forkchoicetypes.BlockAndCheckpoints{Block: ro, HasPayload: fc.HasFullNode(root)})
		have[root] = true
		return nil
	}
	fRoot := s.ensureRootNotZeros(fc.FinalizedCheckpoint().Root)
	var fBlk interfaces.ReadOnlySignedBeaconBlock
	for root := tip; fBlk == nil; {
		blk, err := s.getBlock(ctx, root)
		if err != nil {
			return errors.Wrapf(err, "could not get block %#x", root)
		}
		if err := add(root, blk); err != nil {
			return err
		}
		if root == fRoot || root == finalized {
			fBlk = blk
		} else if !fc.HasNode(root) {
			return errors.Errorf("block %#x does not descend from the finalized block", root)
		}
		root = blk.Block().ParentRoot()
	}
	// Other branches stay, a child of one of them can still arrive and its parent would be missing.
	tips, _ := fc.Tips()
	for _, r := range tips {
		var roots [][32]byte
		var blks []interfaces.ReadOnlySignedBeaconBlock
		for root := r; !have[root]; root = blks[len(blks)-1].Block().ParentRoot() {
			blk, err := s.getBlock(ctx, root)
			if err != nil {
				return errors.Wrapf(err, "could not get block %#x", root)
			}
			if blk.Block().Slot() <= fBlk.Block().Slot() {
				roots = nil
				break
			}
			roots, blks = append(roots, root), append(blks, blk)
		}
		for i := range roots {
			if err := add(roots[i], blks[i]); err != nil {
				return err
			}
		}
	}
	slices.SortFunc(chain, func(a, b *forkchoicetypes.BlockAndCheckpoints) int {
		return cmp.Compare(a.Block.Block().Slot(), b.Block.Block().Slot())
	})
	f := chain[0].Block
	// A pre-fork F matches the upgraded state: height 1, entry, J and F all F, J and F at height 0.
	anchor := decoupled.Block{Root: f.Root(), Slot: f.Block().Slot(), FG: decoupled.FGSummary{
		Target:    decoupled.Pair{Height: 1, Root: f.Root()},
		Justified: decoupled.Pair{Root: f.Root()},
		Finalized: decoupled.Pair{Root: f.Root()},
	}}
	if f.Version() >= version.Decoupled {
		fst, err := s.cfg.StateGen.StateByRoot(ctx, f.Root())
		if err != nil {
			return errors.Wrapf(err, "could not get finalized state %#x", f.Root())
		}
		if anchor.FG, err = decoupled.Summary(fst, f.Root()); err != nil {
			return err
		}
		// F is finalized as the entry of its own height.
		anchor.FG.Finalized = decoupled.Pair{Height: anchor.FG.Target.Height, Root: f.Root()}
	}
	// TODO(DC): Persist locks, their original deadlines and latest_confirmed.
	// Restore them as their blocks are loaded after restart.
	// TODO(DC): Persist FG attestations received since the last block.
	// F's registry can predate validators added before the switch, which would lower every threshold.
	store, err := decoupled.New(validators, s.genesisTime, anchor, now)
	if err != nil {
		return err
	}
	if err := fc.Activate(store, tip, chain); err != nil {
		return err
	}
	log.WithFields(logrus.Fields{"tip": rootRef(tip), "finalized": rootRef(f.Root()), "blocks": len(chain)}).Info("Started the decoupled fork choice")
	return nil
}

// The first decoupled proposal needs an engine before any decoupled block exists, so the last pre-fork slot's 2Δ starts one from the legacy head with the canonical pre-fork chain as its nodes. AC messages begin at the fork, so a node synced by then has observed all of them since that slot's start and joins at the fork, a later activation observes a full slot first.
func (s *Service) activateDecoupled(ctx context.Context, slot primitives.Slot) error {
	cfg := params.BeaconConfig()
	forkSlot, err := slots.EpochStart(cfg.DecoupledForkEpoch)
	if err != nil || slot+1 < forkSlot || s.goldilocks() != nil {
		return nil
	}
	headRoot, err := s.HeadRoot(ctx)
	if err != nil {
		return err
	}
	headState, err := s.HeadState(ctx)
	if err != nil {
		return err
	}
	// The fork slot's epoch transition can still move F.
	st, err := transition.ProcessSlotsUsingNextSlotCache(ctx, headState, headRoot, forkSlot)
	if err != nil {
		return err
	}
	finalized, err := st.Finalized()
	if err != nil {
		return err
	}
	restart := prysmTime.Now()
	if slot+1 == forkSlot {
		restart = s.genesisTime.Add(time.Duration(slot) * cfg.SlotDuration())
	}
	s.cfg.ForkChoiceStore.Lock()
	defer s.cfg.ForkChoiceStore.Unlock()
	return s.startDecoupled(ctx, bytesutil.ToBytes32(headRoot), bytesutil.ToBytes32(finalized.Root), uint64(st.NumValidators()), restart)
}

// spawnDecoupledTicker starts the slot-start, Δ and 2Δ ticks after initial sync.
// It prunes the vote pool at slot start, refreshes the head at Δ and 2Δ, and
// emits confirmation and fork choice events at 2Δ. Δ is one quarter of a slot.
//
// TODO(DC): Detect machine sleep and process stalls, then restart observation
// before resuming duties.
func (s *Service) spawnDecoupledTicker() {
	go func() {
		if _, err := s.clockWaiter.WaitForClock(s.ctx); err != nil {
			log.WithError(err).Error("Failed to receive genesis data")
			return
		}
		if err := s.waitForSync(); err != nil {
			log.WithError(err).Error("Could not wait to sync")
			return
		}
		s.onDecoupledSyncComplete()
		delta := params.BeaconConfig().SlotDuration() / 4
		ticker := slots.NewSlotTickerWithIntervals(s.genesisTime, []time.Duration{0, delta, 2 * delta})
		defer ticker.Done()
		synced := true
		for {
			select {
			case <-s.ctx.Done():
				return
			case t := <-ticker.C():
				// A resync misses votes like a restart does, so observation restarts when it ends.
				regular := s.inRegularSync()
				if regular && !synced {
					s.onDecoupledSyncComplete()
				}
				synced = regular
				if t.Interval == 0 && s.cfg.DecoupledVotePool != nil {
					s.cfg.DecoupledVotePool.Prune(t.Slot)
				}
				if t.Interval == 2 {
					if err := s.activateDecoupled(s.ctx, t.Slot); err != nil {
						log.WithError(err).Error("Could not start the decoupled fork choice at the fork")
					}
				}
				if s.tickDecoupled(t.Interval) && t.Interval > 0 {
					s.UpdateHead(s.ctx, t.Slot+1)
				}
				if t.Interval == 2 {
					s.notifyFastConfirmation(t.Slot)
					s.notifyDecoupledTick()
				}
			}
		}
	}()
}

// tickDecoupled runs due phases and updates the gauges at the 2Δ tick.
// It returns false before the switch to Goldilocks.
func (s *Service) tickDecoupled(interval int) bool {
	s.cfg.ForkChoiceStore.Lock()
	defer s.cfg.ForkChoiceStore.Unlock()
	st := s.goldilocks()
	if st == nil {
		return false
	}
	now := prysmTime.Now()
	st.OnTick(now)
	if interval == 2 {
		decoupledHeadSlot.Set(float64(st.Head(now).Slot))
		decoupledConfirmedSlot.Set(float64(st.Confirmed(now).Slot))
		decoupledJustifiedHeight.Set(float64(st.JustifiedHeight()))
		decoupledFinalizedHeight.Set(float64(st.FinalizedHeight()))
	}
	return true
}

// DecoupledConfirmed returns the confirmed root published with the head.
// It is false before the switch to Goldilocks.
func (s *Service) DecoupledConfirmed() ([32]byte, bool) {
	s.cfg.ForkChoiceStore.RLock()
	defer s.cfg.ForkChoiceStore.RUnlock()
	return s.decoupledConfirmed()
}

func (s *Service) decoupledConfirmed() ([32]byte, bool) {
	fc, ok := s.cfg.ForkChoiceStore.(*decoupled.ForkChoice)
	if !ok || fc.Decoupled() == nil {
		return [32]byte{}, false
	}
	return fc.PublishedConfirmed(), true
}

// notifyDecoupledTick publishes a snapshot of the fork choice state for
// subscribers, including accepted checkpoints and current locks.
func (s *Service) notifyDecoupledTick() {
	d, ok := s.DecoupledForkChoiceDump()
	if !ok {
		return
	}
	s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.DecoupledTick,
		Data: &statefeed.DecoupledTickData{
			Slot:                d.Slot,
			Joined:              d.Joined,
			HeadRoot:            d.Head,
			ConfirmedRoot:       d.Confirmed,
			LatestConfirmedRoot: d.LatestConfirmed,
			JustifiedRoot:       d.Justified,
			FinalizedRoot:       d.Finalized,
			JustifiedSlot:       d.JustifiedSlot,
			FinalizedSlot:       d.FinalizedSlot,
			JustifiedHeight:     d.JustifiedHeight,
			FinalizedHeight:     d.FinalizedHeight,
			MaxHeight:           d.MaxHeight,
			CommitteeSize:       d.CommitteeSize,
			Locks:               d.Locks,
		},
	})
}

// DecoupledForkChoiceDump returns a snapshot after processing due phases.
// It returns false before the switch to Goldilocks.
func (s *Service) DecoupledForkChoiceDump() (*decoupled.Dump, bool) {
	s.cfg.ForkChoiceStore.Lock()
	defer s.cfg.ForkChoiceStore.Unlock()
	st := s.goldilocks()
	if st == nil {
		return nil, false
	}
	d := st.Dump(prysmTime.Now())
	return &d, true
}

// DecoupledProposal selects and caches the parent and AC vote lists together
// for slot. It returns false until sync and the observation wait are complete.
// Callers must treat the returned vote slices as read-only.
func (s *Service) DecoupledProposal(slot primitives.Slot) (parent [32]byte, votes, support []decoupled.Vote, ok bool) {
	s.cfg.ForkChoiceStore.Lock()
	defer s.cfg.ForkChoiceStore.Unlock()
	st := s.goldilocks()
	if st == nil || !s.decoupledSynced {
		return
	}
	// An invalid payload can remove the cached parent.
	if m := s.decoupledProposal; m.ok && m.slot == slot && st.HasNode(m.parent) {
		return m.parent, m.votes, m.support, true
	}
	p, v, sp := st.Proposal(slot, prysmTime.Now())
	if p == nil {
		return
	}
	s.decoupledProposal = decoupledProposal{slot: slot, parent: p.Root, votes: v, support: sp, ok: true}
	return p.Root, v, sp, true
}

// ProposerParent returns the parent root for a proposal. It uses legacy fork
// choice until Goldilocks is initialized, then requires a Decoupled proposal.
// It returns an error if the Decoupled proposer is not ready.
func (s *Service) ProposerParent(slot primitives.Slot) ([32]byte, error) {
	if s.goldilocks() == nil {
		return s.GetProposerHead(), nil
	}
	parent, _, _, ok := s.DecoupledProposal(slot)
	if !ok {
		log.WithField("slot", slot).Warn("Skipped the proposal, the decoupled fork choice has not joined")
		return [32]byte{}, errDecoupledUnjoined
	}
	return parent, nil
}

// publishSyncHead keeps fork choice's published head on imported blocks while sync runs without votes.
func (s *Service) publishSyncHead(head [32]byte) {
	if fc, ok := s.cfg.ForkChoiceStore.(*decoupled.ForkChoice); ok && fc.Decoupled() != nil {
		fc.SetPublished(head, fc.Decoupled().Finalized().Root)
	}
}

// publishDecoupledHead saves the head, publishes it with the confirmed block and updates the execution client in regular sync.
// The caller holds the fork choice write lock.
func (s *Service) publishDecoupledHead(ctx context.Context, proposingSlot primitives.Slot) ([32]byte, bool) {
	fc, ok := s.cfg.ForkChoiceStore.(*decoupled.ForkChoice)
	if !ok || fc.Decoupled() == nil {
		return [32]byte{}, false
	}
	st, now := fc.Decoupled(), prysmTime.Now()
	head, confirmed := st.Head(now).Root, st.Confirmed(now).Root
	// Without a full slot of fresh votes the head stays at its anchor, which would stall initial sync.
	if !s.inRegularSync() || !st.Joined(s.CurrentSlot()) {
		head = fc.HighestReceivedBlockRoot()
	}
	// TODO(DC): full or empty from PTC and payload arrival.
	full := true
	headState, headBlock, err := s.getStateAndBlock(ctx, head)
	if err != nil {
		log.WithError(err).WithField("root", fmt.Sprintf("%#x", head)).Error("Could not load the decoupled head")
		return head, false
	}
	if err := s.saveHead(ctx, head, headBlock, headState, full); err != nil {
		log.WithError(err).Error("Could not save head")
	}
	// saveHead can fail or return early without installing the head, and readers must agree with HeadRoot.
	s.headLock.RLock()
	installed := s.headRoot()
	s.headLock.RUnlock()
	if installed != head {
		return head, false
	}
	fc.SetPublished(head, confirmed)
	hash, err := fc.BlockHash(head)
	if err != nil || hash == [32]byte{} || !s.inRegularSync() {
		return head, true
	}
	attr := s.getPayloadAttribute(ctx, headState, proposingSlot, head[:], full)
	safe := fc.ConfirmedPayloadBlockHash(confirmed)
	final := fc.FinalizedPayloadBlockHash()
	if s.decoupledFCUSent(hash, safe, final, attr) {
		return head, true
	}
	fcs := &enginev1.ForkchoiceState{HeadBlockHash: hash[:], SafeBlockHash: safe[:], FinalizedBlockHash: final[:]}
	go s.fcuDecoupled(headBlock, head, fcs, full, attr, proposingSlot)
	return head, true
}

// decoupledFCUSent reports whether the hashes and all payload attributes match
// the last successful execution fork choice update. Failed or SYNCING responses
// are not recorded, so they can be retried.
func (s *Service) decoupledFCUSent(head, safe, finalized [32]byte, attr payloadattribute.Attributer) bool {
	s.decoupledFCULock.Lock()
	defer s.decoupledFCULock.Unlock()
	last := s.decoupledFCU
	return last.head == head && last.safe == safe && last.finalized == finalized && sameAttributes(last.attr, attr)
}

// fcuDecoupled sends the execution fork choice update and caches any payload ID.
// An INVALID response removes the invalid branch and triggers head selection
// again. Only successful responses other than SYNCING are saved for deduplication.
func (s *Service) fcuDecoupled(headBlock interfaces.ReadOnlySignedBeaconBlock, hr [32]byte, fcs *enginev1.ForkchoiceState, full bool, attr payloadattribute.Attributer, proposingSlot primitives.Slot) {
	pid, syncing, err := s.forkchoiceUpdatedGloas(s.ctx, fcs, attr)
	if err != nil {
		log.WithError(err).Error("Could not update forkchoice with engine")
		if !IsInvalidBlock(err) {
			return
		}
		s.cfg.ForkChoiceStore.Lock()
		defer s.cfg.ForkChoiceStore.Unlock()
		parentHash := s.cfg.ForkChoiceStore.ParentHash(hr)
		if err := s.pruneInvalidBlock(s.ctx, hr, headBlock.Block().ParentRoot(), parentHash, InvalidBlockLVH(err)); err != nil && !IsInvalidBlock(err) {
			log.WithError(err).Error("Could not prune the invalid decoupled head")
			return
		}
		s.publishDecoupledHead(s.ctx, proposingSlot)
		return
	}
	if !syncing {
		s.recordDecoupledFCU(fcs, attr)
	}
	if pid == nil {
		if !attr.IsEmpty() {
			log.Warn("Engine did not return a payload ID for the fork choice update with attributes")
		}
		return
	}
	var pId [8]byte
	copy(pId[:], pid[:])
	s.cfg.PayloadIDCache.Set(proposingSlot, hr, full, pId)
	if !attr.IsEmpty() {
		s.firePayloadAttributesEvent(s.cfg.StateNotifier.StateFeed(), headBlock, hr, proposingSlot, attr, fcs.HeadBlockHash)
	}
}

// recordDecoupledFCU saves the last successful execution fork choice update.
// The caller must exclude failed and SYNCING responses.
func (s *Service) recordDecoupledFCU(fcs *enginev1.ForkchoiceState, attr payloadattribute.Attributer) {
	s.decoupledFCULock.Lock()
	defer s.decoupledFCULock.Unlock()
	s.decoupledFCU = decoupledFCU{head: bytesutil.ToBytes32(fcs.HeadBlockHash), safe: bytesutil.ToBytes32(fcs.SafeBlockHash), finalized: bytesutil.ToBytes32(fcs.FinalizedBlockHash), attr: attr}
}

// sameAttributes compares all fields of payload attributes of the same version.
// Nil and empty attributes are equivalent.
func sameAttributes(a, b payloadattribute.Attributer) bool {
	if a == nil || a.IsEmpty() {
		return b == nil || b.IsEmpty()
	}
	if b == nil || b.IsEmpty() || a.Version() != b.Version() {
		return false
	}
	return proto.Equal(attributesProto(a), attributesProto(b))
}

// attributesProto returns the newest protobuf representation supported by a.
func attributesProto(a payloadattribute.Attributer) proto.Message {
	if m, err := a.PbV4(); err == nil {
		return m
	}
	if m, err := a.PbV3(); err == nil {
		return m
	}
	if m, err := a.PbV2(); err == nil {
		return m
	}
	m, _ := a.PbV1()
	return m
}

// persistDecoupledCheckpoints persists the accepted checkpoints and updates the
// finalized index. Root comparisons preserve advances within an epoch. The caller
// must hold the fork choice lock.
func (s *Service) persistDecoupledCheckpoints(ctx context.Context) error {
	j, f := s.cfg.ForkChoiceStore.JustifiedCheckpoint(), s.cfg.ForkChoiceStore.FinalizedCheckpoint()
	saved, err := s.cfg.BeaconDB.JustifiedCheckpoint(ctx)
	if err != nil {
		return err
	}
	if !bytes.Equal(saved.Root, j.Root[:]) {
		if err := s.cfg.BeaconDB.SaveJustifiedCheckpoint(ctx, &ethpb.Checkpoint{Epoch: j.Epoch, Root: j.Root[:]}); err != nil {
			return err
		}
	}
	return s.updateFinalized(ctx, &ethpb.Checkpoint{Epoch: f.Epoch, Root: f.Root[:]})
}
