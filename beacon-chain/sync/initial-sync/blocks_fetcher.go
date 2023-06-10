package initialsync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p"
	p2pTypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/startup"
	prysmsync "github.com/prysmaticlabs/prysm/v4/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/v4/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	consensus_types "github.com/prysmaticlabs/prysm/v4/consensus-types"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	leakybucket "github.com/prysmaticlabs/prysm/v4/container/leaky-bucket"
	"github.com/prysmaticlabs/prysm/v4/crypto/rand"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/math"
	p2ppb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

const (
	// maxPendingRequests limits how many concurrent fetch request one can initiate.
	maxPendingRequests = 64
	// peersPercentagePerRequest caps percentage of peers to be used in a request.
	peersPercentagePerRequest = 0.75
	// handshakePollingInterval is a polling interval for checking the number of received handshakes.
	handshakePollingInterval = 5 * time.Second
	// peerLocksPollingInterval is a polling interval for checking if there are stale peer locks.
	peerLocksPollingInterval = 5 * time.Minute
	// peerLockMaxAge is maximum time before stale lock is purged.
	peerLockMaxAge = 60 * time.Minute
	// nonSkippedSlotsFullSearchEpochs how many epochs to check in full, before resorting to random
	// sampling of slots once per epoch
	nonSkippedSlotsFullSearchEpochs = 10
	// peerFilterCapacityWeight defines how peer's capacity affects peer's score. Provided as
	// percentage, i.e. 0.3 means capacity will determine 30% of peer's score.
	peerFilterCapacityWeight = 0.2
	// backtrackingMaxHops how many hops (during search for common ancestor in backtracking) to do
	// before giving up.
	backtrackingMaxHops = 128
)

var (
	errNoPeersAvailable      = errors.New("no peers available, waiting for reconnect")
	errFetcherCtxIsDone      = errors.New("fetcher's context is done, reinitialize")
	errSlotIsTooHigh         = errors.New("slot is higher than the finalized slot")
	errBlockAlreadyProcessed = errors.New("block is already processed")
	errParentDoesNotExist    = errors.New("beacon node doesn't have a parent in db with root")
	errNoPeersWithAltBlocks  = errors.New("no peers with alternative blocks found")
)

// Period to calculate expected limit for a single peer.
var blockLimiterPeriod = 30 * time.Second

// blocksFetcherConfig is a config to setup the block fetcher.
type blocksFetcherConfig struct {
	clock                    *startup.Clock
	ctxMap                   prysmsync.ContextByteVersions
	chain                    blockchainService
	p2p                      p2p.P2P
	db                       db.ReadOnlyDatabase
	peerFilterCapacityWeight float64
	mode                     syncMode
}

// blocksFetcher is a service to fetch chain data from peers.
// On an incoming requests, requested block range is evenly divided
// among available peers (for fair network load distribution).
type blocksFetcher struct {
	sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
	rand            *rand.Rand
	chain           blockchainService
	clock           *startup.Clock
	ctxMap          prysmsync.ContextByteVersions
	p2p             p2p.P2P
	db              db.ReadOnlyDatabase
	blocksPerPeriod uint64
	rateLimiter     *leakybucket.Collector
	peerLocks       map[peer.ID]*peerLock
	fetchRequests   chan *fetchRequestParams
	fetchResponses  chan *fetchRequestResponse
	capacityWeight  float64       // how remaining capacity affects peer selection
	mode            syncMode      // allows to use fetcher in different sync scenarios
	quit            chan struct{} // termination notifier
}

// peerLock restricts fetcher actions on per peer basis. Currently, used for rate limiting.
type peerLock struct {
	sync.Mutex
	accessed time.Time
}

// fetchRequestParams holds parameters necessary to schedule a fetch request.
type fetchRequestParams struct {
	ctx   context.Context // if provided, it is used instead of global fetcher's context
	start primitives.Slot // starting slot
	count uint64          // how many slots to receive (fetcher may return fewer slots)
}

// fetchRequestResponse is a combined type to hold results of both successful executions and errors.
// Valid usage pattern will be to check whether result's `err` is nil, before using `blocks`.
type fetchRequestResponse struct {
	pid      peer.ID
	start    primitives.Slot
	count    uint64
	blocks   []interfaces.ReadOnlySignedBeaconBlock
	blobMap  map[[32]byte][]*p2ppb.BlobSidecar
	blobsPid peer.ID
	err      error
}

