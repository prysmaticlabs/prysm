package backfill

import (
	"fmt"
	"time"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	log "github.com/sirupsen/logrus"
)

type batchId string

type batch struct {
	scheduled time.Time
	retries   int
	begin     primitives.Slot
	end       primitives.Slot // half-open interval, [begin, end), ie >= start, < end.
	results   []blocks.ROBlock
	err       error
	succeeded bool
}

func (b batch) logFields() log.Fields {
	return map[string]interface{}{
		"batch_id":  b.id(),
		"scheduled": b.scheduled.String(),
		"retries":   b.retries,
	}
}

func (b batch) id() batchId {
	return batchId(fmt.Sprintf("%d:%d", b.begin, b.end))
}
