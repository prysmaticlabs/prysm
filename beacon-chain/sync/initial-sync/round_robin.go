package initialsync

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/paulbellamy/ratecounter"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	blockfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/block"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	prysmsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

const blockBatchSize = 64
const counterSeconds = 20
const refreshTime = 6 * time.Second

// Round Robin sync looks at the latest peer statuses and syncs with the highest
// finalized peer.
//
// Step 1 - Sync to finalized epoch.
// Sync with peers of lowest finalized root with epoch greater than head state.
//
// Step 2 - Sync to head from finalized epoch.
// Using the finalized root as the head_block_root and the epoch start slot
// after the finalized epoch, request blocks to head from some subset of peers
// where step = 1.
func (s *Service) roundRobinSync(genesis time.Time) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer s.chain.ClearCachedStates()

	if cfg := featureconfig.Get(); cfg.EnableSkipSlotsCache {
		cfg.EnableSkipSlotsCache = false
		featureconfig.Init(cfg)
		defer func() {
			cfg := featureconfig.Get()
			cfg.EnableSkipSlotsCache = true
			featureconfig.Init(cfg)
		}()
	}

	counter := ratecounter.NewRateCounter(counterSeconds * time.Second)
	randGenerator := rand.New(rand.NewSource(time.Now().Unix()))
	var lastEmptyRequests int
	highestFinalizedSlot := helpers.StartSlot(s.highestFinalizedEpoch() + 1)

	fetcher := newBlocksFetcher(&blocksFetcherConfig{
		ctx:         ctx,
		headFetcher: s.chain,
		p2p:         s.p2p,
		rateLimiter: s.blocksRateLimiter,
	})
	fetcher.start()

	// Step 1 - Sync to end of finalized epoch.
	for s.chain.HeadSlot() < highestFinalizedSlot {
		_, finalizedEpoch, peers := s.p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, helpers.SlotToEpoch(s.chain.HeadSlot()))
		if len(peers) == 0 {
			log.Warn("No peers; waiting for reconnect")
			time.Sleep(refreshTime)
			continue
		}

		if len(peers) >= flags.Get().MinimumSyncPeers {
			highestFinalizedSlot = helpers.StartSlot(finalizedEpoch + 1)
		}

		// shuffle peers to prevent a bad peer from
		// stalling sync with invalid blocks
		randGenerator.Shuffle(len(peers), func(i, j int) {
			peers[i], peers[j] = peers[j], peers[i]
		})

		startBlock := s.chain.HeadSlot() + 1
		skippedBlocks := blockBatchSize * uint64(lastEmptyRequests*len(peers))
		if startBlock+skippedBlocks > helpers.StartSlot(finalizedEpoch+1) {
			log.WithField("finalizedEpoch", finalizedEpoch).Debug("Requested block range is greater than the finalized epoch")
			break
		}
		startBlock += skippedBlocks

		// Operations below are simple port of the existing functionality, where synchronization is assumed.
		// This will be refactored and implemented w/i queue, once that component is ready.
		request := func() ([]*eth.SignedBeaconBlock, error) {
			select {
			case <-time.After(60 * time.Second): // extra pre-caution
				return nil, errors.New("it takes too long to receive data from fetcher, unblock")
			case resp, ok := <-fetcher.iter(): // fetcher will stop when upstream ctx is closed
				if !ok { // channel closed
					return nil, errors.New("block fetcher is not running")
				}
				if resp.err != nil {
					return nil, resp.err
				}
				return resp.blocks, nil
			}
		}

		go fetcher.scheduleRequest(&fetchRequestParams{
			start: startBlock,
			count: blockBatchSize,
		})

		blocks, err := request()
		if err != nil {
			log.WithError(err).Error("Round robing sync request failed")
			continue
		}

		// Since the block responses were appended to the list, we must sort them in order to
		// process sequentially. This method doesn't make much wall time compared to block
		// processing.
		sort.Slice(blocks, func(i, j int) bool {
			return blocks[i].Block.Slot < blocks[j].Block.Slot
		})

		for _, blk := range blocks {
			s.logSyncStatus(genesis, blk.Block, peers, counter)
			if !s.db.HasBlock(ctx, bytesutil.ToBytes32(blk.Block.ParentRoot)) {
				log.Debugf("Beacon node doesn't have a block in db with root %#x", blk.Block.ParentRoot)
				continue
			}
			s.blockNotifier.BlockFeed().Send(&feed.Event{
				Type: blockfeed.ReceivedBlock,
				Data: &blockfeed.ReceivedBlockData{SignedBlock: blk},
			})
			if featureconfig.Get().InitSyncNoVerify {
				if err := s.chain.ReceiveBlockNoVerify(ctx, blk); err != nil {
					return err
				}
			} else {
				if err := s.chain.ReceiveBlockNoPubsubForkchoice(ctx, blk); err != nil {
					return err
				}
			}
		}
		// If there were no blocks in the last request range, increment the counter so the same
		// range isn't requested again on the next loop as the headSlot didn't change.
		if len(blocks) == 0 {
			lastEmptyRequests++
		} else {
			lastEmptyRequests = 0
		}
	}

	fetcher.stop()

	log.Debug("Synced to finalized epoch - now syncing blocks up to current head")

	if s.chain.HeadSlot() == helpers.SlotsSince(genesis) {
		return nil
	}

	// Step 2 - sync to head from any single peer.
	// This step might need to be improved for cases where there has been a long period since
	// finality. This step is less important than syncing to finality in terms of threat
	// mitigation. We are already convinced that we are on the correct finalized chain. Any blocks
	// we receive there after must build on the finalized chain or be considered invalid during
	// fork choice resolution / block processing.
	root, _, pids := s.p2p.Peers().BestFinalized(1 /* maxPeers */, s.highestFinalizedEpoch())
	for len(pids) == 0 {
		log.Info("Waiting for a suitable peer before syncing to the head of the chain")
		time.Sleep(refreshTime)
		root, _, pids = s.p2p.Peers().BestFinalized(1 /* maxPeers */, s.highestFinalizedEpoch())
	}
	best := pids[0]

	for head := helpers.SlotsSince(genesis); s.chain.HeadSlot() < head; {
		req := &p2ppb.BeaconBlocksByRangeRequest{
			HeadBlockRoot: root,
			StartSlot:     s.chain.HeadSlot() + 1,
			Count:         mathutil.Min(helpers.SlotsSince(genesis)-s.chain.HeadSlot()+1, 256),
			Step:          1,
		}

		log.WithField("req", req).WithField("peer", best.Pretty()).Debug(
			"Sending batch block request",
		)

		resp, err := s.requestBlocks(ctx, req, best)
		if err != nil {
			return err
		}

		for _, blk := range resp {
			s.logSyncStatus(genesis, blk.Block, []peer.ID{best}, counter)
			if err := s.chain.ReceiveBlockNoPubsubForkchoice(ctx, blk); err != nil {
				log.WithError(err).Error("Failed to process block, exiting init sync")
				return nil
			}
		}
		if len(resp) == 0 {
			break
		}
	}

	return nil
}

