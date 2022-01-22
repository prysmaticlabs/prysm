package store

import (
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func Test_store_PrevJustifiedCheckpt(t *testing.T) {
	s := &store{}
	var cp *ethpb.Checkpoint
	require.Equal(t, cp, s.PrevJustifiedCheckpt())
	cp = &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}}
	s.SetPrevJustifiedCheckpt(cp)
	require.Equal(t, cp, s.PrevJustifiedCheckpt())
}

func Test_store_BestJustifiedCheckpt(t *testing.T) {
	s := &store{}
	var cp *ethpb.Checkpoint
	require.Equal(t, cp, s.BestJustifiedCheckpt())
	cp = &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}}
	s.SetBestJustifiedCheckpt(cp)
	require.Equal(t, cp, s.BestJustifiedCheckpt())
}

func Test_store_JustifiedCheckpt(t *testing.T) {
	s := &store{}
	var cp *ethpb.Checkpoint
	require.Equal(t, cp, s.JustifiedCheckpt())
	cp = &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}}
	s.SetJustifiedCheckpt(cp)
	require.Equal(t, cp, s.JustifiedCheckpt())
}

func Test_store_FinalizedCheckpt(t *testing.T) {
	s := &store{}
	var cp *ethpb.Checkpoint
	require.Equal(t, cp, s.FinalizedCheckpt())
	cp = &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}}
	s.SetFinalizedCheckpt(cp)
	require.Equal(t, cp, s.FinalizedCheckpt())
}
