package initialsync

import (
	"bytes"
	"context"
	"io"
	"math"
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
	"github.com/prysmaticlabs/prysm/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	prysmsync "github.com/prysmaticlabs/prysm/beacon-chain/sync"
	p2ppb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

const (
	// maxPendingRequests limits how many concurrent fetch request one can initiate.
	maxPendingRequests = 8
	// allowedBlocksPerSecond is number of blocks (per peer) fetcher can request per second.
	allowedBlocksPerSecond = 32.0
	// blockBatchSize is a limit on number of blocks fetched per request.
	blockBatchSize = 32
	// peersPercentagePerRequest caps percentage of peers to be used in a request.
	peersPercentagePerRequest = 0.75
	// handshakePollingInterval is a polling interval for checking the number of received handshakes.
	handshakePollingInterval = 5 * time.Second
)

var (
	errNoPeersAvailable = errors.New("no peers available, waiting for reconnect")
	errFetcherCtxIsDone = errors.New("fetcher's context is done, reinitialize")
	errSlotIsTooHigh    = errors.New("slot is higher than the finalized slot")
)

// blocksFetcherConfig is a config to setup the block fetcher.
type blocksFetcherConfig struct {
	headFetcher blockchain.HeadFetcher
	p2p         p2p.P2P
}

// blocksFetcher is a service to fetch chain data from peers.
// On an incoming requests, requested block range is evenly divided
// among available peers (for fair network load distribution).
type blocksFetcher struct {
	sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
	headFetcher    blockchain.HeadFetcher
	p2p            p2p.P2P
	rateLimiter    *leakybucket.Collector
	fetchRequests  chan *fetchRequestParams
	fetchResponses chan *fetchRequestResponse
	quit           chan struct{} // termination notifier
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
	start, count uint64
	blocks       []*eth.SignedBeaconBlock
	err          error
	peers        []peer.ID
}

// newBlocksFetcher creates ready to use fetcher.
func newBlocksFetcher(ctx context.Context, cfg *blocksFetcherConfig) *blocksFetcher {
	ctx, cancel := context.WithCancel(ctx)
	rateLimiter := leakybucket.NewCollector(
		allowedBlocksPerSecond, /* rate */
		allowedBlocksPerSecond, /* capacity */
		false                   /* deleteEmptyBuckets */)

	return &blocksFetcher{
		ctx:            ctx,
		cancel:         cancel,
		headFetcher:    cfg.headFetcher,
		p2p:            cfg.p2p,
		rateLimiter:    rateLimiter,
		fetchRequests:  make(chan *fetchRequestParams, maxPendingRequests),
		fetchResponses: make(chan *fetchRequestResponse, maxPendingRequests),
		quit:           make(chan struct{}),
	}
}

// start boots up the fetcher, which starts listening for incoming fetch requests.
func (f *blocksFetcher) start() error {
	select {
	case <-f.ctx.Done():
		return errFetcherCtxIsDone
	default:
		go f.loop()
		return nil
	}
}

// stop terminates all fetcher operations.
func (f *blocksFetcher) stop() {
	f.cancel()
	<-f.quit // make sure that loop() is done
}

// requestResponses exposes a channel into which fetcher pushes generated request responses.
func (f *blocksFetcher) requestResponses() <-chan *fetchRequestResponse {
	return f.fetchResponses
}

// loop is a main fetcher loop, listens for incoming requests/cancellations, forwards outgoing responses.
func (f *blocksFetcher) loop() {
	defer close(f.quit)

	// Wait for all loop's goroutines to finish, and safely release resources.
	wg := &sync.WaitGroup{}
	defer func() {
		wg.Wait()
		close(f.fetchResponses)
	}()

	for {
		// Make sure there is are available peers before processing requests.
		if _, err := f.waitForMinimumPeers(f.ctx); err != nil {
			log.Error(err)
		}

		select {
		case <-f.ctx.Done():
			log.Debug("Context closed, exiting goroutine (blocks fetcher)")
			return
		case req := <-f.fetchRequests:
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case <-f.ctx.Done():
				case f.fetchResponses <- f.handleRequest(req.ctx, req.start, req.count):
				}
			}()
		}
	}
}

// scheduleRequest adds request to incoming queue.
func (f *blocksFetcher) scheduleRequest(ctx context.Context, start, count uint64) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	request := &fetchRequestParams{
		ctx:   ctx,
		start: start,
		count: count,
	}
	select {
	case <-f.ctx.Done():
		return errFetcherCtxIsDone
	case f.fetchRequests <- request:
	}
	return nil
}

