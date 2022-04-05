package store

import (
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestNew(t *testing.T) {
	j := &ethpb.Checkpoint{
		Epoch: 0,
		Root:  []byte("hi"),
	}
	f := &ethpb.Checkpoint{
		Epoch: 0,
		Root:  []byte("hello"),
	}
	s := New(j, f)
	require.DeepSSZEqual(t, s.JustifiedCheckpt(), j)
	require.DeepSSZEqual(t, s.BestJustifiedCheckpt(), j)
	require.DeepSSZEqual(t, s.PrevJustifiedCheckpt(), j)
	require.DeepSSZEqual(t, s.FinalizedCheckpt(), f)
	require.DeepSSZEqual(t, s.PrevFinalizedCheckpt(), f)
}
