package initialsync

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/kevinms/leakybucket-go"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	eth "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	prysmsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var errNoPeersAvailable = errors.New("no peers available, waiting for reconnect")

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
	ctx   context.Context // if provided, it is used instead of global fetcher's context
	start uint64          // starting slot
	count uint64          // how many slots to receive (fetcher may return fewer slots)
}

// fetchRequestResponse is a combined type to hold results of both successful executions and errors.
// Valid usage pattern will be to check whether result's `err` is nil, before using `blocks`.
type fetchRequestResponse struct {
	params *fetchRequestParams
	blocks []*eth.SignedBeaconBlock
	err    error
	peers  []peer.ID
}

// newBlocksFetcher creates ready to use fetcher
func newBlocksFetcher(cfg *blocksFetcherConfig) *blocksFetcher {
	rateLimiter := cfg.rateLimiter
	if rateLimiter == nil {
		rateLimiter = leakybucket.NewCollector(allowedBlocksPerSecond, allowedBlocksPerSecond, false)
	}

	ctx := cfg.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	return &blocksFetcher{
		ctx:                    ctx,
		headFetcher:            cfg.headFetcher,
		p2p:                    cfg.p2p,
		rateLimiter:            rateLimiter,
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
	select {
	case <-f.quit:
	default:
		close(f.quit)
	}
}

// requestResponses exposes a channel into which fetcher pushes generated request responses.
func (f *blocksFetcher) requestResponses() <-chan *fetchRequestResponse {
	return f.receivedFetchResponses
}

// loop is a main fetcher loop, listens for incoming requests/cancellations, forwards outgoing responses
func (f *blocksFetcher) loop() {
	defer close(f.receivedFetchResponses)

	for {
		select {
		case <-f.ctx.Done():
			// Upstream context is done.
			err := errors.New("upstream context canceled")
			log.WithError(err).Warn("Block fetch loop closed unexpectedly")
			f.receivedFetchResponses <- &fetchRequestResponse{
				err: err,
			}
			f.stop()
			return
		case <-f.quit:
			// Terminating abort all operations.
			log.Debug("Blocks fetcher received a stop request.")
			return
		case req := <-f.requests:
			if req.ctx == nil {
				req.ctx = f.ctx
			}
			go f.handleRequest(req)
		}
	}
}

// scheduleRequest adds request to incoming queue.
func (f *blocksFetcher) scheduleRequest(ctx context.Context, start, count uint64) {
	select {
	case <-f.quit:
		return
	default:
		f.requests <- &fetchRequestParams{
			ctx:   ctx,
			start: start,
			count: count,
		}
	}
}

// handleRequest parses fetch request and forwards it to response builder.
func (f *blocksFetcher) handleRequest(req *fetchRequestParams) {
	if req.ctx.Err() != nil {
		f.receivedFetchResponses <- &fetchRequestResponse{
			params: req,
			err:    req.ctx.Err(),
		}
		return
	}

	root, finalizedEpoch, peers := f.p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, helpers.SlotToEpoch(f.headFetcher.HeadSlot()))
	log.WithFields(logrus.Fields{
		"request":        req,
		"finalizedEpoch": finalizedEpoch,
		"numPeers":       len(peers),
	}).Debug("Block fetcher receives request")

	if len(peers) == 0 {
		log.Error(errNoPeersAvailable)
		return
	}

	// Short circuit start far exceeding the highest finalized epoch in some infinite loop.
	highestFinalizedSlot := helpers.StartSlot(finalizedEpoch + 1)
	if req.start > highestFinalizedSlot {
		err := errors.Errorf("requested a start slot of %d which is greater than the next highest slot of %d", req.start, highestFinalizedSlot)
		log.WithError(err).Debug("Block fetch request failed")
		f.receivedFetchResponses <- &fetchRequestResponse{
			params: req,
			err:    err,
		}
		return
	}

	resp, err := f.collectPeerResponses(req.ctx, root, finalizedEpoch, req.start, 1, req.count, peers)
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
		peers:  peers,
	}
}

