package helpers

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	grpcutil "github.com/prysmaticlabs/prysm/v3/api/grpc"
	chainmock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	doublylinkedtree "github.com/prysmaticlabs/prysm/v3/beacon-chain/forkchoice/doubly-linked-tree"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/testutil"
	syncmock "github.com/prysmaticlabs/prysm/v3/beacon-chain/sync/initial-sync/testing"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
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
		headSlot := primitives.Slot(100)
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
			[]string{`{"sync_details":{"head_slot":"50","sync_distance":"50","is_syncing":true,"is_optimistic":false,"el_offline":false}}`},
			v,
		)
	})
	t.Run("not syncing", func(t *testing.T) {
		syncChecker := &syncmock.Sync{
			IsSyncing: false,
		}
		headSlot := primitives.Slot(100)
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

	t.Run("head optimistic", func(t *testing.T) {
		cs := &chainmock.ChainService{Optimistic: true}
		o, err := IsOptimistic(ctx, []byte("head"), cs, nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, true, o)
	})
	t.Run("head not optimistic", func(t *testing.T) {
		cs := &chainmock.ChainService{Optimistic: false}
		o, err := IsOptimistic(ctx, []byte("head"), cs, nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, false, o)
	})
	t.Run("genesis", func(t *testing.T) {
		o, err := IsOptimistic(ctx, []byte("genesis"), nil, nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, false, o)
	})
	t.Run("finalized", func(t *testing.T) {
		o, err := IsOptimistic(ctx, []byte("finalized"), nil, nil, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, false, o)
	})
	t.Run("justified", func(t *testing.T) {
		t.Run("is head and head is optimistic", func(t *testing.T) {
			st, err := util.NewBeaconState()
			require.NoError(t, err)
			cs := &chainmock.ChainService{Optimistic: true}
			mf := &testutil.MockFetcher{BeaconState: st}
			o, err := IsOptimistic(ctx, []byte("justified"), cs, mf, cs, nil)
			require.NoError(t, err)
			assert.Equal(t, true, o)
		})
		t.Run("is head and head is not optimistic", func(t *testing.T) {
			st, err := util.NewBeaconState()
			require.NoError(t, err)
			cs := &chainmock.ChainService{Optimistic: false}
			mf := &testutil.MockFetcher{BeaconState: st}
			o, err := IsOptimistic(ctx, []byte("justified"), cs, mf, cs, nil)
			require.NoError(t, err)
			assert.Equal(t, false, o)
		})
		t.Run("head is not optimistic", func(t *testing.T) {
			b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
			require.NoError(t, err)
			db := dbtest.SetupDB(t)
			require.NoError(t, db.SaveBlock(ctx, b))
			fetcherSt, err := util.NewBeaconState()
			require.NoError(t, err)
			chainState, err := util.NewBeaconState()
			require.NoError(t, err)
			require.NoError(t, chainState.SetSlot(fieldparams.SlotsPerEpoch))
			cs := &chainmock.ChainService{Optimistic: false, State: chainState}
			mf := &testutil.MockFetcher{BeaconState: fetcherSt}
			o, err := IsOptimistic(ctx, []byte("justified"), cs, mf, cs, db)
			require.NoError(t, err)
			assert.Equal(t, false, o)
		})
		t.Run("is finalized when head is optimistic", func(t *testing.T) {
			b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
			require.NoError(t, err)
			db := dbtest.SetupDB(t)
			require.NoError(t, db.SaveBlock(ctx, b))
			fetcherSt, err := util.NewBeaconState()
			require.NoError(t, err)
			chainState, err := util.NewBeaconState()
			require.NoError(t, err)
			require.NoError(t, chainState.SetSlot(fieldparams.SlotsPerEpoch))
			cs := &chainmock.ChainService{Optimistic: true, State: chainState, FinalizedCheckPoint: &eth.Checkpoint{Epoch: 1}}
			mf := &testutil.MockFetcher{BeaconState: fetcherSt}
			o, err := IsOptimistic(ctx, []byte("justified"), cs, mf, cs, db)
			require.NoError(t, err)
			assert.Equal(t, false, o)
		})
		t.Run("is not finalized when head is optimistic", func(t *testing.T) {
			b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
			require.NoError(t, err)
			db := dbtest.SetupDB(t)
			require.NoError(t, db.SaveBlock(ctx, b))
			fetcherSt, err := util.NewBeaconState()
			require.NoError(t, err)
			require.NoError(t, fetcherSt.SetSlot(fieldparams.SlotsPerEpoch))
			chainState, err := util.NewBeaconState()
			require.NoError(t, err)
			require.NoError(t, chainState.SetSlot(fieldparams.SlotsPerEpoch*2))
			cs := &chainmock.ChainService{Optimistic: true, State: chainState, FinalizedCheckPoint: &eth.Checkpoint{Epoch: 0}}
			mf := &testutil.MockFetcher{BeaconState: fetcherSt}
			o, err := IsOptimistic(ctx, []byte("justified"), cs, mf, cs, db)
			require.NoError(t, err)
			assert.Equal(t, true, o)
		})
		t.Run("no canonical blocks", func(t *testing.T) {
			b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
			require.NoError(t, err)
			db := dbtest.SetupDB(t)
			require.NoError(t, db.SaveBlock(ctx, b))
			fetcherSt, err := util.NewBeaconState()
			require.NoError(t, err)
			chainState, err := util.NewBeaconState()
			require.NoError(t, err)
			require.NoError(t, chainState.SetSlot(fieldparams.SlotsPerEpoch))
			cs := &chainmock.ChainService{Optimistic: false, State: chainState, CanonicalRoots: map[[32]byte]bool{}}
			mf := &testutil.MockFetcher{BeaconState: fetcherSt}
			o, err := IsOptimistic(ctx, []byte("justified"), nil, mf, cs, db)
			require.NoError(t, err)
			assert.Equal(t, true, o)
		})
	})
	t.Run("root", func(t *testing.T) {
		t.Run("is head and head is optimistic", func(t *testing.T) {
			st, err := util.NewBeaconState()
			require.NoError(t, err)
			cs := &chainmock.ChainService{Optimistic: true}
			mf := &testutil.MockFetcher{BeaconState: st}
			o, err := IsOptimistic(ctx, bytesutil.PadTo([]byte("root"), 32), cs, mf, cs, nil)
			require.NoError(t, err)
			assert.Equal(t, true, o)
		})
		t.Run("is head and head is not optimistic", func(t *testing.T) {
			st, err := util.NewBeaconState()
			require.NoError(t, err)
			cs := &chainmock.ChainService{Optimistic: false}
			mf := &testutil.MockFetcher{BeaconState: st}
			o, err := IsOptimistic(ctx, bytesutil.PadTo([]byte("root"), 32), cs, mf, cs, nil)
			require.NoError(t, err)
			assert.Equal(t, false, o)
		})
		t.Run("head is not optimistic", func(t *testing.T) {
			b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
			require.NoError(t, err)
			db := dbtest.SetupDB(t)
			require.NoError(t, db.SaveBlock(ctx, b))
			fetcherSt, err := util.NewBeaconState()
			require.NoError(t, err)
			ChainSt, err := util.NewBeaconState()
			require.NoError(t, err)
			require.NoError(t, ChainSt.SetSlot(fieldparams.SlotsPerEpoch))
			cs := &chainmock.ChainService{Optimistic: false, State: ChainSt}
			mf := &testutil.MockFetcher{BeaconState: fetcherSt}
			o, err := IsOptimistic(ctx, bytesutil.PadTo([]byte("root"), 32), cs, mf, cs, db)
			require.NoError(t, err)
			assert.Equal(t, false, o)
		})
		t.Run("is finalized when head is optimistic", func(t *testing.T) {
			b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
			require.NoError(t, err)
			db := dbtest.SetupDB(t)
			require.NoError(t, db.SaveBlock(ctx, b))
			fetcherSt, err := util.NewBeaconState()
			require.NoError(t, err)
			chainSt, err := util.NewBeaconState()
			require.NoError(t, err)
			require.NoError(t, chainSt.SetSlot(fieldparams.SlotsPerEpoch))
			cs := &chainmock.ChainService{Optimistic: true, State: chainSt, FinalizedCheckPoint: &eth.Checkpoint{Epoch: 1}}
			mf := &testutil.MockFetcher{BeaconState: fetcherSt}
			o, err := IsOptimistic(ctx, bytesutil.PadTo([]byte("root"), 32), cs, mf, cs, db)
			require.NoError(t, err)
			assert.Equal(t, false, o)
		})
		t.Run("is not finalized when head is optimistic", func(t *testing.T) {
			b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
			require.NoError(t, err)
			db := dbtest.SetupDB(t)
			require.NoError(t, db.SaveBlock(ctx, b))
			fetcherSt, err := util.NewBeaconState()
			require.NoError(t, err)
			require.NoError(t, fetcherSt.SetSlot(fieldparams.SlotsPerEpoch))
			ChainSt, err := util.NewBeaconState()
			require.NoError(t, err)
			require.NoError(t, ChainSt.SetSlot(fieldparams.SlotsPerEpoch*2))
			cs := &chainmock.ChainService{Optimistic: true, State: ChainSt, FinalizedCheckPoint: &eth.Checkpoint{Epoch: 0}}
			mf := &testutil.MockFetcher{BeaconState: fetcherSt}
			o, err := IsOptimistic(ctx, bytesutil.PadTo([]byte("root"), 32), cs, mf, cs, db)
			require.NoError(t, err)
			assert.Equal(t, true, o)
		})
		t.Run("no canonical blocks", func(t *testing.T) {
			b, err := blocks.NewSignedBeaconBlock(util.NewBeaconBlock())
			require.NoError(t, err)
			db := dbtest.SetupDB(t)
			require.NoError(t, db.SaveBlock(ctx, b))
			fetcherSt, err := util.NewBeaconState()
			require.NoError(t, err)
			chainSt, err := util.NewBeaconState()
			require.NoError(t, err)
			require.NoError(t, chainSt.SetSlot(fieldparams.SlotsPerEpoch))
			cs := &chainmock.ChainService{Optimistic: false, State: chainSt, CanonicalRoots: map[[32]byte]bool{}}
			mf := &testutil.MockFetcher{BeaconState: fetcherSt}
			o, err := IsOptimistic(ctx, bytesutil.PadTo([]byte("root"), 32), nil, mf, cs, db)
			require.NoError(t, err)
			assert.Equal(t, true, o)
		})
	})
	t.Run("slot", func(t *testing.T) {
		t.Run("head is not optimistic", func(t *testing.T) {
			cs := &chainmock.ChainService{Optimistic: false}
			o, err := IsOptimistic(ctx, []byte("0"), cs, nil, nil, nil)
			require.NoError(t, err)
			assert.Equal(t, false, o)
		})
		t.Run("is finalized when head is optimistic", func(t *testing.T) {
			cs := &chainmock.ChainService{Optimistic: true, FinalizedCheckPoint: &eth.Checkpoint{Epoch: 1}}
			o, err := IsOptimistic(ctx, []byte("0"), cs, nil, cs, nil)
			require.NoError(t, err)
			assert.Equal(t, false, o)
		})
		t.Run("is head", func(t *testing.T) {
			fetcherSt, err := util.NewBeaconState()
			require.NoError(t, err)
			chainSt, err := util.NewBeaconState()
			require.NoError(t, err)
			require.NoError(t, chainSt.SetSlot(fieldparams.SlotsPerEpoch))
			cs := &chainmock.ChainService{Optimistic: true, State: chainSt, FinalizedCheckPoint: &eth.Checkpoint{Epoch: 0}}
			mf := &testutil.MockFetcher{BeaconState: fetcherSt}
			o, err := IsOptimistic(ctx, []byte(strconv.Itoa(fieldparams.SlotsPerEpoch)), cs, mf, cs, nil)
			require.NoError(t, err)
			assert.Equal(t, true, o)
		})
		t.Run("ancestor is optimistic", func(t *testing.T) {
			r := bytesutil.ToBytes32([]byte("root"))
			st, err := util.NewBeaconState()
			require.NoError(t, err)
			fcs := &doublylinkedtree.ForkChoice{}
			cs := &chainmock.ChainService{Optimistic: true, ForkChoiceStore: fcs, OptimisticRoots: map[[32]byte]bool{r: true}, FinalizedCheckPoint: &eth.Checkpoint{Epoch: 0}}
			mf := &testutil.MockFetcher{BeaconState: st}
			o, err := IsOptimistic(ctx, []byte(strconv.Itoa(fieldparams.SlotsPerEpoch)), cs, mf, cs, nil)
			require.NoError(t, err)
			assert.Equal(t, true, o)
		})
		t.Run("ancestor is not optimistic", func(t *testing.T) {
			r := bytesutil.ToBytes32([]byte("root"))
			st, err := util.NewBeaconState()
			require.NoError(t, err)
			fcs := &doublylinkedtree.ForkChoice{}
			cs := &chainmock.ChainService{Optimistic: true, ForkChoiceStore: fcs, OptimisticRoots: map[[32]byte]bool{r: false}, FinalizedCheckPoint: &eth.Checkpoint{Epoch: 0}}
			mf := &testutil.MockFetcher{BeaconState: st}
			o, err := IsOptimistic(ctx, []byte(strconv.Itoa(fieldparams.SlotsPerEpoch)), cs, mf, cs, nil)
			require.NoError(t, err)
			assert.Equal(t, true, o)
		})
	})
}
