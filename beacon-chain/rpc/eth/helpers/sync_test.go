package helpers

import (
	"context"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	grpcutil "github.com/prysmaticlabs/prysm/v3/api/grpc"
	chainmock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	syncmock "github.com/prysmaticlabs/prysm/v3/beacon-chain/sync/initial-sync/testing"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"google.golang.org/grpc"
)

func TestValidateSync(t *testing.T) {
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})
	t.Run("syncing", func(t *testing.T) {
		syncChecker := &syncmock.Sync{
			IsSyncing: true,
		}
		headSlot := types.Slot(100)
		st, err := util.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, st.SetSlot(50))
		chainService := &chainmock.ChainService{
			Slot:  &headSlot,
			State: st,
		}
		err = ValidateSync(ctx, syncChecker, chainService, chainService, chainService)
		require.NotNil(t, err)
		sts, ok := grpc.ServerTransportStreamFromContext(ctx).(*runtime.ServerTransportStream)
		require.Equal(t, true, ok, "type assertion failed")
		md := sts.Header()
		v, ok := md[strings.ToLower(grpcutil.CustomErrorMetadataKey)]
		require.Equal(t, true, ok, "could not retrieve custom error metadata value")
		assert.DeepEqual(
			t,
			[]string{"{\"sync_details\":{\"head_slot\":\"50\",\"sync_distance\":\"50\",\"is_syncing\":true,\"is_optimistic\":false}}"},
			v,
		)
	})
	t.Run("not syncing", func(t *testing.T) {
		syncChecker := &syncmock.Sync{
			IsSyncing: false,
		}
		headSlot := types.Slot(100)
		st, err := util.NewBeaconState()
		require.NoError(t, err)
		require.NoError(t, st.SetSlot(50))
		chainService := &chainmock.ChainService{
			Slot:  &headSlot,
			State: st,
		}
		err = ValidateSync(ctx, syncChecker, nil, nil, chainService)
		require.NoError(t, err)
	})
}

func TestIsOptimistic(t *testing.T) {
	ctx := context.Background()
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	t.Run("optimistic", func(t *testing.T) {
		mockOptSyncFetcher := &chainmock.ChainService{Optimistic: true}
		o, err := IsOptimistic(ctx, st, mockOptSyncFetcher)
		require.NoError(t, err)
		assert.Equal(t, true, o)
	})
	t.Run("not optimistic", func(t *testing.T) {
		mockOptSyncFetcher := &chainmock.ChainService{Optimistic: false}
		o, err := IsOptimistic(ctx, st, mockOptSyncFetcher)
		require.NoError(t, err)
		assert.Equal(t, false, o)
	})
	t.Run("zero state root", func(t *testing.T) {
		zeroRootSt, err := util.NewBeaconState()
		require.NoError(t, err)
		h := zeroRootSt.LatestBlockHeader()
		h.StateRoot = make([]byte, 32)
		require.NoError(t, zeroRootSt.SetLatestBlockHeader(h))
		mockOptSyncFetcher := &chainmock.ChainService{}
		_, err = IsOptimistic(ctx, st, mockOptSyncFetcher)
		require.NoError(t, err)
		assert.DeepEqual(
			t,
			[32]byte{0xfc, 0x0, 0xe9, 0x6d, 0xb, 0x8b, 0x2, 0x2f, 0x61, 0xeb, 0x92, 0x10, 0xfd, 0x80, 0x84, 0x2b, 0x26, 0x61, 0xdc, 0x94, 0x5f, 0x7a, 0xf0, 0x0, 0xbc, 0x38, 0x6, 0x38, 0x71, 0x95, 0x43, 0x1},
			mockOptSyncFetcher.OptimisticCheckRootReceived,
		)
	})
	t.Run("non-zero state root", func(t *testing.T) {
		mockOptSyncFetcher := &chainmock.ChainService{}
		_, err = IsOptimistic(ctx, st, mockOptSyncFetcher)
		require.NoError(t, err)
		assert.DeepEqual(
			t,
			[32]byte{0xfc, 0x0, 0xe9, 0x6d, 0xb, 0x8b, 0x2, 0x2f, 0x61, 0xeb, 0x92, 0x10, 0xfd, 0x80, 0x84, 0x2b, 0x26, 0x61, 0xdc, 0x94, 0x5f, 0x7a, 0xf0, 0x0, 0xbc, 0x38, 0x6, 0x38, 0x71, 0x95, 0x43, 0x1},
			mockOptSyncFetcher.OptimisticCheckRootReceived,
		)
	})
}
