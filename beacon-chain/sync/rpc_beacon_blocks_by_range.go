package sync

import (
	"context"
	"sort"
	"time"

	libp2pcore "github.com/libp2p/go-libp2p/core"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/db/filters"
	p2ptypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/v3/cmd/beacon-chain/flags"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/monitoring/tracing"
	pb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// beaconBlocksByRangeRPCHandler looks up the request blocks from the database from a given start block.
func (s *Service) beaconBlocksByRangeRPCHandler(ctx context.Context, msg interface{}, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.BeaconBlocksByRangeHandler")
	defer span.End()
	ctx, cancel := context.WithTimeout(ctx, respTimeout)
	defer cancel()
	SetRPCStreamDeadlines(stream)

	m, ok := msg.(*pb.BeaconBlocksByRangeRequest)
	if !ok {
		return errors.New("message is not type *pb.BeaconBlockByRangeRequest")
	}
	start, end, size, err := validateRangeRequest(m, s.cfg.chain.CurrentSlot())
	if err != nil {
		s.writeErrorResponseToStream(responseCodeInvalidRequest, err.Error(), stream)
		s.cfg.p2p.Peers().Scorers().BadResponsesScorer().Increment(stream.Conn().RemotePeer())
		tracing.AnnotateError(span, err)
		return err
	}

	blockLimiter, err := s.rateLimiter.topicCollector(string(stream.Protocol()))
	if err != nil {
		return err
	}
	remainingBucketCapacity := blockLimiter.Remaining(stream.Conn().RemotePeer().String())
	span.AddAttributes(
		trace.Int64Attribute("start", int64(start)), // lint:ignore uintcast -- This conversion is OK for tracing.
		trace.Int64Attribute("end", int64(end)),     // lint:ignore uintcast -- This conversion is OK for tracing.
		trace.Int64Attribute("count", int64(m.Count)),
		trace.StringAttribute("peer", stream.Conn().RemotePeer().Pretty()),
		trace.Int64Attribute("remaining_capacity", remainingBucketCapacity),
	)

	// Ticker to stagger out large requests.
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	batcher := &blockRangeBatcher{
		start:       start,
		end:         end,
		size:        size,
		db:          s.cfg.beaconDB,
		limiter:     s.rateLimiter,
		isCanonical: s.cfg.chain.IsCanonical,
		ticker:      ticker,
	}

	// prevRoot is used to ensure that returned chains are strictly linear for singular steps
	// by comparing the previous root of the block in the list with the current block's parent.
	var batch blockBatch
	for batch, ok = batcher.Next(ctx, stream); ok; batch, ok = batcher.Next(ctx, stream) {
		batchStart := time.Now()
		rpcBlocksByRangeResponseLatency.Observe(float64(time.Since(batchStart).Milliseconds()))
		if err := s.writeBlockBatchToStream(ctx, batch, stream); err != nil {
			s.writeErrorResponseToStream(responseCodeServerError, p2ptypes.ErrGeneric.Error(), stream)
			return err
		}
	}
	if err := batch.Err(); err != nil {
		log.WithError(err).Debug("error in BlocksByRange batch")
		s.writeErrorResponseToStream(responseCodeServerError, p2ptypes.ErrGeneric.Error(), stream)
		tracing.AnnotateError(span, err)
		return err
	}
	closeStream(stream, log)
	return nil
}

func validateRangeRequest(r *pb.BeaconBlocksByRangeRequest, current primitives.Slot) (primitives.Slot, primitives.Slot, uint64, error) {
	start := r.StartSlot
	count := r.Count
	maxRequest := params.BeaconNetworkConfig().MaxRequestBlocks
	// Ensure all request params are within appropriate bounds
	if count == 0 || count > maxRequest {
		return 0, 0, 0, p2ptypes.ErrInvalidRequest
	}
	// Allow some wiggle room, up to double the MaxRequestBlocks past the current slot,
	// to give nodes syncing close to the head of the chain some margin for error.
	maxStart, err := current.SafeAdd(maxRequest * 2)
	if err != nil {
		return 0, 0, 0, p2ptypes.ErrInvalidRequest
	}
	if start > maxStart {
		return 0, 0, 0, p2ptypes.ErrInvalidRequest
	}
	end, err := start.SafeAdd((count - 1))
	if err != nil {
		return 0, 0, 0, p2ptypes.ErrInvalidRequest
	}

	limit := uint64(flags.Get().BlockBatchLimit)
	if limit > maxRequest {
		limit = maxRequest
	}
	batchSize := count
	if batchSize > limit {
		batchSize = limit
	}

	return start, end, batchSize, nil
}

func (s *Service) writeBlockBatchToStream(ctx context.Context, batch blockBatch, stream libp2pcore.Stream) error {
	ctx, span := trace.StartSpan(ctx, "sync.WriteBlockRangeToStream")
	defer span.End()

	blks := batch.Sequence()
	// If the blocks are blinded, we reconstruct the full block via the execution client.
	blindedExists := false
	blindedIndex := 0
	for i, b := range blks {
		// Since the blocks are sorted in ascending order, we assume that the following
		// blocks from the first blinded block are also ascending.
		if b.IsBlinded() {
			blindedExists = true
			blindedIndex = i
			break
		}
	}

	for _, b := range blks {
		if err := blocks.BeaconBlockIsNil(b); err != nil {
			continue
		}
		if b.IsBlinded() {
			continue
		}
		if chunkErr := s.chunkBlockWriter(stream, b); chunkErr != nil {
			log.WithError(chunkErr).Debug("Could not send a chunked response")
			return chunkErr
		}
	}

	var err error
	var reconstructedBlock []interfaces.SignedBeaconBlock
	if blindedExists {
		blinded := blks[blindedIndex:]
		unwrapped := make([]interfaces.ReadOnlySignedBeaconBlock, len(blinded))
		for i := range blinded {
			unwrapped[i] = blks[i].ReadOnlySignedBeaconBlock
		}
		reconstructedBlock, err = s.cfg.executionPayloadReconstructor.ReconstructFullBellatrixBlockBatch(ctx, unwrapped)
		if err != nil {
			log.WithError(err).Error("Could not reconstruct full bellatrix block batch from blinded bodies")
			return err
		}
	}
	for _, b := range reconstructedBlock {
		if err := blocks.BeaconBlockIsNil(b); err != nil {
			continue
		}
		if b.IsBlinded() {
			continue
		}
		if chunkErr := s.chunkBlockWriter(stream, b); chunkErr != nil {
			log.WithError(chunkErr).Debug("Could not send a chunked response")
			return chunkErr
		}
	}

	return nil
}

type canonicalChecker func(context.Context, [32]byte) (bool, error)

// filters all the provided blocks to ensure they are canonical
// and are strictly linear.
func filterCanonical(ctx context.Context, blks []blocks.ROBlock, prevRoot *[32]byte, canonical canonicalChecker) ([]blocks.ROBlock, []blocks.ROBlock, error) {
	seq := make([]blocks.ROBlock, 0, len(blks))
	nseq := make([]blocks.ROBlock, 0)
	for i, b := range blks {
		cb, err := canonical(ctx, b.Root())
		if err != nil {
			return nil, nil, err
		}
		if !cb {
			continue
		}
		// filterCanonical is called in batches, so prevRoot can be the last root from the previous batch.
		// prevRoot will be the zero value until we find the first canonical block in a given request.
		first := *prevRoot == [32]byte{}
		// We assume blocks are processed in order, so the previous canonical root should be the parent of the next.
		// If the current block isn't descended from the last, something is wrong. Append everything remaining
		// to the list of non-sequential blocks and stop building the canonical list.
		if !first && *prevRoot != b.Block().ParentRoot() {
			nseq = append(nseq, blks[i:]...)
			break
		}
		seq = append(seq, blks[i])
		// Set the previous root as the
		// newly added block's root
		currRoot := b.Root()
		*prevRoot = currRoot
	}
	return seq, nseq, nil
}

// returns a copy of the []ROBlock list in sorted order with duplicates removed
func sortedUniqueBlocks(blks []blocks.ROBlock) []blocks.ROBlock {
	// Remove duplicate blocks received
	sort.Sort(blocks.ROBlockSlice(blks))
	u := 0
	for i := 1; i < len(blks); i++ {
		if blks[i].Root() != blks[u].Root() {
			u += 1
			if u != i {
				blks[u] = blks[i]
			}
		}
	}
	return blks[:u+1]
}

type blockBatch struct {
	start  primitives.Slot
	end    primitives.Slot
	seq    []blocks.ROBlock
	nonseq []blocks.ROBlock
	err    error
}

func (bb blockBatch) RateLimitCost() int {
	return int(bb.end - bb.start)
}

func (bb blockBatch) Sequence() []blocks.ROBlock {
	return bb.seq
}

func (bb blockBatch) SequenceBroken() bool {
	return len(bb.nonseq) > 0
}

func (bb blockBatch) Err() error {
	return bb.err
}

type blockRangeBatcher struct {
	start       primitives.Slot
	end         primitives.Slot
	size        uint64
	db          db.NoHeadAccessDatabase
	limiter     *limiter
	isCanonical canonicalChecker
	ticker      *time.Ticker

	lastSeq [32]byte
	current *blockBatch
}

func (bb *blockRangeBatcher) genesisBlock(ctx context.Context) (blocks.ROBlock, error) {
	b, err := bb.db.GenesisBlock(ctx)
	if err != nil {
		return blocks.ROBlock{}, err
	}
	htr, err := b.Block().HashTreeRoot()
	if err != nil {
		return blocks.ROBlock{}, err
	}
	return blocks.NewROBlock(b, htr), nil
}

func newBlockBatch(start, reqEnd primitives.Slot, size uint64) (blockBatch, bool) {
	if start > reqEnd {
		return blockBatch{}, false
	}
	nb := blockBatch{start: start, end: start.Add(size - 1)}
	if nb.end > reqEnd {
		nb.end = reqEnd
	}
	return nb, true
}

func (bat blockBatch) Next(reqEnd primitives.Slot, size uint64) (blockBatch, bool) {
	if bat.SequenceBroken() {
		return blockBatch{}, false
	}
	return newBlockBatch(bat.end.Add(1), reqEnd, size)
}

func (bb *blockRangeBatcher) Next(ctx context.Context, stream libp2pcore.Stream) (blockBatch, bool) {
	var nb blockBatch
	var ok bool
	if bb.current != nil {
		current := *bb.current
		nb, ok = current.Next(bb.end, bb.size)
	} else {
		nb, ok = newBlockBatch(bb.start, bb.end, bb.size)
	}
	if !ok {
		return blockBatch{}, false
	}
	if err := bb.limiter.validateRequest(stream, bb.size); err != nil {
		return blockBatch{err: errors.Wrap(err, "throttled by rate limiter")}, false
	}

	// block if there is work to do, unless this is the first batch
	if bb.ticker != nil && bb.current != nil {
		<-bb.ticker.C
	}
	filter := filters.NewFilter().SetStartSlot(nb.start).SetEndSlot(nb.end)
	blks, roots, err := bb.db.Blocks(ctx, filter)
	if err != nil {
		return blockBatch{err: errors.Wrap(err, "Could not retrieve blocks")}, false
	}

	// make slice with extra +1 capacity in case we want to grow it to also hold the genesis block
	rob := make([]blocks.ROBlock, len(blks), len(blks)+1)
	goff := 0 // offset for genesis value
	if nb.start == 0 {
		gb, err := bb.genesisBlock(ctx)
		if err != nil {
			return blockBatch{err: errors.Wrap(err, "could not retrieve genesis block")}, false
		}
		rob = append(rob, blocks.ROBlock{}) // grow the slice to its capacity to hold the genesis block
		rob[0] = gb
		goff = 1
	}
	for i := 0; i < len(blks); i++ {
		rob[goff+i] = blocks.NewROBlock(blks[i], roots[i])
	}
	// Filter and sort our retrieved blocks, so that
	// we only return valid sets of blocks.
	rob = sortedUniqueBlocks(rob)

	nb.seq, nb.nonseq, nb.err = filterCanonical(ctx, rob, &bb.lastSeq, bb.isCanonical)

	// Decrease allowed blocks capacity by the number of streamed blocks.
	bb.limiter.add(stream, int64(1+nb.end.SubSlot(nb.start)))
	bb.current = &nb
	return *bb.current, true
}
