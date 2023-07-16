package blockchain

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	coreTime "github.com/prysmaticlabs/prysm/v4/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/config/features"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	consensusblocks "github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1/attestation"
	"github.com/prysmaticlabs/prysm/v4/runtime/version"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// A custom slot deadline for processing state slots in our cache.
const slotDeadline = 5 * time.Second

// A custom deadline for deposit trie insertion.
const depositDeadline = 20 * time.Second

// This defines size of the upper bound for initial sync block cache.
var initialSyncBlockCacheSize = uint64(2 * params.BeaconConfig().SlotsPerEpoch)

// postBlockProcess is called when a gossip block is received. This function performs
// several duties most importantly informing the engine if head was updated,
// saving the new head information to the blockchain package and
// handling attestations, slashings and similar included in the block.
func (s *Service) postBlockProcess(ctx context.Context, signed interfaces.ReadOnlySignedBeaconBlock, blockRoot [32]byte, postState state.BeaconState, isValidPayload bool) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.onBlock")
	defer span.End()
	if err := consensusblocks.BeaconBlockIsNil(signed); err != nil {
		return invalidBlock{error: err}
	}
	startTime := time.Now()
	b := signed.Block()

	if err := s.cfg.ForkChoiceStore.InsertNode(ctx, postState, blockRoot); err != nil {
		return errors.Wrapf(err, "could not insert block %d to fork choice store", signed.Block().Slot())
	}
	if err := s.handleBlockAttestations(ctx, signed.Block(), postState); err != nil {
		return errors.Wrap(err, "could not handle block's attestations")
	}

	s.InsertSlashingsToForkChoiceStore(ctx, signed.Block().Body().AttesterSlashings())
	if isValidPayload {
		if err := s.cfg.ForkChoiceStore.SetOptimisticToValid(ctx, blockRoot); err != nil {
			return errors.Wrap(err, "could not set optimistic block to valid")
		}
	}

	start := time.Now()
	headRoot, err := s.cfg.ForkChoiceStore.Head(ctx)
	if err != nil {
		log.WithError(err).Warn("Could not update head")
	}
	if blockRoot != headRoot {
		receivedWeight, err := s.cfg.ForkChoiceStore.Weight(blockRoot)
		if err != nil {
			log.WithField("root", fmt.Sprintf("%#x", blockRoot)).Warn("could not determine node weight")
		}
		headWeight, err := s.cfg.ForkChoiceStore.Weight(headRoot)
		if err != nil {
			log.WithField("root", fmt.Sprintf("%#x", headRoot)).Warn("could not determine node weight")
		}
		log.WithFields(logrus.Fields{
			"receivedRoot":   fmt.Sprintf("%#x", blockRoot),
			"receivedWeight": receivedWeight,
			"headRoot":       fmt.Sprintf("%#x", headRoot),
			"headWeight":     headWeight,
		}).Debug("Head block is not the received block")
	} else {
		// Updating next slot state cache can happen in the background. It shouldn't block rest of the process.
		go func() {
			// Use a custom deadline here, since this method runs asynchronously.
			// We ignore the parent method's context and instead create a new one
			// with a custom deadline, therefore using the background context instead.
			slotCtx, cancel := context.WithTimeout(context.Background(), slotDeadline)
			defer cancel()
			if err := transition.UpdateNextSlotCache(slotCtx, blockRoot[:], postState); err != nil {
				log.WithError(err).Debug("could not update next slot state cache")
			}
		}()
	}
	newBlockHeadElapsedTime.Observe(float64(time.Since(start).Milliseconds()))

	// verify conditions for FCU, notifies FCU, and saves the new head.
	// This function also prunes attestations, other similar operations happen in prunePostBlockOperationPools.
	if _, err := s.forkchoiceUpdateWithExecution(ctx, headRoot, s.CurrentSlot()+1); err != nil {
		return err
	}

	// Send notification of the processed block to the state feed.
	s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			Slot:        signed.Block().Slot(),
			BlockRoot:   blockRoot,
			SignedBlock: signed,
			Verified:    true,
		},
	})

	defer reportAttestationInclusion(b)
	if err := s.handleEpochBoundary(ctx, postState, blockRoot[:]); err != nil {
		return err
	}
	onBlockProcessingTime.Observe(float64(time.Since(startTime).Milliseconds()))
	return nil
}