// handleRequest parses fetch request and forwards it to response builder.
func (f *blocksFetcher) handleRequest(ctx context.Context, start, count uint64) *fetchRequestResponse {
	ctx, span := trace.StartSpan(ctx, "initialsync.handleRequest")
	defer span.End()

	response := &fetchRequestResponse{
		start:  start,
		count:  count,
		blocks: []*eth.SignedBeaconBlock{},
		err:    nil,
		peers:  []peer.ID{},
	}

	if ctx.Err() != nil {
		response.err = ctx.Err()
		return response
	}

	headEpoch := helpers.SlotToEpoch(f.headFetcher.HeadSlot())
	root, finalizedEpoch, peers := f.p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, headEpoch)

	if len(peers) == 0 {
		response.err = errNoPeersAvailable
		return response
	}

	// Short circuit start far exceeding the highest finalized epoch in some infinite loop.
	highestFinalizedSlot := helpers.StartSlot(finalizedEpoch + 1)
	if start > highestFinalizedSlot {
		response.err = errSlotIsTooHigh
		return response
	}

	blocks, err := f.collectPeerResponses(ctx, root, finalizedEpoch, start, 1, count, peers)
	if err != nil {
		response.err = err
		return response
	}

	response.blocks = blocks
	response.peers = peers
	return response
}

// collectPeerResponses orchestrates block fetching from the available peers.
// In each request a range of blocks is to be requested from multiple peers.
// Example:
//   - number of peers = 4
//   - range of block slots is 64...128
//   Four requests will be spread across the peers using step argument to distribute the load
//   i.e. the first peer is asked for block 64, 68, 72... while the second peer is asked for
//   65, 69, 73... and so on for other peers.
func (f *blocksFetcher) collectPeerResponses(
	ctx context.Context,
	root []byte,
	finalizedEpoch, start, step, count uint64,
	peers []peer.ID,
) ([]*eth.SignedBeaconBlock, error) {
	ctx, span := trace.StartSpan(ctx, "initialsync.collectPeerResponses")
	defer span.End()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	peers = f.selectPeers(peers)
	if len(peers) == 0 {
		return nil, errNoPeersAvailable
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
		return nil, errSlotIsTooHigh
	}

	// Spread load evenly among available peers.
	perPeerCount := mathutil.Min(count/uint64(len(peers)), allowedBlocksPerSecond)
	remainder := int(count % uint64(len(peers)))
	for i, pid := range peers {
		start, step := start+uint64(i)*step, step*uint64(len(peers))

		// If the count was divided by an odd number of peers, there will be some blocks
		// missing from the first requests so we accommodate that scenario.
		count := perPeerCount
		if i < remainder {
			count++
		}
		// Asking for no blocks may cause the client to hang.
		if count == 0 {
			p2pRequests.Done()
			continue
		}

		go func(ctx context.Context, pid peer.ID) {
			defer p2pRequests.Done()

			blocks, err := f.requestBeaconBlocksByRange(ctx, pid, root, start, step, count)
			if err != nil {
				select {
				case <-ctx.Done():
				case errChan <- err:
					return
				}
			}
			select {
			case <-ctx.Done():
			case blocksChan <- blocks:
			}
		}(ctx, pid)
	}

	var unionRespBlocks []*eth.SignedBeaconBlock
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
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

// requestBeaconBlocksByRange prepares BeaconBlocksByRange request, and handles possible stale peers
// (by resending the request).
func (f *blocksFetcher) requestBeaconBlocksByRange(
	ctx context.Context,
	pid peer.ID,
	root []byte,
	start, step, count uint64,
) ([]*eth.SignedBeaconBlock, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	req := &p2ppb.BeaconBlocksByRangeRequest{
		StartSlot: start,
		Count:     count,
		Step:      step,
	}

	resp, respErr := f.requestBlocks(ctx, req, pid)
	if respErr != nil {
		// Fail over to some other, randomly selected, peer.
		headEpoch := helpers.SlotToEpoch(f.headFetcher.HeadSlot())
		root1, _, peers := f.p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, headEpoch)
		if bytes.Compare(root, root1) != 0 {
			return nil, errors.Errorf("can not resend, root mismatch: %x:%x", root, root1)
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

	return resp, nil
}

// requestBlocks is a wrapper for handling BeaconBlocksByRangeRequest requests/streams.
func (f *blocksFetcher) requestBlocks(
	ctx context.Context,
	req *p2ppb.BeaconBlocksByRangeRequest,
	pid peer.ID,
) ([]*eth.SignedBeaconBlock, error) {
	f.Lock()
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
	}).Debug("Requesting blocks")
	f.Unlock()
	stream, err := f.p2p.Send(ctx, req, p2p.RPCBlocksByRangeTopic, pid)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := stream.Close(); err != nil {
			log.WithError(err).Error("Failed to close stream")
		}
	}()

	resp := make([]*eth.SignedBeaconBlock, 0, req.Count)
	for {
		blk, err := prysmsync.ReadChunkedBlock(stream, f.p2p)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		resp = append(resp, blk)
	}

	return resp, nil
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

	randGenerator := rand.New(rand.NewSource(roughtime.Now().Unix()))
	randGenerator.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

	return peers[0], nil
}

