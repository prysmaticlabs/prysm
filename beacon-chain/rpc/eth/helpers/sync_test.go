package helpers

import (
	"context"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	types "github.com/prysmaticlabs/eth2-types"
	grpcutil "github.com/prysmaticlabs/prysm/api/grpc"
	chainmock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	syncmock "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
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
		err = ValidateSync(ctx, syncChecker, chainService, chainService)
		require.NotNil(t, err)
		sts, ok := grpc.ServerTransportStreamFromContext(ctx).(*runtime.ServerTransportStream)
		require.Equal(t, true, ok, "type assertion failed")
		md := sts.Header()
		v, ok := md[strings.ToLower(grpcutil.CustomErrorMetadataKey)]
		require.Equal(t, true, ok, "could not retrieve custom error metadata value")
		assert.DeepEqual(
			t,
			[]string{"{\"sync_details\":{\"head_slot\":\"50\",\"sync_distance\":\"50\",\"is_syncing\":true}}"},
			v,
		)
	})
	t.Run("not syncing", func(t *testing.T) {
		syncChecker := &syncmock.Sync{
			IsSyncing: false,
		}
		err := ValidateSync(ctx, syncChecker, nil, nil)
		require.NoError(t, err)
	})
}

func TestIsOptimistic(t *testing.T) {
	ctx := context.Background()
	st, err := util.NewBeaconState()
	require.NoError(t, err)

	t.Run("optimistic", func(t *testing.T) {
		mockHeadFetcher := &chainmock.ChainService{Optimistic: true}
		o, err := IsOptimistic(ctx, st, mockHeadFetcher)
		require.NoError(t, err)
		assert.Equal(t, true, o)
	})
	t.Run("not optimistic", func(t *testing.T) {
		mockHeadFetcher := &chainmock.ChainService{Optimistic: false}
		o, err := IsOptimistic(ctx, st, mockHeadFetcher)
		require.NoError(t, err)
		assert.Equal(t, false, o)
	})
}
