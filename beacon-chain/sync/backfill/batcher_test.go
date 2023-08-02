package backfill

import (
	"testing"

	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
)

func TestBatcherBefore(t *testing.T) {
	cases := []struct {
		name   string
		b      batcher
		upTo   primitives.Slot
		expect batch
	}{
		{
			b:      batcher{min: 0, size: 10},
			upTo:   33,
			expect: batch{begin: 23, end: 33, state: batchInit},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.b, c.b.before(c.upTo))
		})
	}
}
