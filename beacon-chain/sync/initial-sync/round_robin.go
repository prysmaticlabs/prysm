package initialsync

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/paulbellamy/ratecounter"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/sirupsen/logrus"
)

const (
	// counterSeconds is an interval over which an average rate will be calculated.
	counterSeconds = 20
)

// blockReceiverFn defines block receiving function.
type blockReceiverFn func(ctx context.Context, block interfaces.SignedBeaconBlock, blockRoot [32]byte) error

// batchBlockReceiverFn defines batch receiving function.
type batchBlockReceiverFn func(ctx context.Context, blks []interfaces.SignedBeaconBlock, roots [][32]byte) error

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
	queue := newBlocksQueue(ctx, &blocksQueueConfig{
		p2p:                 s.cfg.P2P,
		db:                  s.cfg.DB,
		chain:               s.cfg.Chain,
		highestExpectedSlot: highestFinalizedSlot,
		mode:                modeStopOnFinalizedEpoch,
	})
	if err := queue.start(); err != nil {
		return err
	}

	for data := range queue.fetchedData {
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
	queue := newBlocksQueue(ctx, &blocksQueueConfig{
		p2p:                 s.cfg.P2P,
		db:                  s.cfg.DB,
		chain:               s.cfg.Chain,
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
	ctx context.Context, genesis time.Time, startSlot types.Slot, data *blocksQueueFetchedData) {
	defer s.updatePeerScorerStats(data.pid, startSlot)

	// Use Batch Block Verify to process and verify batches directly.
	if err := s.processBatchedBlocks(ctx, genesis, data.blocks, s.cfg.Chain.ReceiveBlockBatch); err != nil {
		log.WithError(err).Warn("Skip processing batched blocks")
	}
}

// processFetchedData processes data received from queue.
func (s *Service) processFetchedDataRegSync(
	ctx context.Context, genesis time.Time, startSlot types.Slot, data *blocksQueueFetchedData) {
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
func (s *Service) highestFinalizedEpoch() types.Epoch {
	highest := types.Epoch(0)
	for _, pid := range s.cfg.P2P.Peers().Connected() {
		peerChainState, err := s.cfg.P2P.Peers().ChainState(pid)
		if err == nil && peerChainState != nil && peerChainState.FinalizedEpoch > highest {
			highest = peerChainState.FinalizedEpoch
		}
	}

	return highest
}

// logSyncStatus and increment block processing counter.
func (s *Service) logSyncStatus(genesis time.Time, blk interfaces.BeaconBlock, blkRoot [32]byte) {
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
func (s *Service) logBatchSyncStatus(genesis time.Time, blks []interfaces.SignedBeaconBlock, blkRoot [32]byte) {
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
	blk interfaces.SignedBeaconBlock,
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

func (s *Service) processBatchedBlocks(ctx context.Context, genesis time.Time,
	blks []interfaces.SignedBeaconBlock, bFunc batchBlockReceiverFn) error {
	if len(blks) == 0 {
		return errors.New("0 blocks provided into method")
	}
	firstBlock := blks[0]
	blkRoot, err := firstBlock.Block().HashTreeRoot()
	if err != nil {
		return err
	}
	headSlot := s.cfg.Chain.HeadSlot()
	for headSlot >= firstBlock.Block().Slot() && s.isProcessedBlock(ctx, firstBlock, blkRoot) {
		if len(blks) == 1 {
			return fmt.Errorf("headSlot:%d, blockSlot:%d , root %#x:%w", headSlot, firstBlock.Block().Slot(), blkRoot, errBlockAlreadyProcessed)
		}
		blks = blks[1:]
		firstBlock = blks[0]
		blkRoot, err = firstBlock.Block().HashTreeRoot()
		if err != nil {
			return err
		}
	}
	s.logBatchSyncStatus(genesis, blks, blkRoot)
	parentRoot := firstBlock.Block().ParentRoot()
	if !s.cfg.Chain.HasBlock(ctx, parentRoot) {
		return fmt.Errorf("%w: %#x (in processBatchedBlocks, slot=%d)", errParentDoesNotExist, firstBlock.Block().ParentRoot(), firstBlock.Block().Slot())
	}
	blockRoots := make([][32]byte, len(blks))
	blockRoots[0] = blkRoot
	for i := 1; i < len(blks); i++ {
		b := blks[i]
		if b.Block().ParentRoot() != blockRoots[i-1] {
			return fmt.Errorf("expected linear block list with parent root of %#x but received %#x",
				blockRoots[i-1][:], b.Block().ParentRoot())
		}
		blkRoot, err := b.Block().HashTreeRoot()
		if err != nil {
			return err
		}
		blockRoots[i] = blkRoot
	}
	return bFunc(ctx, blks, blockRoots)
}

// updatePeerScorerStats adjusts monitored metrics for a peer.
func (s *Service) updatePeerScorerStats(pid peer.ID, startSlot types.Slot) {
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
func (s *Service) isProcessedBlock(ctx context.Context, blk interfaces.SignedBeaconBlock, blkRoot [32]byte) bool {
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
