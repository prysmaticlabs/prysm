package initialsync

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/paulbellamy/ratecounter"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/sync"
	consensus_types "github.com/prysmaticlabs/prysm/v4/consensus-types"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"github.com/sirupsen/logrus"
)

const (
	// counterSeconds is an interval over which an average rate will be calculated.
	counterSeconds = 20
)

// blockReceiverFn defines block receiving function.
type blockReceiverFn func(ctx context.Context, block interfaces.ReadOnlySignedBeaconBlock, blockRoot [32]byte) error

// batchBlockReceiverFn defines batch receiving function.
type batchBlockReceiverFn func(ctx context.Context, blks []interfaces.ReadOnlySignedBeaconBlock, roots [][32]byte) error

// Round Robin sync looks at the latest peer statuses and syncs up to the highest known epoch.
//
// Step 1 - Sync to finalized epoch.
// Sync with peers having the majority on best finalized epoch greater than node's head state.
//
// Step 2 - Sync to head from finalized epoch.
// Using enough peers (at least, MinimumSyncPeers*2, for example) obtain best non-finalized epoch,
// known to majority of the peers, and keep fetching blocks, up until that epoch is reached.
func (s *Service) roundRobinSync(genesis time.Time) error {
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()
	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	s.counter = ratecounter.NewRateCounter(counterSeconds * time.Second)

	// Step 1 - Sync to end of finalized epoch.
	if err := s.syncToFinalizedEpoch(ctx, genesis); err != nil {
		return err
	}

	// Already at head, no need for 2nd phase.
	if s.cfg.Chain.HeadSlot() == slots.Since(genesis) {
		return nil
	}

	// Step 2 - sync to head from majority of peers (from no less than MinimumSyncPeers*2 peers)
	// having the same world view on non-finalized epoch.
	return s.syncToNonFinalizedEpoch(ctx, genesis)
}

// syncToFinalizedEpoch sync from head to best known finalized epoch.
func (s *Service) syncToFinalizedEpoch(ctx context.Context, genesis time.Time) error {
	highestFinalizedSlot, err := slots.EpochStart(s.highestFinalizedEpoch() + 1)
	if err != nil {
		return err
	}
	if s.cfg.Chain.HeadSlot() >= highestFinalizedSlot {
		// No need to sync, already synced to the finalized slot.
		log.Debug("Already synced to finalized epoch")
		return nil
	}

	vr := s.clock.GenesisValidatorsRoot()
	ctxMap, err := sync.ContextByteVersionsForValRoot(vr)
	if err != nil {
		return errors.Wrapf(err, "unable to initialize context version map using genesis validator root = %#x", vr)
	}
	queue := newBlocksQueue(ctx, &blocksQueueConfig{
		p2p:                 s.cfg.P2P,
		db:                  s.cfg.DB,
		chain:               s.cfg.Chain,
		clock:               s.clock,
		ctxMap:              ctxMap,
		highestExpectedSlot: highestFinalizedSlot,
		mode:                modeStopOnFinalizedEpoch,
	})
	if err := queue.start(); err != nil {
		return err
	}

	for data := range queue.fetchedData {
		// If blobs are available. Verify blobs and blocks are consistence.
		// We can't import a block if there's no associated blob within DA bound.
		// The blob has to pass aggregated proof check.
		s.processFetchedData(ctx, genesis, s.cfg.Chain.HeadSlot(), data)
	}

	log.WithFields(logrus.Fields{
		"syncedSlot":  s.cfg.Chain.HeadSlot(),
		"currentSlot": slots.Since(genesis),
	}).Info("Synced to finalized epoch - now syncing blocks up to current head")
	if err := queue.stop(); err != nil {
		log.WithError(err).Debug("Error stopping queue")
	}

	return nil
}

