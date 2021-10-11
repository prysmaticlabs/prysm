package main

import (
	"fmt"
	"testing"

	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/require"
)

func TestProveCheckpoint(t *testing.T) {
	root := [32]byte{1}
	check := &ethpb.Checkpoint{
		Epoch: 1,
		Root:  root[:],
	}
	tr, err := check.GetTree()
	require.NoError(t, err)
	a, err := tr.Get(0)
	require.NoError(t, err)
	b, err := tr.Get(1)
	require.NoError(t, err)
	fmt.Println(a.Hash())
	fmt.Println(b.Hash())
}