// collectPeerResponses orchestrates block fetching from the available peers.
// In each request a range of blocks is to be requested from multiple peers.
// Example:
//   - number of peers = 4
//   - range of block slots is 64...128
//   Four requests will be spread across the peers using step argument to distribute the load
//   i.e. the first peer is asked for block 64, 68, 72... while the second peer is asked for
//   65, 69, 73... and so on for other peers.
func (f *blocksFetcher) collectPeerResponses(ctx context.Context, root []byte, finalizedEpoch, start, step, count uint64, peers []peer.ID) ([]*eth.SignedBeaconBlock, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if len(peers) == 0 {
		return nil, errNoPeersAvailable
	}

	// Shuffle peers to prevent a bad peer from
	// stalling sync with invalid blocks.
	randGenerator := rand.New(rand.NewSource(time.Now().Unix()))
	randGenerator.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

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

	// Spread load evenly among available peers.
	perPeerCount := count / uint64(len(peers))
	remainder := int(count % uint64(len(peers)))
	log.WithFields(logrus.Fields{
		"start":        start,
		"count":        count,
		"perPeerCount": perPeerCount,
		"remainder":    remainder,
	}).Debug("Distribute request among available peers")
	for i, pid := range peers {
		start, step := start+uint64(i)*step, step*uint64(len(peers))

		// If the count was divided by an odd number of peers, there will be some blocks
		// missing from the first requests so we accommodate that scenario.
		count := perPeerCount
		if i < remainder {
			count++
		}
		// Asking for no blocks may cause the client to hang. This should never happen and
		// the peer may return an error anyway, but we'll ask for at least one block.
		if count == 0 {
			count++
		}

		go func(ctx context.Context, pid peer.ID) {
			defer p2pRequests.Done()

			blocks, err := f.requestBeaconBlocksByRange(ctx, pid, root, start, step, count)
			if err != nil {
				errChan <- err
				return
			}
			blocksChan <- blocks
		}(ctx, pid)
	}

	var unionRespBlocks []*eth.SignedBeaconBlock
	for {
		select {
		case err := <-errChan:
			return nil, err
		case resp, ok := <-blocksChan:
			if ok {
				unionRespBlocks = append(unionRespBlocks, resp...)
			} else {
				sort.Slice(unionRespBlocks, func(i, j int) bool {
					return unionRespBlocks[i].Block.Slot < unionRespBlocks[j].Block.Slot
				})
				return unionRespBlocks, nil
			}
		}
	}
}

// requestBeaconBlocksByRange prepares BeaconBlocksByRange request, and handles possible stale peers (by resending the request).
func (f *blocksFetcher) requestBeaconBlocksByRange(ctx context.Context, pid peer.ID, root []byte, start, step, count uint64) ([]*eth.SignedBeaconBlock, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	req := &p2ppb.BeaconBlocksByRangeRequest{
		HeadBlockRoot: root,
		StartSlot:     start,
		Count:         count,
		Step:          step,
	}

	resp, respErr := f.requestBlocks(ctx, req, pid)
	if respErr != nil {
		// Fail over to some other, randomly selected, peer.
		root1, _, peers := f.p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, helpers.SlotToEpoch(f.headFetcher.HeadSlot()))
		if bytes.Compare(root, root1) != 0 {
			return nil, errors.Errorf("can not forward request to another peer, root mismatch: %x:%x", root, root1)
		}
		newPID, err := selectFailOverPeer(pid, peers)
		if err != nil {
			return nil, err
		}

		log.WithError(respErr).WithFields(logrus.Fields{
			"numPeers":   len(peers),
			"failedPeer": pid.Pretty(),
			"newPeer":    newPID.Pretty(),
		}).Debug("Request failed, trying to forward request to another peer")

		return f.requestBeaconBlocksByRange(ctx, newPID, root, start, step, count)
	}

	log.WithField("peer", pid).WithField("count", len(resp)).Debug("Received blocks")
	return resp, nil
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

// selectFailOverPeer randomly selects fail over peer from the list of available peers.
func selectFailOverPeer(excludedPID peer.ID, peers []peer.ID) (peer.ID, error) {
	for i, pid := range peers {
		if pid == excludedPID {
			peers = append(peers[:i], peers[i+1:]...)
			break
		}
	}

	if len(peers) == 0 {
		return "", errNoPeersAvailable
	}

	randGenerator := rand.New(rand.NewSource(time.Now().Unix()))
	randGenerator.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

	return peers[0], nil
}