// syncToNonFinalizedEpoch sync from head to best known non-finalized epoch supported by majority
// of peers (no less than MinimumSyncPeers*2 peers).
func (s *Service) syncToNonFinalizedEpoch(ctx context.Context, genesis time.Time) error {
	vr := s.clock.GenesisValidatorsRoot()
	ctxMap, err := sync.ContextByteVersionsForValRoot(vr)
	if err != nil {
		return errors.Wrapf(err, "unable to initialize context version map using genesis validator root = %#x", vr)
	}
	queue := newBlocksQueue(ctx, &blocksQueueConfig{
		p2p:                 s.cfg.P2P,
		db:                  s.cfg.DB,
		chain:               s.cfg.Chain,
		clock:               s.clock,
		ctxMap:              ctxMap,
		highestExpectedSlot: slots.Since(genesis),
		mode:                modeNonConstrained,
	})
	if err := queue.start(); err != nil {
		return err
	}
	for data := range queue.fetchedData {
		s.processFetchedDataRegSync(ctx, genesis, s.cfg.Chain.HeadSlot(), data)
	}
	log.WithFields(logrus.Fields{
		"syncedSlot":  s.cfg.Chain.HeadSlot(),
		"currentSlot": slots.Since(genesis),
	}).Info("Synced to head of chain")
	if err := queue.stop(); err != nil {
		log.WithError(err).Debug("Error stopping queue")
	}

	return nil
}

// processFetchedData processes data received from queue.
func (s *Service) processFetchedData(
	ctx context.Context, genesis time.Time, startSlot primitives.Slot, data *blocksQueueFetchedData) {
	defer s.updatePeerScorerStats(data.pid, startSlot)

	// Use Batch Block Verify to process and verify batches directly.
	if err := s.processBatchedBlocks(ctx, genesis, data.blocks, s.cfg.Chain.ReceiveBlockBatch, data.blobMap); err != nil {
		log.WithError(err).Warn("Skip processing batched blocks")
	}
}

// processFetchedData processes data received from queue.
func (s *Service) processFetchedDataRegSync(
	ctx context.Context, genesis time.Time, startSlot primitives.Slot, data *blocksQueueFetchedData) {
	defer s.updatePeerScorerStats(data.pid, startSlot)

	blockReceiver := s.cfg.Chain.ReceiveBlock
	invalidBlocks := 0
	blksWithoutParentCount := 0
	for _, blk := range data.blocks {
		if err := s.processBlock(ctx, genesis, blk, blockReceiver); err != nil {
			switch {
			case errors.Is(err, errBlockAlreadyProcessed):
				log.WithError(err).Debug("Block is not processed")
				invalidBlocks++
			case errors.Is(err, errParentDoesNotExist):
				blksWithoutParentCount++
				invalidBlocks++
			default:
				log.WithError(err).Warn("Block is not processed")
			}
			continue
		}
	}
	if blksWithoutParentCount > 0 {
		log.WithFields(logrus.Fields{
			"missingParent": fmt.Sprintf("%#x", data.blocks[0].Block().ParentRoot()),
			"firstSlot":     data.blocks[0].Block().Slot(),
			"lastSlot":      data.blocks[blksWithoutParentCount-1].Block().Slot(),
		}).Debug("Could not process batch blocks due to missing parent")
	}
	// Add more visible logging if all blocks cannot be processed.
	if len(data.blocks) == invalidBlocks {
		log.WithField("error", "Range had no valid blocks to process").Warn("Range is not processed")
	}
}

// highestFinalizedEpoch returns the absolute highest finalized epoch of all connected peers.
// Note this can be lower than our finalized epoch if we have no peers or peers that are all behind us.
func (s *Service) highestFinalizedEpoch() primitives.Epoch {
	highest := primitives.Epoch(0)
	for _, pid := range s.cfg.P2P.Peers().Connected() {
		peerChainState, err := s.cfg.P2P.Peers().ChainState(pid)
		if err == nil && peerChainState != nil && peerChainState.FinalizedEpoch > highest {
			highest = peerChainState.FinalizedEpoch
		}
	}

	return highest
}

