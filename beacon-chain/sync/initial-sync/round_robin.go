package initialsync

import (
	"context"
	"fmt"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/paulbellamy/ratecounter"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	prysmsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/beacon-chain/sync/peerstatus"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	eth "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
)

const blockBatchSize = 64
const maxPeersToSync = 15
const counterSeconds = 20

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
func (s *InitialSync) roundRobinSync(genesis time.Time) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	counter := ratecounter.NewRateCounter(counterSeconds * time.Second)

	var lastEmptyRequests int
	errChan := make(chan error)
	// Step 1 - Sync to end of finalized epoch.
	for s.chain.HeadSlot() < helpers.StartSlot(highestFinalizedEpoch()) {
		log.WithField("head status:", s.chain.HeadSlot()).Debugf("helpers.StartSlot(highestFinalizedEpoch()+1)-1: %v", helpers.StartSlot(highestFinalizedEpoch()))

		root, finalizedEpoch, peers := bestFinalized()

		var blocks []*eth.BeaconBlock

		// request a range of blocks to be requested from multiple peers.
		// Example:
		//   - number of peers = 4
		//   - range of block slots is 64...128
		//   Four requests will be spread across the peers using step argument to distribute the load
		//   i.e. the first peer is asked for block 64, 68, 72... while the second peer is asked for
		//   65, 69, 73... and so on for other peers.
		var request func(start uint64, step uint64, count uint64, peers []peer.ID, remainder int) ([]*eth.BeaconBlock, error)
		request = func(start uint64, step uint64, count uint64, peers []peer.ID, remainder int) ([]*eth.BeaconBlock, error) {
			if len(peers) == 0 {
				return nil, errors.WithStack(errors.New("no peers left to request blocks"))
			}
			var wg sync.WaitGroup

			// Handle block large block ranges of skipped slots.
			start += count * uint64(lastEmptyRequests*len(peers))

			for i, pid := range peers {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				stp := step * uint64(len(peers))
				cnt := count / uint64(len(peers))
				str := start + uint64(i)*step
				cnt = mathutil.Min(cnt, (helpers.StartSlot(finalizedEpoch+1)-str)/stp)
				// If the count was divided by an odd number of peers, there will be some blocks
				// missing from the first requests so we accommodate that scenario.
				if i < remainder {
					cnt++
				}
				// asking for no blocks may cause the client to hang. This should never happen and
				// the peer may return an error anyway, but we'll ask for at least one block.
				if cnt == 0 {
					break
				}
				req := &p2ppb.BeaconBlocksByRangeRequest{
					HeadBlockRoot: root,
					StartSlot:     str,
					Count:         cnt,
					Step:          stp,
				}

				// Fulfill requests asynchronously, in parallel, and wait for results from all.
				wg.Add(1)
				go func(i int, pid peer.ID) {
					defer wg.Done()
					resp, err := s.requestBlocks(ctx, req, pid)
					log.WithField("peer", pid.Pretty()).Debugf("Received %d blocks", len(resp))
					if err != nil {
						// fail over to other peers by splitting this requests evenly across them.
						ps := append(peers[:i], peers[i+1:]...)
						log.WithError(err).WithField(
							"remaining peers",
							len(ps),
						).WithField(
							"peer",
							pid.Pretty(),
						).Debug("Request failed, trying to round robin with other peers")
						if len(ps) == 0 {
							errChan <- errors.WithStack(errors.New("no peers left to request blocks"))
							return
						}
						_, err = request(str, stp, cnt /*count*/, ps, int(cnt)%len(ps) /*remainder*/)
						if err != nil {
							errChan <- err
							return
						}
					}
					blocks = append(blocks, resp...)
				}(i, pid)
			}

			// Wait for done signal or any error.
			done := make(chan interface{})
			go func() {
				wg.Wait()
				done <- true
			}()
			for {
				select {
				case err := <-errChan:
					return nil, err
				case <-done:
					return blocks, nil
				}
			}
		}

		blocks, err := request(
			s.chain.HeadSlot()+1, // start
			1,                    // step
			blockBatchSize,       // count
			peers,                // peers
			0,                    // remainder
		)
		if err != nil {
			return err
		}

		// Since the block responses were appended to the list, we must sort them in order to
		// process sequentially. This method doesn't make much wall time compared to block
		// processing.
		sort.Slice(blocks, func(i, j int) bool {
			return blocks[i].Slot < blocks[j].Slot
		})

		for _, blk := range blocks {
			logSyncStatus(genesis, blk, peers, counter)
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

	log.Debug("Synced to finalized epoch - now syncing blocks up to current head")

	if s.chain.HeadSlot() == slotsSinceGenesis(genesis) {
		return nil
	}

	// Step 2 - sync to head from any single peer.
	// This step might need to be improved for cases where there has been a long period since
	// finality. This step is less important than syncing to finality in terms of threat
	// mitigation. We are already convinced that we are on the correct finalized chain. Any blocks
	// we receive there after must build on the finalized chain or be considered invalid during
	// fork choice resolution / block processing.
	root, _, peers := bestFinalized()
	for head := slotsSinceGenesis(genesis); s.chain.HeadSlot() < head; {
		req := &p2ppb.BeaconBlocksByRangeRequest{
			HeadBlockRoot: root,
			StartSlot:     s.chain.HeadSlot() + 1,
			Count:         mathutil.Min(slotsSinceGenesis(genesis)-s.chain.HeadSlot()+1, 256),
			Step:          1,
		}

		var resp []*eth.BeaconBlock
		var err error
		var p peer.ID
		for _, p = range peers {
			log.WithField("req", req).WithField("peer", p.Pretty()).Debug(
				"Sending batch block request",
			)
			resp, err = s.requestBlocks(ctx, req, p)
			if err == nil {
				break
			}
		}

		for _, blk := range resp {
			logSyncStatus(genesis, blk, []peer.ID{p}, counter)
			if err := s.chain.ReceiveBlockNoPubsubForkchoice(ctx, blk); err != nil {
				return err
			}
		}
		if len(resp) == 0 {
			break
		}
	}

	return nil
}

// requestBlocks by range to a specific peer.
func (s *InitialSync) requestBlocks(ctx context.Context, req *p2ppb.BeaconBlocksByRangeRequest, pid peer.ID) ([]*eth.BeaconBlock, error) {
	log.WithField("peer", pid.Pretty()).WithField("req", req).Debug("Requesting blocks...")
	stream, err := s.p2p.Send(ctx, req, pid)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request to peer")
	}
	defer stream.Close()

	resp := make([]*eth.BeaconBlock, 0, req.Count)
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

// highestFinalizedEpoch as reported by peers. This is the absolute highest finalized epoch as
// reported by peers.
func highestFinalizedEpoch() uint64 {
	_, epoch, _ := bestFinalized()
	return epoch
}

// bestFinalized returns the highest finalized epoch that is agreed upon by the majority of
// peers. This method may not return the absolute highest finalized, but the finalized epoch in
// which most peers can serve blocks. Ideally, all peers would be reporting the same finalized
// epoch.
// Returns the best finalized root, epoch number, and peers that agree.
func bestFinalized() ([]byte, uint64, []peer.ID) {
	finalized := make(map[[32]byte]uint64)
	rootToEpoch := make(map[[32]byte]uint64)
	for _, k := range peerstatus.Keys() {
		s := peerstatus.Get(k)
		r := bytesutil.ToBytes32(s.FinalizedRoot)
		finalized[r]++
		rootToEpoch[r] = s.FinalizedEpoch
	}

	var mostVotedFinalizedRoot [32]byte
	var mostVotes uint64
	for root, count := range finalized {
		if count > mostVotes {
			mostVotes = count
			mostVotedFinalizedRoot = root
		}
	}

	var pids []peer.ID
	for _, k := range peerstatus.Keys() {
		s := peerstatus.Get(k)
		if s.FinalizedEpoch >= rootToEpoch[mostVotedFinalizedRoot] {
			pids = append(pids, k)
			if len(pids) >= maxPeersToSync {
				break
			}
		}
	}

	return mostVotedFinalizedRoot[:], rootToEpoch[mostVotedFinalizedRoot], pids
}

// bestPeer returns the peer ID of the peer reporting the highest head slot.
func bestPeer() peer.ID {
	var best peer.ID
	var bestSlot uint64
	for _, k := range peerstatus.Keys() {
		s := peerstatus.Get(k)
		if s.HeadSlot >= bestSlot {
			bestSlot = s.HeadSlot
			best = k
		}
	}
	return best
}

// logSyncStatus and increment block processing counter.
func logSyncStatus(genesis time.Time, blk *eth.BeaconBlock, peers []peer.ID, counter *ratecounter.RateCounter) {
	counter.Incr(1)
	rate := float64(counter.Rate()) / counterSeconds
	if rate == 0 {
		rate = 1
	}
	timeRemaining := time.Duration(float64(slotsSinceGenesis(genesis)-blk.Slot)/rate) * time.Second
	log.WithField(
		"peers",
		fmt.Sprintf("%d/%d", len(peers), len(peerstatus.Keys())),
	).WithField(
		"blocksPerSecond",
		fmt.Sprintf("%.1f", rate),
	).Infof(
		"Processing block %d/%d - estimated time remaining %s",
		blk.Slot,
		slotsSinceGenesis(genesis),
		timeRemaining,
	)
}
