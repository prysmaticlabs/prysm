package backfill

import (
	"fmt"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/das"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/sync"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
)

// ErrChainBroken indicates a backfill batch can't be imported to the db because it is not known to be the ancestor
// of the canonical chain.
var ErrChainBroken = errors.New("batch is not the ancestor of a known finalized root")

type batchState int

func (s batchState) String() string {
	switch s {
	case batchNil:
		return "nil"
	case batchInit:
		return "init"
	case batchSequenced:
		return "sequenced"
	case batchErrRetryable:
		return "error_retryable"
	case batchImportable:
		return "importable"
	case batchImportComplete:
		return "import_complete"
	case batchEndSequence:
		return "end_sequence"
	case batchBlobSync:
		return "blob_sync"
	default:
		return "unknown"
	}
}

const (
	batchNil batchState = iota
	batchInit
	batchSequenced
	batchErrRetryable
	batchBlobSync
	batchImportable
	batchImportComplete
	batchEndSequence
)

type batchId string

type batch struct {
	firstScheduled time.Time
	scheduled      time.Time
	seq            int // sequence identifier, ie how many times has the sequence() method served this batch
	retries        int
	begin          primitives.Slot
	end            primitives.Slot // half-open interval, [begin, end), ie >= start, < end.
	results        verifiedROBlocks
	err            error
	state          batchState
	busy           peer.ID
	blockPid       peer.ID
	blobPid        peer.ID
	bs             *blobSync
}

func (b batch) logFields() log.Fields {
	return map[string]interface{}{
		"batch_id":  b.id(),
		"state":     b.state.String(),
		"scheduled": b.scheduled.String(),
		"seq":       b.seq,
		"retries":   b.retries,
		"begin":     b.begin,
		"end":       b.end,
		"busyPid":   b.busy,
		"blockPid":  b.blockPid,
		"blobPid":   b.blobPid,
	}
}

func (b batch) replaces(r batch) bool {
	if r.state == batchImportComplete {
		return false
	}
	if b.begin != r.begin {
		return false
	}
	if b.end != r.end {
		return false
	}
	return b.seq >= r.seq
}

func (b batch) id() batchId {
	return batchId(fmt.Sprintf("%d:%d", b.begin, b.end))
}

func (b batch) ensureParent(expected [32]byte) error {
	tail := b.results[len(b.results)-1]
	if tail.Root() != expected {
		return errors.Wrapf(ErrChainBroken, "last parent_root=%#x, tail root=%#x", expected, tail.Root())
	}
	return nil
}

func (b batch) blockRequest() *eth.BeaconBlocksByRangeRequest {
	return &eth.BeaconBlocksByRangeRequest{
		StartSlot: b.begin,
		Count:     uint64(b.end - b.begin),
		Step:      1,
	}
}

func (b batch) blobRequest() *eth.BlobSidecarsByRangeRequest {
	return &eth.BlobSidecarsByRangeRequest{
		StartSlot: b.begin,
		Count:     uint64(b.end - b.begin),
	}
}

func (b batch) withResults(results verifiedROBlocks, bs *blobSync) batch {
	b.results = results
	b.bs = bs
	if bs.blobsNeeded() > 0 {
		return b.withState(batchBlobSync)
	}
	return b.withState(batchImportable)
}

func (b batch) postBlobSync() batch {
	if b.blobsNeeded() > 0 {
		log.WithFields(b.logFields()).WithField("blobs_missing", b.blobsNeeded()).Error("batch still missing blobs after downloading from peer")
		b.bs = nil
		b.results = []blocks.ROBlock{}
		return b.withState(batchErrRetryable)
	}
	return b.withState(batchImportable)
}

func (b batch) withState(s batchState) batch {
	if s == batchSequenced {
		b.scheduled = time.Now()
		switch b.state {
		case batchErrRetryable:
			b.retries += 1
			log.WithFields(b.logFields()).Info("sequencing batch for retry")
		case batchInit, batchNil:
			b.firstScheduled = b.scheduled
		}
	}
	if s == batchImportComplete {
		backfillBatchTimeRoundtrip.Observe(float64(time.Since(b.firstScheduled).Milliseconds()))
		log.WithFields(b.logFields()).Debug("Backfill batch imported.")
	}
	b.state = s
	b.seq += 1
	return b
}

func (b batch) withPeer(p peer.ID) batch {
	b.blockPid = p
	backfillBatchTimeWaiting.Observe(float64(time.Since(b.scheduled).Milliseconds()))
	return b
}

func (b batch) withRetryableError(err error) batch {
	b.err = err
	return b.withState(batchErrRetryable)
}

func (b batch) blobsNeeded() int {
	return b.bs.blobsNeeded()
}

func (b batch) blobResponseValidator() sync.BlobResponseValidation {
	return b.bs.validateNext
}

func (b batch) availabilityStore() das.AvailabilityStore {
	return b.bs.store
}

func sortBatchDesc(bb []batch) {
	sort.Slice(bb, func(i, j int) bool {
		return bb[j].end < bb[i].end
	})
}