// waitForMinimumPeers spins and waits up until enough peers are available.
func (f *blocksFetcher) waitForMinimumPeers(ctx context.Context) ([]peer.ID, error) {
	required := params.BeaconConfig().MaxPeersToSync
	if flags.Get().MinimumSyncPeers < required {
		required = flags.Get().MinimumSyncPeers
	}
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		headEpoch := helpers.SlotToEpoch(f.headFetcher.HeadSlot())
		_, _, peers := f.p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, headEpoch)
		if len(peers) >= required {
			return peers, nil
		}
		log.WithFields(logrus.Fields{
			"suitable": len(peers),
			"required": required}).Info("Waiting for enough suitable peers before syncing")
		time.Sleep(handshakePollingInterval)
	}
}

// selectPeers returns transformed list of peers (randomized, constrained if necessary).
func (f *blocksFetcher) selectPeers(peers []peer.ID) []peer.ID {
	if len(peers) == 0 {
		return peers
	}

	// Shuffle peers to prevent a bad peer from
	// stalling sync with invalid blocks.
	randGenerator := rand.New(rand.NewSource(roughtime.Now().Unix()))
	randGenerator.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})

	required := params.BeaconConfig().MaxPeersToSync
	if flags.Get().MinimumSyncPeers < required {
		required = flags.Get().MinimumSyncPeers
	}

	limit := uint64(math.Round(float64(len(peers)) * peersPercentagePerRequest))
	limit = mathutil.Max(limit, uint64(required))
	limit = mathutil.Min(limit, uint64(len(peers)))
	return peers[:limit]
}

// nonSkippedSlotAfter checks slots after the given one in an attempt to find non-empty future slot.
func (f *blocksFetcher) nonSkippedSlotAfter(ctx context.Context, slot uint64) (uint64, error) {
	headEpoch := helpers.SlotToEpoch(f.headFetcher.HeadSlot())
	_, epoch, peers := f.p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, headEpoch)
	if len(peers) == 0 {
		return 0, errNoPeersAvailable
	}

	randGenerator := rand.New(rand.NewSource(roughtime.Now().Unix()))
	nextPID := func() peer.ID {
		randGenerator.Shuffle(len(peers), func(i, j int) {
			peers[i], peers[j] = peers[j], peers[i]
		})
		return peers[0]
	}

	for slot <= helpers.StartSlot(epoch+1) {
		req := &p2ppb.BeaconBlocksByRangeRequest{
			StartSlot: slot + 1,
			Count:     blockBatchSize,
			Step:      1,
		}

		blocks, err := f.requestBlocks(ctx, req, nextPID())
		if err != nil {
			return slot, err
		}

		if len(blocks) > 0 {
			slots := make([]uint64, len(blocks))
			for i, block := range blocks {
				slots[i] = block.Block.Slot
			}
			return blocks[0].Block.Slot, nil
		}
		slot += blockBatchSize
	}

	return slot, nil
}

// bestFinalizedSlot returns the highest finalized slot of the majority of connected peers.
func (f *blocksFetcher) bestFinalizedSlot() uint64 {
	headEpoch := helpers.SlotToEpoch(f.headFetcher.HeadSlot())
	_, finalizedEpoch, _ := f.p2p.Peers().BestFinalized(params.BeaconConfig().MaxPeersToSync, headEpoch)
	return helpers.StartSlot(finalizedEpoch)
}
