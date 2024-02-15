package backfill

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/testing/require"
)

func TestSortBatchDesc(t *testing.T) {
	orderIn := []primitives.Slot{100, 10000, 1}
	orderOut := []primitives.Slot{10000, 100, 1}
	batches := make([]batch, len(orderIn))
	for i := range orderIn {
		batches[i] = batch{end: orderIn[i]}
	}
	sortBatchDesc(batches)
	for i := range orderOut {
		require.Equal(t, orderOut[i], batches[i].end)
	}
}
