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
	cp, err := s.JustifiedCheckpt()
	require.NoError(t, err)
	require.DeepSSZEqual(t, j, cp)
	cp, err = s.BestJustifiedCheckpt()
	require.NoError(t, err)
	require.DeepSSZEqual(t, j, cp)
	cp, err = s.PrevJustifiedCheckpt()
	require.NoError(t, err)
	require.DeepSSZEqual(t, j, cp)
	cp, err = s.FinalizedCheckpt()
	require.NoError(t, err)
	require.DeepSSZEqual(t, f, cp)
	cp, err = s.PrevFinalizedCheckpt()
	require.NoError(t, err)
	require.DeepSSZEqual(t, f, cp)
}