// logSyncStatus and increment block processing counter.
func (s *Service) logSyncStatus(genesis time.Time, blk interfaces.ReadOnlyBeaconBlock, blkRoot [32]byte) {
	s.counter.Incr(1)
	rate := float64(s.counter.Rate()) / counterSeconds
	if rate == 0 {
		rate = 1
	}
	if slots.IsEpochStart(blk.Slot()) {
		timeRemaining := time.Duration(float64(slots.Since(genesis)-blk.Slot())/rate) * time.Second
		log.WithFields(logrus.Fields{
			"peers":           len(s.cfg.P2P.Peers().Connected()),
			"blocksPerSecond": fmt.Sprintf("%.1f", rate),
		}).Infof(
			"Processing block %s %d/%d - estimated time remaining %s",
			fmt.Sprintf("0x%s...", hex.EncodeToString(blkRoot[:])[:8]),
			blk.Slot(), slots.Since(genesis), timeRemaining,
		)
	}
}

// logBatchSyncStatus and increments the block processing counter.
func (s *Service) logBatchSyncStatus(genesis time.Time, blks []interfaces.ReadOnlySignedBeaconBlock, blkRoot [32]byte) {
	s.counter.Incr(int64(len(blks)))
	rate := float64(s.counter.Rate()) / counterSeconds
	if rate == 0 {
		rate = 1
	}
	firstBlk := blks[0]
	timeRemaining := time.Duration(float64(slots.Since(genesis)-firstBlk.Block().Slot())/rate) * time.Second
	log.WithFields(logrus.Fields{
		"peers":           len(s.cfg.P2P.Peers().Connected()),
		"blocksPerSecond": fmt.Sprintf("%.1f", rate),
	}).Infof(
		"Processing block batch of size %d starting from  %s %d/%d - estimated time remaining %s",
		len(blks), fmt.Sprintf("0x%s...", hex.EncodeToString(blkRoot[:])[:8]),
		firstBlk.Block().Slot(), slots.Since(genesis), timeRemaining,
	)
}

// processBlock performs basic checks on incoming block, and triggers receiver function.
func (s *Service) processBlock(
	ctx context.Context,
	genesis time.Time,
	blk interfaces.ReadOnlySignedBeaconBlock,
	blockReceiver blockReceiverFn,
) error {
	blkRoot, err := blk.Block().HashTreeRoot()
	if err != nil {
		return err
	}
	if s.isProcessedBlock(ctx, blk, blkRoot) {
		return fmt.Errorf("slot: %d , root %#x: %w", blk.Block().Slot(), blkRoot, errBlockAlreadyProcessed)
	}

	s.logSyncStatus(genesis, blk.Block(), blkRoot)
	if !s.cfg.Chain.HasBlock(ctx, blk.Block().ParentRoot()) {
		return fmt.Errorf("%w: (in processBlock, slot=%d) %#x", errParentDoesNotExist, blk.Block().Slot(), blk.Block().ParentRoot())
	}
	return blockReceiver(ctx, blk, blkRoot)
}

var ErrMissingBlobsForBlockCommitments = errors.New("blobs unavailable for processing block with kzg commitments")
var ErrMisalignedIndices = errors.New("unexpected ordering of blobs for block")
var ErrBlobCommitmentMismatch = errors.New("kzg_commitment in blob does not match block")

