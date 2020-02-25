package initialsync

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/kevinms/leakybucket-go"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	prysmsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

// blocksFetcherConfig is a config to setup the block fetcher.
type blocksFetcherConfig struct {
	ctx         context.Context
	headFetcher blockchain.HeadFetcher
	p2p         p2p.P2P
	rateLimiter *leakybucket.Collector
}

// blocksFetcher is a service to fetch chain data from peers.
// On an incoming requests, requested block range is evenly divided
// among available peers (for fair network load distribution).
type blocksFetcher struct {
	ctx                    context.Context
	headFetcher            blockchain.HeadFetcher
	p2p                    p2p.P2P
	rateLimiter            *leakybucket.Collector
	requests               chan *fetchRequestParams   // incoming fetch requests from downstream clients
	receivedFetchResponses chan *fetchRequestResponse // responses from peers are forwarded to downstream clients
	quit                   chan struct{}              // termination notifier
}

// fetchRequestParams holds parameters necessary to schedule a fetch request.
type fetchRequestParams struct {
	start uint64 // starting slot
	count uint64 // how many slots to receive (fetcher may return fewer slots)
}

// fetchRequestResponse is a combined type to hold results of both successful executions and errors.
// Valid usage pattern will be to check whether result's `err` is nil, before using `blocks`.
type fetchRequestResponse struct {
	params *fetchRequestParams
	blocks []*eth.SignedBeaconBlock
	err    error
}

// newBlocksFetcher creates ready to use fetcher
func newBlocksFetcher(cfg *blocksFetcherConfig) *blocksFetcher {
	return &blocksFetcher{
		ctx:                    cfg.ctx,
		headFetcher:            cfg.headFetcher,
		p2p:                    cfg.p2p,
		rateLimiter:            cfg.rateLimiter,
		requests:               make(chan *fetchRequestParams),
		receivedFetchResponses: make(chan *fetchRequestResponse),
		quit:                   make(chan struct{}),
	}
}

// start boots up the fetcher, which starts listening for incoming fetch requests.
func (f *blocksFetcher) start() {
	go f.loop()
}

// stop terminates all fetcher operations.
func (f *blocksFetcher) stop() {
	close(f.quit)
}

// iter returns an outgoing channel, on which consumers are expected to constantly iterate for results/errors.
func (f *blocksFetcher) iter() <-chan *fetchRequestResponse {
	return f.receivedFetchResponses
}

// loop is a main fetcher loop, listens for incoming requests/cancellations, forwards outgoing responses
func (f *blocksFetcher) loop() {
	defer close(f.receivedFetchResponses)

	randGenerator := rand.New(rand.NewSource(time.Now().Unix()))
	highestFinalizedSlot := helpers.StartSlot(f.highestFinalizedEpoch() + 1)
	maxPeersToSync := params.BeaconConfig().MaxPeersToSync

	for {
		select {
		case <-f.ctx.Done():
			// Upstream context is done.
			log.Warn("Upstream context canceled. Blocks fetcher is stopped.")
			return
		case <-f.quit:
			// Terminating abort all operations.
			return
		case req := <-f.requests:
			root, finalizedEpoch, peers := f.p2p.Peers().BestFinalized(maxPeersToSync, helpers.SlotToEpoch(f.headFetcher.HeadSlot()))
			log.WithFields(logrus.Fields{
				"request":        req,
				"finalizedEpoch": finalizedEpoch,
				"numPeers":       len(peers),
			}).Debug("Block fetcher receives request")

			if len(peers) == 0 {
				log.Warn("No peers available, waiting for reconnect...")
				time.Sleep(refreshTime)
				continue
			}

			if len(peers) >= flags.Get().MinimumSyncPeers {
				highestFinalizedSlot = helpers.StartSlot(finalizedEpoch + 1)
			}

			// Short circuit start far exceeding the highest finalized epoch in some infinite loop.
			if req.start > highestFinalizedSlot {
				err := errors.Errorf("requested a start slot of %d which is greater than the next highest slot of %d", req.start, highestFinalizedSlot)
				log.WithError(err).Debug("Block fetch request failed")
				f.receivedFetchResponses <- &fetchRequestResponse{
					params: req,
					err:    err,
				}
				continue
			}

			// Shuffle peers to prevent a bad peer from
			// stalling sync with invalid blocks.
			randGenerator.Shuffle(len(peers), func(i, j int) {
				peers[i], peers[j] = peers[j], peers[i]
			})

			// TODO(4815): Consider splitting peers into sets (when there are many of them),
			// so that block fetching is for lesser chunks and is less affected by some slow peer
			go func(root []byte, finalizedEpoch, start, count uint64, peers []peer.ID) {
				resp, err := f.processFetchRequest(root, finalizedEpoch, start, 1, count, peers)
				if err != nil {
					log.WithError(err).Debug("Block fetch request failed")
					f.receivedFetchResponses <- &fetchRequestResponse{
						params: req,
						err:    err,
					}
					return
				}

				f.receivedFetchResponses <- &fetchRequestResponse{
					params: req,
					blocks: resp,
				}
			}(root, finalizedEpoch, req.start, req.count, peers)
		}
	}
}

