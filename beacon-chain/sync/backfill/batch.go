package backfill

import (
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	log "github.com/sirupsen/logrus"
)

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
	case batchErrFatal:
		return "error_fatal"
	case batchImportable:
		return "importable"
	case batchImportComplete:
		return "import_complete"
	case batchEndSequence:
		return "end_sequence"
	default:
		return "unknown"
	}
}

const (
	batchNil batchState = iota
	batchInit
	batchSequenced
	batchErrRetryable
	batchErrFatal
	batchImportable
	batchImportComplete
	batchEndSequence
)

type batchId string

type batch struct {
	scheduled time.Time
	seq       int // sequence identifier, ie how many times has the sequence() method served this batch
	retries   int
	begin     primitives.Slot
	end       primitives.Slot // half-open interval, [begin, end), ie >= start, < end.
	results   []blocks.ROBlock
	err       error
	state     batchState
}

func (b batch) logFields() log.Fields {
	return map[string]interface{}{
		"batch_id":    b.id(),
		"batch_state": b.state.String(),
		"scheduled":   b.scheduled.String(),
		"retries":     b.retries,
	}
}

func (b *batch) inc() {
	b.seq += 1
}

func (b batch) replaces(r batch) bool {
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

func (b batch) size() primitives.Slot {
	return b.end - b.begin
}