func (s *Service) processBatchedBlocks(ctx context.Context, genesis time.Time,
	blks []interfaces.ReadOnlySignedBeaconBlock, bFunc batchBlockReceiverFn, blobs map[[32]byte][]*ethpb.BlobSidecar) error {
	if len(blks) == 0 {
		return errors.New("0 blocks provided into method")
	}
	blockRoots := make([][32]byte, 0)
	unprocessed := make([]interfaces.ReadOnlySignedBeaconBlock, 0)
	for i := range blks {
		b := blks[i]
		root, err := b.Block().HashTreeRoot()
		if err != nil {
			return err
		}
		if s.isProcessedBlock(ctx, b, root) {
			continue
		}
		if len(blockRoots) > 0 {
			max := len(blockRoots) - 1
			proot := blockRoots[max]
			if proot != b.Block().ParentRoot() {
				pblock := unprocessed[max]
				return fmt.Errorf("expected linear block list with parent root of %#x (slot %d) but received %#x (slot %d)",
					proot, pblock.Block().Slot(), b.Block().ParentRoot(), b.Block().Slot())
			}
		}
		blockRoots = append(blockRoots, root)
		unprocessed = append(unprocessed, b)

		// TODO: This blob processing should be moved down into blocks_fetcher.go so that we can
		// try other peers if we get blobs that don't match the block commitments.
		denebB, err := b.PbDenebBlock()
		if err != nil {
			if errors.Is(err, consensus_types.ErrUnsupportedGetter) {
				// expected for pre-deneb blocks
				continue
			}
			return errors.Wrapf(err, "unexpected error when checking if block root=%#x is post-deneb", root)
		}
		kcs := denebB.GetBlock().GetBody().BlobKzgCommitments
		if len(kcs) == 0 {
			continue
		}
		bblobs := blobs[root]
		if len(bblobs) != len(kcs) {
			return errors.Wrapf(ErrMissingBlobsForBlockCommitments, "block root %#x, %d commitments, %d blobs", root, len(kcs), len(bblobs))
		}
		// make sure commitments match
		for ci, kc := range kcs {
			blobi := bblobs[ci]
			if blobi.Index != uint64(ci) {
				return errors.Wrapf(ErrMisalignedIndices, "block root %#x, block commitment %#x, offset %d, blob commitment %#x, Index=%d",
					root, kc, ci, blobi.KzgCommitment, blobi.Index)
			}
			if bytesutil.ToBytes48(kc) != bytesutil.ToBytes48(blobi.KzgCommitment) {
				return errors.Wrapf(ErrBlobCommitmentMismatch, "block root %#x, Index %d, block commitment=%#x, blob commitment=%#x",
					root, ci, kc, blobi.KzgCommitment)
			}
		}
	}

	if len(unprocessed) == 0 {
		maxIncoming := blks[len(blks)-1]
		maxRoot, err := maxIncoming.Block().HashTreeRoot()
		if err != nil {
			log.Errorf("TODO: use ROBlock so this kind of silly error handling isn't needed")
		}
		headSlot := s.cfg.Chain.HeadSlot()
		return fmt.Errorf("headSlot:%d, blockSlot:%d , root %#x:%w", headSlot, maxIncoming.Block().Slot(), maxRoot, errBlockAlreadyProcessed)
	}
	if !s.cfg.Chain.HasBlock(ctx, unprocessed[0].Block().ParentRoot()) {
		return fmt.Errorf("%w: %#x (in processBatchedBlocks, slot=%d)", errParentDoesNotExist, unprocessed[0].Block().ParentRoot(), unprocessed[0].Block().Slot())
	}
	s.logBatchSyncStatus(genesis, unprocessed, blockRoots[0])
	for _, root := range blockRoots {
		scs, has := blobs[root]
		if !has {
			continue
		}
		if err := s.cfg.DB.SaveBlobSidecar(ctx, scs); err != nil {
			return errors.Wrapf(err, "failed to save blobs for block %#x", root)
		}
	}
	if err := bFunc(ctx, unprocessed, blockRoots); err != nil {
		return err
	}
	return nil
}

// updatePeerScorerStats adjusts monitored metrics for a peer.
func (s *Service) updatePeerScorerStats(pid peer.ID, startSlot primitives.Slot) {
	if pid == "" {
		return
	}
	headSlot := s.cfg.Chain.HeadSlot()
	if startSlot >= headSlot {
		return
	}
	if diff := s.cfg.Chain.HeadSlot() - startSlot; diff > 0 {
		scorer := s.cfg.P2P.Peers().Scorers().BlockProviderScorer()
		scorer.IncrementProcessedBlocks(pid, uint64(diff))
	}
}

// isProcessedBlock checks DB and local cache for presence of a given block, to avoid duplicates.
func (s *Service) isProcessedBlock(ctx context.Context, blk interfaces.ReadOnlySignedBeaconBlock, blkRoot [32]byte) bool {
	cp := s.cfg.Chain.FinalizedCheckpt()
	finalizedSlot, err := slots.EpochStart(cp.Epoch)
	if err != nil {
		return false
	}
	// If block is before our finalized checkpoint
	// we do not process it.
	if blk.Block().Slot() <= finalizedSlot {
		return true
	}
	// If block exists in our db and is before or equal to our current head
	// we ignore it.
	if s.cfg.Chain.HeadSlot() >= blk.Block().Slot() && s.cfg.Chain.HasBlock(ctx, blkRoot) {
		return true
	}
	return false
}
