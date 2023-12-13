package cache_test

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	"github.com/stretchr/testify/require"
)

func TestAttestationCache_RoundTrip(t *testing.T) {
	ctx := context.Background()
	c := cache.NewAttestationCache()

	a, err := c.Get(ctx)
	require.NoError(t, err)
	require.Nil(t, a)

	insert := &cache.AttestationConsensusData{
		Slot:        1,
		HeadRoot:    []byte{1},
		TargetRoot:  []byte{2},
		TargetEpoch: 3,
		SourceRoot:  []byte{4},
		SourceEpoch: 5,
	}
	err = c.Put(ctx, insert)
	require.NoError(t, err)

	a, err = c.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, insert, a)

	insert = &cache.AttestationConsensusData{
		Slot:        6,
		HeadRoot:    []byte{7},
		TargetRoot:  []byte{8},
		TargetEpoch: 9,
		SourceRoot:  []byte{10},
		SourceEpoch: 11,
	}

	err = c.Put(ctx, insert)
	require.NoError(t, err)

	a, err = c.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, insert, a)
}
