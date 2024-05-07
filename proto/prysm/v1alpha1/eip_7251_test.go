package eth_test

import (
	"testing"

	v1alpha1 "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v5/testing/assert"
)

func TestConsolidation_ToPendingConsolidation(t *testing.T) {
	c := v1alpha1.Consolidation{
		SourceIndex: 1,
		TargetIndex: 2,
		Epoch:       3,
	}

	pc := c.ToPendingConsolidation()

	assert.Equal(t, c.SourceIndex, pc.SourceIndex)
	assert.Equal(t, c.TargetIndex, pc.TargetIndex)
}