func getStateVersionAndPayload(st state.BeaconState) (int, interfaces.ExecutionData, error) {
	if st == nil {
		return 0, nil, errors.New("nil state")
	}
	var preStateHeader interfaces.ExecutionData
	var err error
	preStateVersion := st.Version()
	switch preStateVersion {
	case version.Phase0, version.Altair:
	default:
		preStateHeader, err = st.LatestExecutionPayloadHeader()
		if err != nil {
			return 0, nil, err
		}
	}
	return preStateVersion, preStateHeader, nil
}

func (s *Service) onBlockBatch(ctx context.Context, blks []interfaces.ReadOnlySignedBeaconBlock,
	blockRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.onBlockBatch")
	defer span.End()

	if len(blks) == 0 || len(blockRoots) == 0 {
		return errors.New("no blocks provided")
	}

	if len(blks) != len(blockRoots) {
		return errWrongBlockCount
	}

	if err := consensusblocks.BeaconBlockIsNil(blks[0]); err != nil {
		return invalidBlock{error: err}
	}
	b := blks[0].Block()

	// Retrieve incoming block's pre state.
	if err := s.verifyBlkPreState(ctx, b); err != nil {
		return err
	}
	preState, err := s.cfg.StateGen.StateByRootInitialSync(ctx, b.ParentRoot())
	if err != nil {
		return err
	}
	if preState == nil || preState.IsNil() {
		return fmt.Errorf("nil pre state for slot %d", b.Slot())
	}

	// Fill in missing blocks
	if err := s.fillInForkChoiceMissingBlocks(ctx, blks[0].Block(), preState.CurrentJustifiedCheckpoint(), preState.FinalizedCheckpoint()); err != nil {
		return errors.Wrap(err, "could not fill in missing blocks to forkchoice")
	}

	jCheckpoints := make([]*ethpb.Checkpoint, len(blks))
	fCheckpoints := make([]*ethpb.Checkpoint, len(blks))
	sigSet := bls.NewSet()
	type versionAndHeader struct {
		version int
		header  interfaces.ExecutionData
	}
	preVersionAndHeaders := make([]*versionAndHeader, len(blks))
	postVersionAndHeaders := make([]*versionAndHeader, len(blks))
	var set *bls.SignatureBatch
	boundaries := make(map[[32]byte]state.BeaconState)
	for i, b := range blks {
		v, h, err := getStateVersionAndPayload(preState)
		if err != nil {
			return err
		}
		preVersionAndHeaders[i] = &versionAndHeader{
			version: v,
			header:  h,
		}

		set, preState, err = transition.ExecuteStateTransitionNoVerifyAnySig(ctx, preState, b)
		if err != nil {
			return invalidBlock{error: err}
		}
		// Save potential boundary states.
		if slots.IsEpochStart(preState.Slot()) {
			boundaries[blockRoots[i]] = preState.Copy()
		}
		jCheckpoints[i] = preState.CurrentJustifiedCheckpoint()
		fCheckpoints[i] = preState.FinalizedCheckpoint()

		v, h, err = getStateVersionAndPayload(preState)
		if err != nil {
			return err
		}
		postVersionAndHeaders[i] = &versionAndHeader{
			version: v,
			header:  h,
		}
		sigSet.Join(set)
	}

	var verify bool
	if features.Get().EnableVerboseSigVerification {
		verify, err = sigSet.VerifyVerbosely()
	} else {
		verify, err = sigSet.Verify()
	}
	if err != nil {
		return invalidBlock{error: err}
	}
	if !verify {
		return errors.New("batch block signature verification failed")
	}

	// blocks have been verified, save them and call the engine
	pendingNodes := make([]*forkchoicetypes.BlockAndCheckpoints, len(blks))
	var isValidPayload bool
	for i, b := range blks {
		isValidPayload, err = s.notifyNewPayload(ctx,
			postVersionAndHeaders[i].version,
			postVersionAndHeaders[i].header, b)
		if err != nil {
			return s.handleInvalidExecutionError(ctx, err, blockRoots[i], b.Block().ParentRoot())
		}
		if isValidPayload {
			if err := s.validateMergeTransitionBlock(ctx, preVersionAndHeaders[i].version,
				preVersionAndHeaders[i].header, b); err != nil {
				return err
			}
		}
		args := &forkchoicetypes.BlockAndCheckpoints{Block: b.Block(),
			JustifiedCheckpoint: jCheckpoints[i],
			FinalizedCheckpoint: fCheckpoints[i]}
		pendingNodes[len(blks)-i-1] = args
		if err := s.saveInitSyncBlock(ctx, blockRoots[i], b); err != nil {
			tracing.AnnotateError(span, err)
			return err
		}
		if err := s.cfg.BeaconDB.SaveStateSummary(ctx, &ethpb.StateSummary{
			Slot: b.Block().Slot(),
			Root: blockRoots[i][:],
		}); err != nil {
			tracing.AnnotateError(span, err)
			return err
		}
		if i > 0 && jCheckpoints[i].Epoch > jCheckpoints[i-1].Epoch {
			if err := s.cfg.BeaconDB.SaveJustifiedCheckpoint(ctx, jCheckpoints[i]); err != nil {
				tracing.AnnotateError(span, err)
				return err
			}
		}
		if i > 0 && fCheckpoints[i].Epoch > fCheckpoints[i-1].Epoch {
			if err := s.updateFinalized(ctx, fCheckpoints[i]); err != nil {
				tracing.AnnotateError(span, err)
				return err
			}
		}
	}
	// Save boundary states that will be useful for forkchoice
	for r, st := range boundaries {
		if err := s.cfg.StateGen.SaveState(ctx, r, st); err != nil {
			return err
		}
	}
	// Also saves the last post state which to be used as pre state for the next batch.
	lastBR := blockRoots[len(blks)-1]
	if err := s.cfg.StateGen.SaveState(ctx, lastBR, preState); err != nil {
		return err
	}
	// Insert all nodes but the last one to forkchoice
	if err := s.cfg.ForkChoiceStore.InsertChain(ctx, pendingNodes); err != nil {
		return errors.Wrap(err, "could not insert batch to forkchoice")
	}
	// Insert the last block to forkchoice
	if err := s.cfg.ForkChoiceStore.InsertNode(ctx, preState, lastBR); err != nil {
		return errors.Wrap(err, "could not insert last block in batch to forkchoice")
	}
	// Set their optimistic status
	if isValidPayload {
		if err := s.cfg.ForkChoiceStore.SetOptimisticToValid(ctx, lastBR); err != nil {
			return errors.Wrap(err, "could not set optimistic block to valid")
		}
	}
	lastB := blks[len(blks)-1]
	arg := &notifyForkchoiceUpdateArg{
		headState: preState,
		headRoot:  lastBR,
		headBlock: lastB.Block(),
	}
	if _, err := s.notifyForkchoiceUpdate(ctx, arg); err != nil {
		return err
	}
	return s.saveHeadNoDB(ctx, lastB, lastBR, preState)
}

