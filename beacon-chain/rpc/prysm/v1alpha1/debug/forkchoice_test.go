package debug

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	mock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/doubly-linked-tree"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/types"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestServer_GetForkChoice(t *testing.T) {
	store := doublylinkedtree.New()
	fRoot := [32]byte{'a'}
	jRoot := [32]byte{'b'}
	fc := &forkchoicetypes.Checkpoint{Epoch: 2, Root: fRoot}
	jc := &forkchoicetypes.Checkpoint{Epoch: 3, Root: jRoot}
	require.NoError(t, store.UpdateFinalizedCheckpoint(fc))
	require.NoError(t, store.UpdateJustifiedCheckpoint(jc))
	bs := &Server{ForkFetcher: &mock.ChainService{ForkChoiceStore: store}}
	res, err := bs.GetForkChoice(context.Background(), &empty.Empty{})
	require.NoError(t, err)
	require.Equal(t, types.Epoch(3), res.JustifiedCheckpoint.Epoch, "Did not get wanted justified epoch")
	require.Equal(t, types.Epoch(2), res.FinalizedCheckpoint.Epoch, "Did not get wanted finalized epoch")
}
