package store

import (
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func Test_store_PrevJustifiedCheckpt(t *testing.T) {
	s := &Store{}
	var cp *ethpb.Checkpoint
	_, err := s.PrevJustifiedCheckpt()
	require.ErrorIs(t, ErrNilCheckpoint, err)
	cp = &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}}
	s.SetPrevJustifiedCheckpt(cp)
	got, err := s.PrevJustifiedCheckpt()
	require.NoError(t, err)
	require.Equal(t, cp, got)
}

func Test_store_BestJustifiedCheckpt(t *testing.T) {
	s := &Store{}
	var cp *ethpb.Checkpoint
	_, err := s.BestJustifiedCheckpt()
	require.ErrorIs(t, ErrNilCheckpoint, err)
	cp = &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}}
	s.SetBestJustifiedCheckpt(cp)
	got, err := s.BestJustifiedCheckpt()
	require.NoError(t, err)
	require.Equal(t, cp, got)
}

func Test_store_JustifiedCheckpt(t *testing.T) {
	s := &Store{}
	var cp *ethpb.Checkpoint
	_, err := s.JustifiedCheckpt()
	require.ErrorIs(t, ErrNilCheckpoint, err)
	cp = &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}}
	h := [32]byte{'b'}
	s.SetJustifiedCheckptAndPayloadHash(cp, h)
	got, err := s.JustifiedCheckpt()
	require.NoError(t, err)
	require.Equal(t, cp, got)
	require.Equal(t, h, s.JustifiedPayloadBlockHash())
}

func Test_store_FinalizedCheckpt(t *testing.T) {
	s := &Store{}
	var cp *ethpb.Checkpoint
	_, err := s.FinalizedCheckpt()
	require.ErrorIs(t, ErrNilCheckpoint, err)
	cp = &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}}
	h := [32]byte{'b'}
	s.SetFinalizedCheckptAndPayloadHash(cp, h)
	got, err := s.FinalizedCheckpt()
	require.NoError(t, err)
	require.Equal(t, cp, got)
	require.Equal(t, h, s.FinalizedPayloadBlockHash())
}

func Test_store_PrevFinalizedCheckpt(t *testing.T) {
	s := &Store{}
	var cp *ethpb.Checkpoint
	_, err := s.PrevFinalizedCheckpt()
	require.ErrorIs(t, ErrNilCheckpoint, err)
	cp = &ethpb.Checkpoint{Epoch: 1, Root: []byte{'a'}}
	s.SetPrevFinalizedCheckpt(cp)
	got, err := s.PrevFinalizedCheckpt()
	require.NoError(t, err)
	require.Equal(t, cp, got)
}
