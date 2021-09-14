package kv

import (
	"context"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestChainHead(t *testing.T) {
	db := setupDB(t)
	ctx := context.Background()

	tests := []struct {
		head *ethpb.ChainHead
	}{
		{
			head: &ethpb.ChainHead{
				HeadSlot:       20,
				HeadEpoch:      20,
				FinalizedSlot:  10,
				FinalizedEpoch: 10,
				JustifiedSlot:  10,
				JustifiedEpoch: 10,
			},
		},
		{
			head: &ethpb.ChainHead{
				HeadSlot: 1,
			},
		},
		{
			head: &ethpb.ChainHead{
				HeadBlockRoot: make([]byte, 32),
			},
		},
	}

	for _, tt := range tests {
		require.NoError(t, db.SaveChainHead(ctx, tt.head))
		head, err := db.ChainHead(ctx)
		require.NoError(t, err, "Failed to get block")
		assert.NotNil(t, head)
		assert.DeepEqual(t, tt.head, head)
	}
}