// requestBlocks by range to a specific peer.
func (s *Service) requestBlocks(ctx context.Context, req *p2ppb.BeaconBlocksByRangeRequest, pid peer.ID) ([]*eth.SignedBeaconBlock, error) {
	if s.blocksRateLimiter.Remaining(pid.String()) < int64(req.Count) {
		log.WithField("peer", pid).Debug("Slowing down for rate limit")
		time.Sleep(s.blocksRateLimiter.TillEmpty(pid.String()))
	}
	s.blocksRateLimiter.Add(pid.String(), int64(req.Count))
	log.WithFields(logrus.Fields{
		"peer":  pid,
		"start": req.StartSlot,
		"count": req.Count,
		"step":  req.Step,
		"head":  fmt.Sprintf("%#x", req.HeadBlockRoot),
	}).Debug("Requesting blocks")
	stream, err := s.p2p.Send(ctx, req, pid)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request to peer")
	}
	defer stream.Close()

	resp := make([]*eth.SignedBeaconBlock, 0, req.Count)
	for {
		blk, err := prysmsync.ReadChunkedBlock(stream, s.p2p)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to read chunked block")
		}
		resp = append(resp, blk)
	}

	return resp, nil
}

// highestFinalizedEpoch returns the absolute highest finalized epoch of all connected peers.
// Note this can be lower than our finalized epoch if we have no peers or peers that are all behind us.
func (s *Service) highestFinalizedEpoch() uint64 {
	highest := uint64(0)
	for _, pid := range s.p2p.Peers().Connected() {
		peerChainState, err := s.p2p.Peers().ChainState(pid)
		if err == nil && peerChainState != nil && peerChainState.FinalizedEpoch > highest {
			highest = peerChainState.FinalizedEpoch
		}
	}

	return highest
}

// logSyncStatus and increment block processing counter.
func (s *Service) logSyncStatus(genesis time.Time, blk *eth.BeaconBlock, syncingPeers []peer.ID, counter *ratecounter.RateCounter) {
	counter.Incr(1)
	rate := float64(counter.Rate()) / counterSeconds
	if rate == 0 {
		rate = 1
	}
	timeRemaining := time.Duration(float64(helpers.SlotsSince(genesis)-blk.Slot)/rate) * time.Second
	log.WithField(
		"peers",
		fmt.Sprintf("%d/%d", len(syncingPeers), len(s.p2p.Peers().Connected())),
	).WithField(
		"blocksPerSecond",
		fmt.Sprintf("%.1f", rate),
	).Infof(
		"Processing block %d/%d - estimated time remaining %s",
		blk.Slot,
		helpers.SlotsSince(genesis),
		timeRemaining,
	)
}
