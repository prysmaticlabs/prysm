package debug

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	blockchainmock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/doubly-linked-tree"
	forkchoicetypes "github.com/prysmaticlabs/prysm/v4/beacon-chain/forkchoice/types"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestListForkChoiceHeadsV2(t *testing.T) {
	ctx := context.Background()

	expectedSlotsAndRoots := []struct {
		Slot primitives.Slot
		Root [32]byte
	}{{
		Slot: 0,
		Root: bytesutil.ToBytes32(bytesutil.PadTo([]byte("foo"), 32)),
	}, {
		Slot: 1,
		Root: bytesutil.ToBytes32(bytesutil.PadTo([]byte("bar"), 32)),
	}}

	chainService := &blockchainmock.ChainService{}
	server := &Server{
		HeadFetcher:           chainService,
		OptimisticModeFetcher: chainService,
	}
	resp, err := server.ListForkChoiceHeadsV2(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(resp.Data))
	for _, sr := range expectedSlotsAndRoots {
		found := false
		for _, h := range resp.Data {
			if h.Slot == sr.Slot {
				found = true
				assert.DeepEqual(t, sr.Root[:], h.Root)
			}
			assert.Equal(t, false, h.ExecutionOptimistic)
		}
		assert.Equal(t, true, found, "Expected head not found")
	}

	t.Run("optimistic head", func(t *testing.T) {
		chainService := &blockchainmock.ChainService{
			Optimistic:      true,
			OptimisticRoots: make(map[[32]byte]bool),
		}
		for _, sr := range expectedSlotsAndRoots {
			chainService.OptimisticRoots[sr.Root] = true
		}
		server := &Server{
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
		}
		resp, err := server.ListForkChoiceHeadsV2(ctx, &emptypb.Empty{})
		require.NoError(t, err)
		assert.Equal(t, 2, len(resp.Data))
		for _, sr := range expectedSlotsAndRoots {
			found := false
			for _, h := range resp.Data {
				if h.Slot == sr.Slot {
					found = true
					assert.DeepEqual(t, sr.Root[:], h.Root)
				}
				assert.Equal(t, true, h.ExecutionOptimistic)
			}
			assert.Equal(t, true, found, "Expected head not found")
		}
	})
}

func TestServer_GetForkChoice(t *testing.T) {
	store := doublylinkedtree.New()
	fRoot := [32]byte{'a'}
	fc := &forkchoicetypes.Checkpoint{Epoch: 2, Root: fRoot}
	require.NoError(t, store.UpdateFinalizedCheckpoint(fc))
	bs := &Server{ForkchoiceFetcher: &blockchainmock.ChainService{ForkChoiceStore: store}}
	res, err := bs.GetForkChoice(context.Background(), &empty.Empty{})
	require.NoError(t, err)
	require.Equal(t, primitives.Epoch(2), res.FinalizedCheckpoint.Epoch, "Did not get wanted finalized epoch")
}