// newBlocksFetcher creates ready to use fetcher.
func newBlocksFetcher(ctx context.Context, cfg *blocksFetcherConfig) *blocksFetcher {
	blocksPerPeriod := flags.Get().BlockBatchLimit
	allowedBlocksBurst := flags.Get().BlockBatchLimitBurstFactor * flags.Get().BlockBatchLimit
	// Allow fetcher to go almost to the full burst capacity (less a single batch).
	rateLimiter := leakybucket.NewCollector(
		float64(blocksPerPeriod), int64(allowedBlocksBurst-blocksPerPeriod),
		blockLimiterPeriod, false /* deleteEmptyBuckets */)

	capacityWeight := cfg.peerFilterCapacityWeight
	if capacityWeight >= 1 {
		capacityWeight = peerFilterCapacityWeight
	}

	ctx, cancel := context.WithCancel(ctx)
	return &blocksFetcher{
		ctx:             ctx,
		cancel:          cancel,
		rand:            rand.NewGenerator(),
		chain:           cfg.chain,
		clock:           cfg.clock,
		ctxMap:          cfg.ctxMap,
		p2p:             cfg.p2p,
		db:              cfg.db,
		blocksPerPeriod: uint64(blocksPerPeriod),
		rateLimiter:     rateLimiter,
		peerLocks:       make(map[peer.ID]*peerLock),
		fetchRequests:   make(chan *fetchRequestParams, maxPendingRequests),
		fetchResponses:  make(chan *fetchRequestResponse, maxPendingRequests),
		capacityWeight:  capacityWeight,
		mode:            cfg.mode,
		quit:            make(chan struct{}),
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
	defer func() {
		if f.rateLimiter != nil {
			f.rateLimiter.Free()
			f.rateLimiter = nil
		}
	}()
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

	// Periodically remove stale peer locks.
	go func() {
		ticker := time.NewTicker(peerLocksPollingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				f.removeStalePeerLocks(peerLockMaxAge)
			case <-f.ctx.Done():
				return
			}
		}
	}()

	// Main loop.
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
func (f *blocksFetcher) scheduleRequest(ctx context.Context, start primitives.Slot, count uint64) error {
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
func (f *blocksFetcher) handleRequest(ctx context.Context, start primitives.Slot, count uint64) *fetchRequestResponse {
	ctx, span := trace.StartSpan(ctx, "initialsync.handleRequest")
	defer span.End()

	response := &fetchRequestResponse{
		start:  start,
		count:  count,
		blocks: []interfaces.ReadOnlySignedBeaconBlock{},
		err:    nil,
	}

	if ctx.Err() != nil {
		response.err = ctx.Err()
		return response
	}

	_, targetEpoch, peers := f.calculateHeadAndTargetEpochs()
	if len(peers) == 0 {
		response.err = errNoPeersAvailable
		return response
	}

	// Short circuit start far exceeding the highest finalized epoch in some infinite loop.
	if f.mode == modeStopOnFinalizedEpoch {
		highestFinalizedSlot := params.BeaconConfig().SlotsPerEpoch.Mul(uint64(targetEpoch + 1))
		if start > highestFinalizedSlot {
			response.err = fmt.Errorf("%w, slot: %d, highest finalized slot: %d",
				errSlotIsTooHigh, start, highestFinalizedSlot)
			return response
		}
	}

	response.blocks, response.pid, response.err = f.fetchBlocksFromPeer(ctx, start, count, peers)
	if response.err == nil {
		if err := f.fetchBlobsForResponse(ctx, response, peers); err != nil {
			response.err = err
		}
	}
	return response
}

// fetchBlocksFromPeer fetches blocks from a single randomly selected peer.
func (f *blocksFetcher) fetchBlocksFromPeer(
	ctx context.Context,
	start primitives.Slot, count uint64,
	peers []peer.ID,
) ([]interfaces.ReadOnlySignedBeaconBlock, peer.ID, error) {
	ctx, span := trace.StartSpan(ctx, "initialsync.fetchBlocksFromPeer")
	defer span.End()

	peers = f.filterPeers(ctx, peers, peersPercentagePerRequest)
	req := &p2ppb.BeaconBlocksByRangeRequest{
		StartSlot: start,
		Count:     count,
		Step:      1,
	}
	for i := 0; i < len(peers); i++ {
		p := peers[i]
		blocks, err := f.requestBlocks(ctx, req, p)
		if err != nil {
			log.WithField("peer", p).WithError(err).Debug("Could not request blocks by range from peer")
			continue
		}
		f.p2p.Peers().Scorers().BlockProviderScorer().Touch(p)
		return blocks, p, err
	}
	return nil, "", errNoPeersAvailable
}

func blobRequestsForBlocks(blocks []interfaces.ReadOnlySignedBeaconBlock) (*p2ppb.BlobSidecarsByRangeRequest, error) {
	log.WithField("block-len", len(blocks)).Warn("blobRequestsForBlocks 1")
	if len(blocks) == 0 {
		return nil, nil
	}

	req := &p2ppb.BlobSidecarsByRangeRequest{
		StartSlot: 0,
		Count:     0,
	}

	// Loop through the blocks, looking for the earliest deneb block in the slice.
	for i, bl := range blocks {
		commits, err := bl.Block().Body().BlobKzgCommitments()
		if err != nil {
			if errors.Is(err, consensus_types.ErrUnsupportedGetter) {
				continue
			}
			return nil, err
		}
		if len(commits) == 0 {
			continue
		}
		if req.Count == 0 {
			req.StartSlot = bl.Block().Slot()
		}
		req.Count = uint64(len(blocks) - i)
		break
	}
	if req.Count == 0 {
		return nil, nil
	}
	return req, nil
}

// fetchBlobsFromPeer fetches blocks from a single randomly selected peer.
func (f *blocksFetcher) fetchBlobsForResponse(ctx context.Context, resp *fetchRequestResponse, peers []peer.ID) error {
	ctx, span := trace.StartSpan(ctx, "initialsync.fetchBlobsFromPeer")
	defer span.End()

	req, err := blobRequestsForBlocks(resp.blocks)
	if err != nil || req == nil {
		return err
	}
	peers = f.filterPeers(ctx, peers, peersPercentagePerRequest)
	// prioritize the peer we got the block slice from, since they are required to have the blobs
	for i := range peers {
		if peers[i] == resp.pid {
			if i == 0 {
				break
			}
			// if the peer we got the blocks from isn't first on the list, swap it to the front
			peers[0], peers[i] = peers[i], peers[0]
		}
	}
	for i := 0; i < len(peers); i++ {
		pid := peers[i]
		blobs, err := f.requestBlobs(ctx, req, pid)
		if err != nil {
			log.WithError(err).Debug("Could not request blobs by range")
			continue
		}
		f.p2p.Peers().Scorers().BlockProviderScorer().Touch(pid)
		if resp.blobMap == nil {
			resp.blobMap = make(map[[32]byte][]*p2ppb.BlobSidecar)
		}
		for _, b := range blobs {
			resp.blobMap[bytesutil.ToBytes32(b.BlockRoot)] = append(resp.blobMap[bytesutil.ToBytes32(b.BlockRoot)], b)
		}
		resp.blobsPid = pid
		return nil
	}
	return errNoPeersAvailable
}

// requestBlocks is a wrapper for handling BeaconBlocksByRangeRequest requests/streams.
func (f *blocksFetcher) requestBlocks(
	ctx context.Context,
	req *p2ppb.BeaconBlocksByRangeRequest,
	pid peer.ID,
) ([]interfaces.ReadOnlySignedBeaconBlock, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	l := f.peerLock(pid)
	l.Lock()
	log.WithFields(logrus.Fields{
		"peer":     pid,
		"start":    req.StartSlot,
		"count":    req.Count,
		"step":     req.Step,
		"capacity": f.rateLimiter.Remaining(pid.String()),
		"score":    f.p2p.Peers().Scorers().BlockProviderScorer().FormatScorePretty(pid),
	}).Debug("Requesting blocks")
	if f.rateLimiter.Remaining(pid.String()) < int64(req.Count) {
		if err := f.waitForBandwidth(pid, req.Count); err != nil {
			l.Unlock()
			return nil, err
		}
	}
	f.rateLimiter.Add(pid.String(), int64(req.Count))
	l.Unlock()
	return prysmsync.SendBeaconBlocksByRangeRequest(ctx, f.chain, f.p2p, pid, req, nil)
}

func (f *blocksFetcher) requestBlobs(ctx context.Context, req *p2ppb.BlobSidecarsByRangeRequest, pid peer.ID) ([]*p2ppb.BlobSidecar, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	l := f.peerLock(pid)
	l.Lock()
	log.WithFields(logrus.Fields{
		"peer":     pid,
		"start":    req.StartSlot,
		"count":    req.Count,
		"capacity": f.rateLimiter.Remaining(pid.String()),
		"score":    f.p2p.Peers().Scorers().BlockProviderScorer().FormatScorePretty(pid),
	}).Debug("Requesting blobs")
	if f.rateLimiter.Remaining(pid.String()) < int64(req.Count) {
		if err := f.waitForBandwidth(pid, req.Count); err != nil {
			l.Unlock()
			return nil, err
		}
	}
	f.rateLimiter.Add(pid.String(), int64(req.Count))
	l.Unlock()
	return prysmsync.SendBlobsByRangeRequest(ctx, f.clock, f.p2p, pid, f.ctxMap, req)
}

// requestBlocksByRoot is a wrapper for handling BeaconBlockByRootsReq requests/streams.
func (f *blocksFetcher) requestBlocksByRoot(
	ctx context.Context,
	req *p2pTypes.BeaconBlockByRootsReq,
	pid peer.ID,
) ([]interfaces.ReadOnlySignedBeaconBlock, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	l := f.peerLock(pid)
	l.Lock()
	log.WithFields(logrus.Fields{
		"peer":     pid,
		"numRoots": len(*req),
		"capacity": f.rateLimiter.Remaining(pid.String()),
		"score":    f.p2p.Peers().Scorers().BlockProviderScorer().FormatScorePretty(pid),
	}).Debug("Requesting blocks (by roots)")
	if f.rateLimiter.Remaining(pid.String()) < int64(len(*req)) {
		if err := f.waitForBandwidth(pid, uint64(len(*req))); err != nil {
			l.Unlock()
			return nil, err
		}
	}
	f.rateLimiter.Add(pid.String(), int64(len(*req)))
	l.Unlock()

	return prysmsync.SendBeaconBlocksByRootRequest(ctx, f.chain, f.p2p, pid, req, nil)
}

// waitForBandwidth blocks up until peer's bandwidth is restored.
func (f *blocksFetcher) waitForBandwidth(pid peer.ID, count uint64) error {
	log.WithField("peer", pid).Debug("Slowing down for rate limit")
	rem := f.rateLimiter.Remaining(pid.String())
	if uint64(rem) >= count {
		// Exit early if we have sufficient capacity
		return nil
	}
	intCount, err := math.Int(count)
	if err != nil {
		return err
	}
	toWait := timeToWait(int64(intCount), rem, f.rateLimiter.Capacity(), f.rateLimiter.TillEmpty(pid.String()))
	timer := time.NewTimer(toWait)
	defer timer.Stop()
	select {
	case <-f.ctx.Done():
		return errFetcherCtxIsDone
	case <-timer.C:
		// Peer has gathered enough capacity to be polled again.
	}
	return nil
}

// Determine how long it will take for us to have the required number of blocks allowed by our rate limiter.
// We do this by calculating the duration till the rate limiter can request these blocks without exceeding
// the provided bandwidth limits per peer.
func timeToWait(wanted, rem, capacity int64, timeTillEmpty time.Duration) time.Duration {
	// Defensive check if we have more than enough blocks
	// to request from the peer.
	if rem >= wanted {
		return 0
	}
	// Handle edge case where capacity is equal to the remaining amount
	// of blocks. This also handles the impossible case in where remaining blocks
	// exceed the limiter's capacity.
	if capacity <= rem {
		return 0
	}
	blocksNeeded := wanted - rem
	currentNumBlks := capacity - rem
	expectedTime := int64(timeTillEmpty) * blocksNeeded / currentNumBlks
	return time.Duration(expectedTime)
}
