package debug

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	blockchainmock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpbv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetBeaconState(t *testing.T) {
	fakeState, err := util.NewBeaconState()
	require.NoError(t, err)
	server := &Server{
		StateFetcher: &testutil.MockFetcher{
			BeaconState: fakeState,
		},
	}
	resp, err := server.GetBeaconState(context.Background(), &ethpbv1.StateRequest{
		StateId: make([]byte, 0),
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestGetBeaconStateV2(t *testing.T) {
	ctx := context.Background()
	db := dbTest.SetupDB(t)

	t.Run("Phase 0", func(t *testing.T) {
		fakeState, err := util.NewBeaconState()
		require.NoError(t, err)
		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
			HeadFetcher: &blockchainmock.ChainService{},
			BeaconDB:    db,
		}
		resp, err := server.GetBeaconStateV2(context.Background(), &ethpbv2.StateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, ethpbv2.Version_PHASE0, resp.Version)
	})
	t.Run("Altair", func(t *testing.T) {
		fakeState, _ := util.DeterministicGenesisStateAltair(t, 1)
		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
			HeadFetcher: &blockchainmock.ChainService{},
			BeaconDB:    db,
		}
		resp, err := server.GetBeaconStateV2(context.Background(), &ethpbv2.StateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, ethpbv2.Version_ALTAIR, resp.Version)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		fakeState, _ := util.DeterministicGenesisStateBellatrix(t, 1)
		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
			HeadFetcher: &blockchainmock.ChainService{},
			BeaconDB:    db,
		}
		resp, err := server.GetBeaconStateV2(context.Background(), &ethpbv2.StateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, ethpbv2.Version_BELLATRIX, resp.Version)
	})
	t.Run("execution optimistic", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		require.NoError(t, db.SaveBlock(ctx, wrapper.WrappedPhase0SignedBeaconBlock(blk)))
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		fakeState, _ := util.DeterministicGenesisStateBellatrix(t, 1)
		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
			HeadFetcher: &blockchainmock.ChainService{Optimistic: true},
			BeaconDB:    db,
		}
		resp, err := server.GetBeaconStateV2(context.Background(), &ethpbv2.StateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
}

func TestGetBeaconStateSSZ(t *testing.T) {
	fakeState, err := util.NewBeaconState()
	require.NoError(t, err)
	sszState, err := fakeState.MarshalSSZ()
	require.NoError(t, err)

	server := &Server{
		StateFetcher: &testutil.MockFetcher{
			BeaconState: fakeState,
		},
	}
	resp, err := server.GetBeaconStateSSZ(context.Background(), &ethpbv1.StateRequest{
		StateId: make([]byte, 0),
	})
	require.NoError(t, err)
	assert.NotNil(t, resp)

	assert.DeepEqual(t, sszState, resp.Data)
}

func TestGetBeaconStateSSZV2(t *testing.T) {
	t.Run("Phase 0", func(t *testing.T) {
		fakeState, err := util.NewBeaconState()
		require.NoError(t, err)
		sszState, err := fakeState.MarshalSSZ()
		require.NoError(t, err)

		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
		}
		resp, err := server.GetBeaconStateSSZV2(context.Background(), &ethpbv2.StateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		assert.DeepEqual(t, sszState, resp.Data)
	})
	t.Run("Altair", func(t *testing.T) {
		fakeState, _ := util.DeterministicGenesisStateAltair(t, 1)
		sszState, err := fakeState.MarshalSSZ()
		require.NoError(t, err)

		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
		}
		resp, err := server.GetBeaconStateSSZV2(context.Background(), &ethpbv2.StateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		assert.DeepEqual(t, sszState, resp.Data)
	})
	t.Run("Bellatrix", func(t *testing.T) {
		fakeState, _ := util.DeterministicGenesisStateBellatrix(t, 1)
		sszState, err := fakeState.MarshalSSZ()
		require.NoError(t, err)

		server := &Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: fakeState,
			},
		}
		resp, err := server.GetBeaconStateSSZV2(context.Background(), &ethpbv2.StateRequestV2{
			StateId: make([]byte, 0),
		})
		require.NoError(t, err)
		assert.NotNil(t, resp)

		assert.DeepEqual(t, sszState, resp.Data)
	})
}

func TestListForkChoiceHeads(t *testing.T) {
	ctx := context.Background()

	expectedSlotsAndRoots := []struct {
		Slot types.Slot
		Root [32]byte
	}{{
		Slot: 0,
		Root: bytesutil.ToBytes32(bytesutil.PadTo([]byte("foo"), 32)),
	}, {
		Slot: 1,
		Root: bytesutil.ToBytes32(bytesutil.PadTo([]byte("bar"), 32)),
	}}

	server := &Server{
		HeadFetcher: &blockchainmock.ChainService{},
	}
	resp, err := server.ListForkChoiceHeads(ctx, &emptypb.Empty{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(resp.Data))
	for _, sr := range expectedSlotsAndRoots {
		found := false
		for _, h := range resp.Data {
			if h.Slot == sr.Slot {
				found = true
				assert.DeepEqual(t, sr.Root[:], h.Root)
			}
		}
		assert.Equal(t, true, found, "Expected head not found")
	}
}

func TestListForkChoiceHeadsV2(t *testing.T) {
	ctx := context.Background()

	expectedSlotsAndRoots := []struct {
		Slot types.Slot
		Root [32]byte
	}{{
		Slot: 0,
		Root: bytesutil.ToBytes32(bytesutil.PadTo([]byte("foo"), 32)),
	}, {
		Slot: 1,
		Root: bytesutil.ToBytes32(bytesutil.PadTo([]byte("bar"), 32)),
	}}

	server := &Server{
		HeadFetcher: &blockchainmock.ChainService{},
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
		server := &Server{
			HeadFetcher: &blockchainmock.ChainService{Optimistic: true},
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