// scheduleRequest adds request to incoming queue.
// Should be non-blocking, as actual request processing is done asynchronously.
func (f *blocksFetcher) scheduleRequest(req *fetchRequestParams) {
	go func() { // non-blocking, we can re-throw requests within consuming method
		select {
		case <-f.quit:
			return
		case f.requests <- req:
			return
		}
	}()
}

// processFetchRequest orchestrates block fetching from the available peers.
// In each request a range of blocks is to be requested from multiple peers.
// Example:
//   - number of peers = 4
//   - range of block slots is 64...128
//   Four requests will be spread across the peers using step argument to distribute the load
//   i.e. the first peer is asked for block 64, 68, 72... while the second peer is asked for
//   65, 69, 73... and so on for other peers.
func (f *blocksFetcher) processFetchRequest(root []byte, finalizedEpoch, start, step, count uint64, peers []peer.ID) ([]*eth.SignedBeaconBlock, error) {
	if len(peers) == 0 {
		return nil, errors.WithStack(errors.New("no peers left to request blocks"))
	}

	p2pRequests := new(sync.WaitGroup)
	errChan := make(chan error)
	blocksChan := make(chan []*eth.SignedBeaconBlock)

	p2pRequests.Add(len(peers))
	go func() {
		p2pRequests.Wait()
		close(blocksChan)
	}()

	// Short circuit start far exceeding the highest finalized epoch in some infinite loop.
	highestFinalizedSlot := helpers.StartSlot(finalizedEpoch + 1)
	if start > highestFinalizedSlot {
		return nil, errors.Errorf("did not expect request for start slot of %d which is greater than expected %d", start, highestFinalizedSlot)
	}

	// Do not overflow the finalized epoch, spread load evenly among available peers.
	count = mathutil.Min(count, helpers.StartSlot(finalizedEpoch+1)-start+1)
	perPeerCount := count / uint64(len(peers))
	remainder := int(count % uint64(len(peers)))
	log.WithFields(logrus.Fields{
		"start":        start,
		"count":        count,
		"perPeerCount": perPeerCount,
		"remainder":    remainder,
	}).Debug("Distribute request amount peers")
	for i, pid := range peers {
		if f.ctx.Err() != nil {
			return nil, f.ctx.Err()
		}
		start := start + uint64(i)*step
		step := step * uint64(len(peers))
		// If the count was divided by an odd number of peers, there will be some blocks
		// missing from the first requests so we accommodate that scenario.
		count := perPeerCount
		if i < remainder {
			count++
		}
		// Asking for no blocks may cause the client to hang. This should never happen and
		// the peer may return an error anyway, but we'll ask for at least one block.
		if count == 0 {
			count = 1
		}
		req := &p2ppb.BeaconBlocksByRangeRequest{
			HeadBlockRoot: root,
			StartSlot:     start,
			Count:         count,
			Step:          step,
		}

		go func(i int, pid peer.ID) {
			defer p2pRequests.Done()

			resp, err := f.requestBlocks(f.ctx, req, pid)
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
				resp, err = f.processFetchRequest(root, finalizedEpoch, start, step, count, ps)
				if err != nil {
					errChan <- err
					return
				}
			}
			log.WithField("peer", pid).WithField("count", len(resp)).Debug("Received blocks")
			blocksChan <- resp
		}(i, pid)
	}

	var unionRespBlocks []*eth.SignedBeaconBlock
	for {
		select {
		case err := <-errChan:
			return nil, err
		case resp, ok := <-blocksChan:
			if ok {
				for _, block := range resp {
					// trim up to start + count*step (so that upper bound of returned blocks is deterministic)
					if block.Block.Slot > start+count*step {
						break
					}
					unionRespBlocks = append(unionRespBlocks, block)
				}
			} else {
				return unionRespBlocks, nil
			}
		}
	}
}

// requestBlocks is a wrapper for handling BeaconBlocksByRangeRequest requests/streams.
func (f *blocksFetcher) requestBlocks(ctx context.Context, req *p2ppb.BeaconBlocksByRangeRequest, pid peer.ID) ([]*eth.SignedBeaconBlock, error) {
	if f.rateLimiter.Remaining(pid.String()) < int64(req.Count) {
		log.WithField("peer", pid).Debug("Slowing down for rate limit")
		time.Sleep(f.rateLimiter.TillEmpty(pid.String()))
	}
	f.rateLimiter.Add(pid.String(), int64(req.Count))
	log.WithFields(logrus.Fields{
		"peer":  pid,
		"start": req.StartSlot,
		"count": req.Count,
		"step":  req.Step,
		"head":  fmt.Sprintf("%#x", req.HeadBlockRoot),
	}).Debug("Requesting blocks")
	stream, err := f.p2p.Send(ctx, req, pid)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request to peer")
	}
	defer stream.Close()

	resp := make([]*eth.SignedBeaconBlock, 0, req.Count)
	for {
		blk, err := prysmsync.ReadChunkedBlock(stream, f.p2p)
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
func (f *blocksFetcher) highestFinalizedEpoch() uint64 {
	highest := uint64(0)
	for _, pid := range f.p2p.Peers().Connected() {
		peerChainState, err := f.p2p.Peers().ChainState(pid)
		if err == nil && peerChainState != nil && peerChainState.FinalizedEpoch > highest {
			highest = peerChainState.FinalizedEpoch
		}
	}

	return highest
}