// Epoch boundary bookkeeping such as logging epoch summaries.
func (s *Service) handleEpochBoundary(ctx context.Context, postState state.BeaconState, blockRoot []byte) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.handleEpochBoundary")
	defer span.End()

	var err error
	if postState.Slot()+1 == s.nextEpochBoundarySlot {
		copied := postState.Copy()
		copied, err := transition.ProcessSlotsUsingNextSlotCache(ctx, copied, blockRoot, copied.Slot()+1)
		if err != nil {
			return err
		}
		// Update caches for the next epoch at epoch boundary slot - 1.
		if err := helpers.UpdateCommitteeCache(ctx, copied, coreTime.CurrentEpoch(copied)); err != nil {
			return err
		}
		e := coreTime.CurrentEpoch(copied)
		if err := helpers.UpdateProposerIndicesInCache(ctx, copied, e); err != nil {
			return err
		}
		go func() {
			// Use a custom deadline here, since this method runs asynchronously.
			// We ignore the parent method's context and instead create a new one
			// with a custom deadline, therefore using the background context instead.
			slotCtx, cancel := context.WithTimeout(context.Background(), slotDeadline)
			defer cancel()
			if err := helpers.UpdateProposerIndicesInCache(slotCtx, copied, e+1); err != nil {
				log.WithError(err).Warn("Failed to cache next epoch proposers")
			}
		}()
	} else if postState.Slot() >= s.nextEpochBoundarySlot {
		s.nextEpochBoundarySlot, err = slots.EpochStart(coreTime.NextEpoch(postState))
		if err != nil {
			return err
		}

		// Update caches at epoch boundary slot.
		// The following updates have shortcut to return nil cheaply if fulfilled during boundary slot - 1.
		if err := helpers.UpdateCommitteeCache(ctx, postState, coreTime.CurrentEpoch(postState)); err != nil {
			return err
		}
		if err := helpers.UpdateProposerIndicesInCache(ctx, postState, coreTime.CurrentEpoch(postState)); err != nil {
			return err
		}

		headSt, err := s.HeadState(ctx)
		if err != nil {
			return err
		}
		if err := reportEpochMetrics(ctx, postState, headSt); err != nil {
			return err
		}
	}
	return nil
}

