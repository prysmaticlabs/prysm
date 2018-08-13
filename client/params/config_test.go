package params

import (
	"math"
	"testing"
)

func TestAttesterCommitteeSize(t *testing.T) {
	c := DefaultConfig()
	if c.CollationSizeLimit != int64(math.Pow(float64(2), float64(20))) {
		t.Errorf("Shard count incorrect. Wanted %d, got %d", int64(math.Pow(float64(2), float64(20))), c.CollationSizeLimit)
	}
}