// This feeds in the attestations included in the block to fork choice store. It's allows fork choice store
// to gain information on the most current chain.
func (s *Service) handleBlockAttestations(ctx context.Context, blk interfaces.ReadOnlyBeaconBlock, st state.BeaconState) error {
	// Feed in block's attestations to fork choice store.
	for _, a := range blk.Body().Attestations() {
		committee, err := helpers.BeaconCommitteeFromState(ctx, st, a.Data.Slot, a.Data.CommitteeIndex)
		if err != nil {
			return err
		}
		indices, err := attestation.AttestingIndices(a.AggregationBits, committee)
		if err != nil {
			return err
		}
		r := bytesutil.ToBytes32(a.Data.BeaconBlockRoot)
		if s.cfg.ForkChoiceStore.HasNode(r) {
			s.cfg.ForkChoiceStore.ProcessAttestation(ctx, indices, r, a.Data.Target.Epoch)
		} else if err := s.cfg.AttPool.SaveBlockAttestation(a); err != nil {
			return err
		}
	}
	return nil
}

// InsertSlashingsToForkChoiceStore inserts attester slashing indices to fork choice store.
// To call this function, it's caller's responsibility to ensure the slashing object is valid.
// This function requires a write lock on forkchoice.
func (s *Service) InsertSlashingsToForkChoiceStore(ctx context.Context, slashings []*ethpb.AttesterSlashing) {
	for _, slashing := range slashings {
		indices := blocks.SlashableAttesterIndices(slashing)
		for _, index := range indices {
			s.cfg.ForkChoiceStore.InsertSlashedIndex(ctx, primitives.ValidatorIndex(index))
		}
	}
}

// This saves post state info to DB or cache. This also saves post state info to fork choice store.
// Post state info consists of processed block and state. Do not call this method unless the block and state are verified.
func (s *Service) savePostStateInfo(ctx context.Context, r [32]byte, b interfaces.ReadOnlySignedBeaconBlock, st state.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "blockChain.savePostStateInfo")
	defer span.End()
	if err := s.cfg.BeaconDB.SaveBlock(ctx, b); err != nil {
		return errors.Wrapf(err, "could not save block from slot %d", b.Block().Slot())
	}
	if err := s.cfg.StateGen.SaveState(ctx, r, st); err != nil {
		return errors.Wrap(err, "could not save state")
	}
	return nil
}

// This removes the attestations in block `b` from the attestation mem pool.
func (s *Service) pruneAttsFromPool(headBlock interfaces.ReadOnlySignedBeaconBlock) error {
	atts := headBlock.Block().Body().Attestations()
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

// validateMergeTransitionBlock validates the merge transition block.
func (s *Service) validateMergeTransitionBlock(ctx context.Context, stateVersion int, stateHeader interfaces.ExecutionData, blk interfaces.ReadOnlySignedBeaconBlock) error {
	// Skip validation if block is older than Bellatrix.
	if blocks.IsPreBellatrixVersion(blk.Block().Version()) {
		return nil
	}

	// Skip validation if block has an empty payload.
	payload, err := blk.Block().Body().Execution()
	if err != nil {
		return invalidBlock{error: err}
	}
	isEmpty, err := consensusblocks.IsEmptyExecutionData(payload)
	if err != nil {
		return err
	}
	if isEmpty {
		return nil
	}

	// Handle case where pre-state is Altair but block contains payload.
	// To reach here, the block must have contained a valid payload.
	if blocks.IsPreBellatrixVersion(stateVersion) {
		return s.validateMergeBlock(ctx, blk)
	}

	// Skip validation if the block is not a merge transition block.
	// To reach here. The payload must be non-empty. If the state header is empty then it's at transition.
	empty, err := consensusblocks.IsEmptyExecutionData(stateHeader)
	if err != nil {
		return err
	}
	if !empty {
		return nil
	}
	return s.validateMergeBlock(ctx, blk)
}

// This routine checks if there is a cached proposer payload ID available for the next slot proposer.
// If there is not, it will call forkchoice updated with the correct payload attribute then cache the payload ID.
func (s *Service) runLateBlockTasks() {
	if err := s.waitForSync(); err != nil {
		log.WithError(err).Error("failed to wait for initial sync")
		return
	}

	attThreshold := params.BeaconConfig().SecondsPerSlot / 3
	ticker := slots.NewSlotTickerWithOffset(s.genesisTime, time.Duration(attThreshold)*time.Second, params.BeaconConfig().SecondsPerSlot)
	for {
		select {
		case <-ticker.C():
			s.lateBlockTasks(s.ctx)
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting routine")
			return
		}
	}
}

// lateBlockTasks  is called 4 seconds into the slot and performs tasks
// related to late blocks. It emits a MissedSlot state feed event.
// It calls FCU and sets the right attributes if we are proposing next slot
// it also updates the next slot cache to deal with skipped slots.
func (s *Service) lateBlockTasks(ctx context.Context) {
	if s.CurrentSlot() == s.HeadSlot() {
		return
	}
	s.cfg.StateNotifier.StateFeed().Send(&feed.Event{
		Type: statefeed.MissedSlot,
	})

	headRoot := s.headRoot()
	headState := s.headState(ctx)
	lastRoot, lastState := transition.LastCachedState()
	if lastState == nil {
		lastRoot, lastState = headRoot[:], headState
	}
	// Copy all the field tries in our cached state in the event of late
	// blocks.
	lastState.CopyAllTries()
	if err := transition.UpdateNextSlotCache(ctx, lastRoot, lastState); err != nil {
		log.WithError(err).Debug("could not update next slot state cache")
	}

	// Head root should be empty when retrieving proposer index for the next slot.
	_, id, has := s.cfg.ProposerSlotIndexCache.GetProposerPayloadIDs(s.CurrentSlot()+1, [32]byte{} /* head root */)
	// There exists proposer for next slot, but we haven't called fcu w/ payload attribute yet.
	if (!has && !features.Get().PrepareAllPayloads) || id != [8]byte{} {
		return
	}

	s.headLock.RLock()
	headBlock, err := s.headBlock()
	if err != nil {
		s.headLock.RUnlock()
		log.WithError(err).Debug("could not perform late block tasks: failed to retrieve head block")
		return
	}
	s.headLock.RUnlock()
	_, err = s.notifyForkchoiceUpdate(ctx, &notifyForkchoiceUpdateArg{
		headState: headState,
		headRoot:  headRoot,
		headBlock: headBlock.Block(),
	})
	if err != nil {
		log.WithError(err).Debug("could not perform late block tasks: failed to update forkchoice with engine")
	}
}

// waitForSync blocks until the node is synced to the head.
func (s *Service) waitForSync() error {
	select {
	case <-s.syncComplete:
		return nil
	case <-s.ctx.Done():
		return errors.New("context closed, exiting goroutine")
	}
}

func (s *Service) handleInvalidExecutionError(ctx context.Context, err error, blockRoot [32]byte, parentRoot [32]byte) error {
	if IsInvalidBlock(err) && InvalidBlockLVH(err) != [32]byte{} {
		return s.pruneInvalidBlock(ctx, blockRoot, parentRoot, InvalidBlockLVH(err))
	}
	return err
}
