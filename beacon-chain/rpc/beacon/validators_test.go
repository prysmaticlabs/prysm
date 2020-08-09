package beacon

import (
	"context"
	"encoding/binary"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbTest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stategen"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	mockSync "github.com/prysmaticlabs/prysm/beacon-chain/sync/initial-sync/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestServer_GetValidatorActiveSetChanges_CannotRequestFutureEpoch(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()
	st := testutil.NewBeaconState()
	if err := st.SetSlot(0); err != nil {
		t.Fatal(err)
	}
	bs := &Server{
		GenesisTimeFetcher: &mock.ChainService{},
		HeadFetcher: &mock.ChainService{
			State: st,
		},
		BeaconDB: db,
	}

	wanted := "Cannot retrieve information about an epoch in the future"
	_, err := bs.GetValidatorActiveSetChanges(
		ctx,
		&ethpb.GetValidatorActiveSetChangesRequest{
			QueryFilter: &ethpb.GetValidatorActiveSetChangesRequest_Epoch{
				Epoch: helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot()) + 1,
			},
		},
	)
	assert.ErrorContains(t, wanted, err)
}

func TestServer_ListValidatorBalances_CannotRequestFutureEpoch(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(0))
	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: st,
		},
		GenesisTimeFetcher: &mock.ChainService{},
	}

	wanted := "Cannot retrieve information about an epoch in the future"
	_, err := bs.ListValidatorBalances(
		ctx,
		&ethpb.ListValidatorBalancesRequest{
			QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{
				Epoch: helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot()) + 1,
			},
		},
	)
	assert.ErrorContains(t, wanted, err)
}

func TestServer_ListValidatorBalances_NoResults(t *testing.T) {
	db, sc := dbTest.SetupDB(t)
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	ctx := context.Background()
	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(0))
	bs := &Server{
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
	}

	headState := testutil.NewBeaconState()
	b := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, b))
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, db.SaveState(ctx, headState, gRoot))

	wanted := &ethpb.ValidatorBalances{
		Balances:      make([]*ethpb.ValidatorBalances_Balance, 0),
		TotalSize:     int32(0),
		NextPageToken: strconv.Itoa(0),
	}
	res, err := bs.ListValidatorBalances(
		ctx,
		&ethpb.ListValidatorBalancesRequest{
			QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{
				Epoch: 0,
			},
		},
	)
	require.NoError(t, err)
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

func TestServer_ListValidatorBalances_DefaultResponse_NoArchive(t *testing.T) {
	db, sc := dbTest.SetupDB(t)
	ctx := context.Background()

	numItems := 100
	validators := make([]*ethpb.Validator, numItems)
	balances := make([]uint64, numItems)
	balancesResponse := make([]*ethpb.ValidatorBalances_Balance, numItems)
	for i := 0; i < numItems; i++ {
		validators[i] = &ethpb.Validator{
			PublicKey:             pubKey(uint64(i)),
			WithdrawalCredentials: make([]byte, 32),
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
		balancesResponse[i] = &ethpb.ValidatorBalances_Balance{
			PublicKey: pubKey(uint64(i)),
			Index:     uint64(i),
			Balance:   params.BeaconConfig().MaxEffectiveBalance,
		}
	}
	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(0))
	require.NoError(t, st.SetValidators(validators))
	require.NoError(t, st.SetBalances(balances))
	b := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, b))
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, db.SaveState(ctx, st, gRoot))
	bs := &Server{
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
		HeadFetcher: &mock.ChainService{
			State: st,
		},
	}
	res, err := bs.ListValidatorBalances(
		ctx,
		&ethpb.ListValidatorBalancesRequest{
			QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0},
		},
	)
	require.NoError(t, err)
	assert.DeepEqual(t, balancesResponse, res.Balances)
}

func TestServer_ListValidatorBalances_PaginationOutOfRange(t *testing.T) {
	db, sc := dbTest.SetupDB(t)
	ctx := context.Background()
	setupValidators(t, db, 3)
	st := testutil.NewBeaconState()
	b := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, b))
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, db.SaveState(ctx, st, gRoot))

	bs := &Server{
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
		HeadFetcher: &mock.ChainService{
			State: st,
		},
	}

	req := &ethpb.ListValidatorBalancesRequest{PageToken: strconv.Itoa(1), PageSize: 100, QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0}}
	wanted := fmt.Sprintf("page start %d >= list %d", req.PageSize, len(st.Balances()))
	if _, err := bs.ListValidatorBalances(context.Background(), req); err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestServer_ListValidatorBalances_ExceedsMaxPageSize(t *testing.T) {
	bs := &Server{}
	exceedsMax := int32(cmd.Get().MaxRPCPageSize + 1)

	wanted := fmt.Sprintf(
		"Requested page size %d can not be greater than max size %d",
		exceedsMax,
		cmd.Get().MaxRPCPageSize,
	)
	req := &ethpb.ListValidatorBalancesRequest{PageSize: exceedsMax}
	_, err := bs.ListValidatorBalances(context.Background(), req)
	assert.ErrorContains(t, wanted, err)
}

func pubKey(i uint64) []byte {
	pubKey := make([]byte, params.BeaconConfig().BLSPubkeyLength)
	binary.LittleEndian.PutUint64(pubKey, i)
	return pubKey
}

func TestServer_ListValidatorBalances_Pagination_Default(t *testing.T) {
	db, sc := dbTest.SetupDB(t)
	ctx := context.Background()

	setupValidators(t, db, 100)
	headState, err := db.HeadState(context.Background())
	require.NoError(t, err)
	b := testutil.NewBeaconBlock()
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, db.SaveState(ctx, headState, gRoot))

	bs := &Server{
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	tests := []struct {
		req *ethpb.ListValidatorBalancesRequest
		res *ethpb.ValidatorBalances
	}{
		{req: &ethpb.ListValidatorBalancesRequest{PublicKeys: [][]byte{pubKey(99)}, QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0}},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{Index: 99, PublicKey: pubKey(99), Balance: 99},
				},
				NextPageToken: "",
				TotalSize:     1,
			},
		},
		{req: &ethpb.ListValidatorBalancesRequest{Indices: []uint64{1, 2, 3}, QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0}},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{Index: 1, PublicKey: pubKey(1), Balance: 1},
					{Index: 2, PublicKey: pubKey(2), Balance: 2},
					{Index: 3, PublicKey: pubKey(3), Balance: 3},
				},
				NextPageToken: "",
				TotalSize:     3,
			},
		},
		{req: &ethpb.ListValidatorBalancesRequest{PublicKeys: [][]byte{pubKey(10), pubKey(11), pubKey(12)}, QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0}},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{Index: 10, PublicKey: pubKey(10), Balance: 10},
					{Index: 11, PublicKey: pubKey(11), Balance: 11},
					{Index: 12, PublicKey: pubKey(12), Balance: 12},
				},
				NextPageToken: "",
				TotalSize:     3,
			}},
		{req: &ethpb.ListValidatorBalancesRequest{PublicKeys: [][]byte{pubKey(2), pubKey(3)}, Indices: []uint64{3, 4}, QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0}}, // Duplication
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{Index: 2, PublicKey: pubKey(2), Balance: 2},
					{Index: 3, PublicKey: pubKey(3), Balance: 3},
					{Index: 4, PublicKey: pubKey(4), Balance: 4},
				},
				NextPageToken: "",
				TotalSize:     3,
			}},
		{req: &ethpb.ListValidatorBalancesRequest{PublicKeys: [][]byte{{}}, Indices: []uint64{3, 4}, QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0}}, // Public key has a blank value
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{Index: 3, PublicKey: pubKey(3), Balance: 3},
					{Index: 4, PublicKey: pubKey(4), Balance: 4},
				},
				NextPageToken: "",
				TotalSize:     2,
			}},
	}
	for _, test := range tests {
		res, err := bs.ListValidatorBalances(context.Background(), test.req)
		require.NoError(t, err)
		if !proto.Equal(res, test.res) {
			t.Errorf("Expected %v, received %v", test.res, res)
		}
	}
}

func TestServer_ListValidatorBalances_Pagination_CustomPageSizes(t *testing.T) {
	db, sc := dbTest.SetupDB(t)
	ctx := context.Background()

	count := 1000
	setupValidators(t, db, count)
	headState, err := db.HeadState(context.Background())
	require.NoError(t, err)
	b := testutil.NewBeaconBlock()
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, db.SaveState(ctx, headState, gRoot))

	bs := &Server{
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	tests := []struct {
		req *ethpb.ListValidatorBalancesRequest
		res *ethpb.ValidatorBalances
	}{
		{req: &ethpb.ListValidatorBalancesRequest{PageToken: strconv.Itoa(1), PageSize: 3, QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0}},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{PublicKey: pubKey(3), Index: 3, Balance: uint64(3)},
					{PublicKey: pubKey(4), Index: 4, Balance: uint64(4)},
					{PublicKey: pubKey(5), Index: 5, Balance: uint64(5)}},
				NextPageToken: strconv.Itoa(2),
				TotalSize:     int32(count)}},
		{req: &ethpb.ListValidatorBalancesRequest{PageToken: strconv.Itoa(10), PageSize: 5, QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0}},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{PublicKey: pubKey(50), Index: 50, Balance: uint64(50)},
					{PublicKey: pubKey(51), Index: 51, Balance: uint64(51)},
					{PublicKey: pubKey(52), Index: 52, Balance: uint64(52)},
					{PublicKey: pubKey(53), Index: 53, Balance: uint64(53)},
					{PublicKey: pubKey(54), Index: 54, Balance: uint64(54)}},
				NextPageToken: strconv.Itoa(11),
				TotalSize:     int32(count)}},
		{req: &ethpb.ListValidatorBalancesRequest{PageToken: strconv.Itoa(33), PageSize: 3, QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0}},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{PublicKey: pubKey(99), Index: 99, Balance: uint64(99)},
					{PublicKey: pubKey(100), Index: 100, Balance: uint64(100)},
					{PublicKey: pubKey(101), Index: 101, Balance: uint64(101)},
				},
				NextPageToken: "34",
				TotalSize:     int32(count)}},
		{req: &ethpb.ListValidatorBalancesRequest{PageSize: 2, QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0}},
			res: &ethpb.ValidatorBalances{
				Balances: []*ethpb.ValidatorBalances_Balance{
					{PublicKey: pubKey(0), Index: 0, Balance: uint64(0)},
					{PublicKey: pubKey(1), Index: 1, Balance: uint64(1)}},
				NextPageToken: strconv.Itoa(1),
				TotalSize:     int32(count)}},
	}
	for _, test := range tests {
		res, err := bs.ListValidatorBalances(context.Background(), test.req)
		require.NoError(t, err)
		if !proto.Equal(res, test.res) {
			t.Errorf("Expected %v, received %v", test.res, res)
		}
	}
}

func TestServer_ListValidatorBalances_OutOfRange(t *testing.T) {
	db, sc := dbTest.SetupDB(t)
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()
	ctx := context.Background()
	setupValidators(t, db, 1)

	headState, err := db.HeadState(context.Background())
	require.NoError(t, err)
	b := testutil.NewBeaconBlock()
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, db.SaveState(ctx, headState, gRoot))

	bs := &Server{
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	req := &ethpb.ListValidatorBalancesRequest{Indices: []uint64{uint64(1)}, QueryFilter: &ethpb.ListValidatorBalancesRequest_Epoch{Epoch: 0}}
	wanted := "Validator index 1 >= balance list 1"
	_, err = bs.ListValidatorBalances(context.Background(), req)
	assert.ErrorContains(t, wanted, err)
}

func TestServer_ListValidators_CannotRequestFutureEpoch(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(0))
	bs := &Server{
		BeaconDB: db,
		GenesisTimeFetcher: &mock.ChainService{
			// We are in epoch 0.
			Genesis: time.Now(),
		},
		HeadFetcher: &mock.ChainService{
			State: st,
		},
	}

	wanted := "Cannot retrieve information about an epoch in the future"
	_, err := bs.ListValidators(
		ctx,
		&ethpb.ListValidatorsRequest{
			QueryFilter: &ethpb.ListValidatorsRequest_Epoch{
				Epoch: 1,
			},
		},
	)
	assert.ErrorContains(t, wanted, err)
}

func TestServer_ListValidators_NoResults(t *testing.T) {
	db, _ := dbTest.SetupDB(t)

	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	ctx := context.Background()
	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(0))
	gRoot := [32]byte{'g'}
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, db.SaveState(ctx, st, gRoot))
	bs := &Server{
		BeaconDB: db,
		GenesisTimeFetcher: &mock.ChainService{
			// We are in epoch 0.
			Genesis: time.Now(),
		},
		HeadFetcher: &mock.ChainService{
			State: st,
		},
		StateGen: stategen.New(db, cache.NewStateSummaryCache()),
	}
	wanted := &ethpb.Validators{
		ValidatorList: make([]*ethpb.Validators_ValidatorContainer, 0),
		TotalSize:     int32(0),
		NextPageToken: strconv.Itoa(0),
	}
	res, err := bs.ListValidators(
		ctx,
		&ethpb.ListValidatorsRequest{
			QueryFilter: &ethpb.ListValidatorsRequest_Epoch{
				Epoch: 0,
			},
		},
	)
	require.NoError(t, err)
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted %v, received %v", wanted, res)
	}
}

func TestServer_ListValidators_OnlyActiveValidators(t *testing.T) {
	ctx := context.Background()
	db, _ := dbTest.SetupDB(t)
	count := 100
	balances := make([]uint64, count)
	validators := make([]*ethpb.Validator, count)
	activeValidators := make([]*ethpb.Validators_ValidatorContainer, 0)
	for i := 0; i < count; i++ {
		pubKey := pubKey(uint64(i))
		balances[i] = params.BeaconConfig().MaxEffectiveBalance

		// We mark even validators as active, and odd validators as inactive.
		if i%2 == 0 {
			val := &ethpb.Validator{
				PublicKey:             pubKey,
				WithdrawalCredentials: make([]byte, 32),
				ActivationEpoch:       0,
				ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			}
			validators[i] = val
			activeValidators = append(activeValidators, &ethpb.Validators_ValidatorContainer{
				Index:     uint64(i),
				Validator: val,
			})
		} else {
			validators[i] = &ethpb.Validator{
				PublicKey:             pubKey,
				WithdrawalCredentials: make([]byte, 32),
				ActivationEpoch:       0,
				ExitEpoch:             0,
			}
		}
	}
	st := testutil.NewBeaconState()
	require.NoError(t, st.SetValidators(validators))
	require.NoError(t, st.SetBalances(balances))

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: st,
		},
		GenesisTimeFetcher: &mock.ChainService{
			// We are in epoch 0.
			Genesis: time.Now(),
		},
		StateGen: stategen.New(db, cache.NewStateSummaryCache()),
	}

	b := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, b))
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, db.SaveState(ctx, st, gRoot))

	received, err := bs.ListValidators(ctx, &ethpb.ListValidatorsRequest{
		Active: true,
	})
	require.NoError(t, err)
	assert.DeepEqual(t, activeValidators, received.ValidatorList)
}

func TestServer_ListValidators_NoPagination(t *testing.T) {
	db, _ := dbTest.SetupDB(t)

	validators, _ := setupValidators(t, db, 100)
	want := make([]*ethpb.Validators_ValidatorContainer, len(validators))
	for i := 0; i < len(validators); i++ {
		want[i] = &ethpb.Validators_ValidatorContainer{
			Index:     uint64(i),
			Validator: validators[i],
		}
	}
	headState, err := db.HeadState(context.Background())
	require.NoError(t, err)

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		GenesisTimeFetcher: &mock.ChainService{
			// We are in epoch 0.
			Genesis: time.Now(),
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		StateGen: stategen.New(db, cache.NewStateSummaryCache()),
	}

	received, err := bs.ListValidators(context.Background(), &ethpb.ListValidatorsRequest{})
	require.NoError(t, err)
	assert.DeepEqual(t, want, received.ValidatorList, "Incorrect respond of validators")
}

func TestServer_ListValidators_StategenNotUsed(t *testing.T) {
	db, _ := dbTest.SetupDB(t)

	validators, _ := setupValidators(t, db, 100)
	want := make([]*ethpb.Validators_ValidatorContainer, len(validators))
	for i := 0; i < len(validators); i++ {
		want[i] = &ethpb.Validators_ValidatorContainer{
			Index:     uint64(i),
			Validator: validators[i],
		}
	}
	headState, err := db.HeadState(context.Background())
	require.NoError(t, err)

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		GenesisTimeFetcher: &mock.ChainService{
			// We are in epoch 0.
			Genesis: time.Now(),
		},
	}

	received, err := bs.ListValidators(context.Background(), &ethpb.ListValidatorsRequest{})
	require.NoError(t, err)
	assert.DeepEqual(t, want, received.ValidatorList, "Incorrect respond of validators")
}

func TestServer_ListValidators_IndicesPubKeys(t *testing.T) {
	db, _ := dbTest.SetupDB(t)

	validators, _ := setupValidators(t, db, 100)
	indicesWanted := []uint64{2, 7, 11, 17}
	pubkeyIndicesWanted := []uint64{3, 5, 9, 15}
	allIndicesWanted := append(indicesWanted, pubkeyIndicesWanted...)
	want := make([]*ethpb.Validators_ValidatorContainer, len(allIndicesWanted))
	for i, idx := range allIndicesWanted {
		want[i] = &ethpb.Validators_ValidatorContainer{
			Index:     idx,
			Validator: validators[idx],
		}
	}
	sort.Slice(want, func(i int, j int) bool {
		return want[i].Index < want[j].Index
	})

	headState, err := db.HeadState(context.Background())
	require.NoError(t, err)

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{
			// We are in epoch 0.
			Genesis: time.Now(),
		},
		StateGen: stategen.New(db, cache.NewStateSummaryCache()),
	}

	pubKeysWanted := make([][]byte, len(pubkeyIndicesWanted))
	for i, indice := range pubkeyIndicesWanted {
		pubKeysWanted[i] = pubKey(indice)
	}
	req := &ethpb.ListValidatorsRequest{
		Indices:    indicesWanted,
		PublicKeys: pubKeysWanted,
	}
	received, err := bs.ListValidators(context.Background(), req)
	require.NoError(t, err)
	assert.DeepEqual(t, want, received.ValidatorList, "Incorrect respond of validators")
}

func TestServer_ListValidators_Pagination(t *testing.T) {
	db, _ := dbTest.SetupDB(t)

	count := 100
	setupValidators(t, db, count)
	headState, err := db.HeadState(context.Background())
	require.NoError(t, err)

	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{
			// We are in epoch 0.
			Genesis: time.Now(),
		},
		StateGen: stategen.New(db, cache.NewStateSummaryCache()),
	}

	tests := []struct {
		req *ethpb.ListValidatorsRequest
		res *ethpb.Validators
	}{
		{req: &ethpb.ListValidatorsRequest{PageToken: strconv.Itoa(1), PageSize: 3},
			res: &ethpb.Validators{
				ValidatorList: []*ethpb.Validators_ValidatorContainer{
					{
						Validator: &ethpb.Validator{
							PublicKey:             pubKey(3),
							WithdrawalCredentials: make([]byte, 32),
						},
						Index: 3,
					},
					{
						Validator: &ethpb.Validator{
							PublicKey:             pubKey(4),
							WithdrawalCredentials: make([]byte, 32),
						},
						Index: 4,
					},
					{
						Validator: &ethpb.Validator{
							PublicKey:             pubKey(5),
							WithdrawalCredentials: make([]byte, 32),
						},
						Index: 5,
					},
				},
				NextPageToken: strconv.Itoa(2),
				TotalSize:     int32(count)}},
		{req: &ethpb.ListValidatorsRequest{PageToken: strconv.Itoa(10), PageSize: 5},
			res: &ethpb.Validators{
				ValidatorList: []*ethpb.Validators_ValidatorContainer{
					{
						Validator: &ethpb.Validator{
							PublicKey:             pubKey(50),
							WithdrawalCredentials: make([]byte, 32),
						},
						Index: 50,
					},
					{
						Validator: &ethpb.Validator{
							PublicKey:             pubKey(51),
							WithdrawalCredentials: make([]byte, 32),
						},
						Index: 51,
					},
					{
						Validator: &ethpb.Validator{
							PublicKey:             pubKey(52),
							WithdrawalCredentials: make([]byte, 32),
						},
						Index: 52,
					},
					{
						Validator: &ethpb.Validator{
							PublicKey:             pubKey(53),
							WithdrawalCredentials: make([]byte, 32),
						},
						Index: 53,
					},
					{
						Validator: &ethpb.Validator{
							PublicKey:             pubKey(54),
							WithdrawalCredentials: make([]byte, 32),
						},
						Index: 54,
					},
				},
				NextPageToken: strconv.Itoa(11),
				TotalSize:     int32(count)}},
		{req: &ethpb.ListValidatorsRequest{PageToken: strconv.Itoa(33), PageSize: 3},
			res: &ethpb.Validators{
				ValidatorList: []*ethpb.Validators_ValidatorContainer{
					{
						Validator: &ethpb.Validator{
							PublicKey:             pubKey(99),
							WithdrawalCredentials: make([]byte, 32),
						},
						Index: 99,
					},
				},
				NextPageToken: "",
				TotalSize:     int32(count)}},
		{req: &ethpb.ListValidatorsRequest{PageSize: 2},
			res: &ethpb.Validators{
				ValidatorList: []*ethpb.Validators_ValidatorContainer{
					{
						Validator: &ethpb.Validator{
							PublicKey:             pubKey(0),
							WithdrawalCredentials: make([]byte, 32),
						},
						Index: 0,
					},
					{
						Validator: &ethpb.Validator{
							PublicKey:             pubKey(1),
							WithdrawalCredentials: make([]byte, 32),
						},
						Index: 1,
					},
				},
				NextPageToken: strconv.Itoa(1),
				TotalSize:     int32(count)}},
	}
	for _, test := range tests {
		res, err := bs.ListValidators(context.Background(), test.req)
		require.NoError(t, err)
		if !proto.Equal(res, test.res) {
			t.Errorf("Incorrect validator response, wanted %v, received %v", test.res, res)
		}
	}
}

func TestServer_ListValidators_PaginationOutOfRange(t *testing.T) {
	db, _ := dbTest.SetupDB(t)

	count := 1
	validators, _ := setupValidators(t, db, count)
	headState, err := db.HeadState(context.Background())
	require.NoError(t, err)

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{
			// We are in epoch 0.
			Genesis: time.Now(),
		},
		StateGen: stategen.New(db, cache.NewStateSummaryCache()),
	}

	req := &ethpb.ListValidatorsRequest{PageToken: strconv.Itoa(1), PageSize: 100}
	wanted := fmt.Sprintf("page start %d >= list %d", req.PageSize, len(validators))
	_, err = bs.ListValidators(context.Background(), req)
	assert.ErrorContains(t, wanted, err)
}

func TestServer_ListValidators_ExceedsMaxPageSize(t *testing.T) {
	bs := &Server{}
	exceedsMax := int32(cmd.Get().MaxRPCPageSize + 1)

	wanted := fmt.Sprintf("Requested page size %d can not be greater than max size %d", exceedsMax, cmd.Get().MaxRPCPageSize)
	req := &ethpb.ListValidatorsRequest{PageToken: strconv.Itoa(0), PageSize: exceedsMax}
	_, err := bs.ListValidators(context.Background(), req)
	assert.ErrorContains(t, wanted, err)
}

func TestServer_ListValidators_DefaultPageSize(t *testing.T) {
	db, _ := dbTest.SetupDB(t)

	validators, _ := setupValidators(t, db, 1000)
	want := make([]*ethpb.Validators_ValidatorContainer, len(validators))
	for i := 0; i < len(validators); i++ {
		want[i] = &ethpb.Validators_ValidatorContainer{
			Index:     uint64(i),
			Validator: validators[i],
		}
	}
	headState, err := db.HeadState(context.Background())
	require.NoError(t, err)

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{
				Epoch: 0,
			},
		},
		GenesisTimeFetcher: &mock.ChainService{
			// We are in epoch 0.
			Genesis: time.Now(),
		},
		StateGen: stategen.New(db, cache.NewStateSummaryCache()),
	}

	req := &ethpb.ListValidatorsRequest{}
	res, err := bs.ListValidators(context.Background(), req)
	require.NoError(t, err)

	i := 0
	j := params.BeaconConfig().DefaultPageSize
	assert.DeepEqual(t, want[i:j], res.ValidatorList, "Incorrect respond of validators")
}

func TestServer_ListValidators_FromOldEpoch(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	numEpochs := 30
	validators := make([]*ethpb.Validator, numEpochs)
	for i := 0; i < numEpochs; i++ {
		validators[i] = &ethpb.Validator{
			ActivationEpoch:       uint64(i),
			PublicKey:             make([]byte, 48),
			WithdrawalCredentials: make([]byte, 32),
		}
	}
	want := make([]*ethpb.Validators_ValidatorContainer, len(validators))
	for i := 0; i < len(validators); i++ {
		want[i] = &ethpb.Validators_ValidatorContainer{
			Index:     uint64(i),
			Validator: validators[i],
		}
	}

	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(helpers.StartSlot(30)))
	require.NoError(t, st.SetValidators(validators))
	b := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, b))
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, st, gRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	secondsPerEpoch := params.BeaconConfig().SecondsPerSlot * params.BeaconConfig().SlotsPerEpoch
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: st,
		},
		GenesisTimeFetcher: &mock.ChainService{
			// We are in epoch 30
			Genesis: time.Now().Add(time.Duration(-1*int64(30*secondsPerEpoch)) * time.Second),
		},
		StateGen: stategen.New(db, cache.NewStateSummaryCache()),
	}

	req := &ethpb.ListValidatorsRequest{
		QueryFilter: &ethpb.ListValidatorsRequest_Genesis{
			Genesis: true,
		},
	}
	res, err := bs.ListValidators(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 1, len(res.ValidatorList))

	req = &ethpb.ListValidatorsRequest{
		QueryFilter: &ethpb.ListValidatorsRequest_Epoch{
			Epoch: 20,
		},
	}
	res, err = bs.ListValidators(context.Background(), req)
	require.NoError(t, err)
	assert.DeepEqual(t, want[:21], res.ValidatorList, "Incorrect number of validators")
}

func TestServer_ListValidators_ProcessHeadStateSlots(t *testing.T) {
	db, _ := dbTest.SetupDB(t)
	ctx := context.Background()

	headSlot := uint64(32)
	numValidators := params.BeaconConfig().MinGenesisActiveValidatorCount
	validators := make([]*ethpb.Validator, numValidators)
	balances := make([]uint64, numValidators)
	for i := uint64(0); i < numValidators; i++ {
		validators[i] = &ethpb.Validator{
			ActivationEpoch:       0,
			PublicKey:             make([]byte, 48),
			WithdrawalCredentials: make([]byte, 32),
			EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}
	want := make([]*ethpb.Validators_ValidatorContainer, len(validators))
	for i := 0; i < len(validators); i++ {
		want[i] = &ethpb.Validators_ValidatorContainer{
			Index:     uint64(i),
			Validator: validators[i],
		}
	}

	st := testutil.NewBeaconState()
	require.NoError(t, st.SetSlot(headSlot))
	require.NoError(t, st.SetValidators(validators))
	require.NoError(t, st.SetBalances(balances))
	b := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, b))
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, st, gRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	secondsPerEpoch := params.BeaconConfig().SecondsPerSlot * params.BeaconConfig().SlotsPerEpoch
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: st,
		},
		GenesisTimeFetcher: &mock.ChainService{
			Genesis: time.Now().Add(time.Duration(-1*int64(secondsPerEpoch)) * time.Second),
		},
		StateGen: stategen.New(db, cache.NewStateSummaryCache()),
	}

	req := &ethpb.ListValidatorsRequest{
		QueryFilter: &ethpb.ListValidatorsRequest_Epoch{
			Epoch: 1,
		},
	}
	res, err := bs.ListValidators(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, len(want), len(res.ValidatorList), "Incorrect number of validators")
	for i := 0; i < len(res.ValidatorList); i++ {
		assert.DeepEqual(t, want[i], res.ValidatorList[i])
	}
}

func TestServer_GetValidator(t *testing.T) {
	count := 30
	validators := make([]*ethpb.Validator, count)
	for i := 0; i < count; i++ {
		validators[i] = &ethpb.Validator{
			ActivationEpoch:       uint64(i),
			PublicKey:             pubKey(uint64(i)),
			WithdrawalCredentials: make([]byte, 32),
		}
	}

	st := testutil.NewBeaconState()
	require.NoError(t, st.SetValidators(validators))

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: st,
		},
	}

	tests := []struct {
		req     *ethpb.GetValidatorRequest
		res     *ethpb.Validator
		wantErr bool
		err     string
	}{
		{
			req: &ethpb.GetValidatorRequest{
				QueryFilter: &ethpb.GetValidatorRequest_Index{
					Index: 0,
				},
			},
			res:     validators[0],
			wantErr: false,
		},
		{
			req: &ethpb.GetValidatorRequest{
				QueryFilter: &ethpb.GetValidatorRequest_Index{
					Index: uint64(count - 1),
				},
			},
			res:     validators[count-1],
			wantErr: false,
		},
		{
			req: &ethpb.GetValidatorRequest{
				QueryFilter: &ethpb.GetValidatorRequest_PublicKey{
					PublicKey: pubKey(5),
				},
			},
			res:     validators[5],
			wantErr: false,
		},
		{
			req: &ethpb.GetValidatorRequest{
				QueryFilter: &ethpb.GetValidatorRequest_PublicKey{
					PublicKey: []byte("bad-keyxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"),
				},
			},
			res:     nil,
			wantErr: true,
			err:     "No validator matched filter criteria",
		},
		{
			req: &ethpb.GetValidatorRequest{
				QueryFilter: &ethpb.GetValidatorRequest_Index{
					Index: uint64(len(validators)),
				},
			},
			res:     nil,
			wantErr: true,
			err:     fmt.Sprintf("there are only %d validators", len(validators)),
		},
	}

	for _, test := range tests {
		res, err := bs.GetValidator(context.Background(), test.req)
		if test.wantErr && err != nil {
			if !strings.Contains(err.Error(), test.err) {
				t.Fatalf("Wanted %v, received %v", test.err, err)
			}
		} else if err != nil {
			t.Fatal(err)
		}
		assert.DeepEqual(t, test.res, res)
	}
}

func TestServer_GetValidatorActiveSetChanges(t *testing.T) {
	db, sc := dbTest.SetupDB(t)
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	ctx := context.Background()
	validators := make([]*ethpb.Validator, 8)
	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetSlot(0))
	require.NoError(t, headState.SetValidators(validators))
	for i := 0; i < len(validators); i++ {
		activationEpoch := params.BeaconConfig().FarFutureEpoch
		withdrawableEpoch := params.BeaconConfig().FarFutureEpoch
		exitEpoch := params.BeaconConfig().FarFutureEpoch
		slashed := false
		balance := params.BeaconConfig().MaxEffectiveBalance
		// Mark indices divisible by two as activated.
		if i%2 == 0 {
			activationEpoch = 0
		} else if i%3 == 0 {
			// Mark indices divisible by 3 as slashed.
			withdrawableEpoch = params.BeaconConfig().EpochsPerSlashingsVector
			slashed = true
		} else if i%5 == 0 {
			// Mark indices divisible by 5 as exited.
			exitEpoch = 0
			withdrawableEpoch = params.BeaconConfig().MinValidatorWithdrawabilityDelay
		} else if i%7 == 0 {
			// Mark indices divisible by 7 as ejected.
			exitEpoch = 0
			withdrawableEpoch = params.BeaconConfig().MinValidatorWithdrawabilityDelay
			balance = params.BeaconConfig().EjectionBalance
		}
		err := headState.UpdateValidatorAtIndex(uint64(i), &ethpb.Validator{
			ActivationEpoch:       activationEpoch,
			PublicKey:             pubKey(uint64(i)),
			EffectiveBalance:      balance,
			WithdrawalCredentials: make([]byte, 32),
			WithdrawableEpoch:     withdrawableEpoch,
			Slashed:               slashed,
			ExitEpoch:             exitEpoch,
		})
		require.NoError(t, err)
	}
	b := testutil.NewBeaconBlock()
	require.NoError(t, db.SaveBlock(ctx, b))

	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	require.NoError(t, db.SaveState(ctx, headState, gRoot))

	bs := &Server{
		FinalizationFetcher: &mock.ChainService{
			FinalizedCheckPoint: &ethpb.Checkpoint{Epoch: 0, Root: make([]byte, 32)},
		},
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
	}
	res, err := bs.GetValidatorActiveSetChanges(ctx, &ethpb.GetValidatorActiveSetChangesRequest{
		QueryFilter: &ethpb.GetValidatorActiveSetChangesRequest_Genesis{Genesis: true},
	})
	require.NoError(t, err)
	wantedActive := [][]byte{
		pubKey(0),
		pubKey(2),
		pubKey(4),
		pubKey(6),
	}
	wantedActiveIndices := []uint64{0, 2, 4, 6}
	wantedExited := [][]byte{
		pubKey(5),
	}
	wantedExitedIndices := []uint64{5}
	wantedSlashed := [][]byte{
		pubKey(3),
	}
	wantedSlashedIndices := []uint64{3}
	wantedEjected := [][]byte{
		pubKey(7),
	}
	wantedEjectedIndices := []uint64{7}
	wanted := &ethpb.ActiveSetChanges{
		Epoch:               0,
		ActivatedPublicKeys: wantedActive,
		ActivatedIndices:    wantedActiveIndices,
		ExitedPublicKeys:    wantedExited,
		ExitedIndices:       wantedExitedIndices,
		SlashedPublicKeys:   wantedSlashed,
		SlashedIndices:      wantedSlashedIndices,
		EjectedPublicKeys:   wantedEjected,
		EjectedIndices:      wantedEjectedIndices,
	}
	if !proto.Equal(wanted, res) {
		t.Errorf("Wanted \n%v, received \n%v", wanted, res)
	}
}

func TestServer_GetValidatorQueue_PendingActivation(t *testing.T) {
	headState, err := stateTrie.InitializeFromProto(&pbp2p.BeaconState{
		Validators: []*ethpb.Validator{
			{
				ActivationEpoch:            helpers.ActivationExitEpoch(0),
				ActivationEligibilityEpoch: 3,
				PublicKey:                  pubKey(3),
				WithdrawalCredentials:      make([]byte, 32),
			},
			{
				ActivationEpoch:            helpers.ActivationExitEpoch(0),
				ActivationEligibilityEpoch: 2,
				PublicKey:                  pubKey(2),
				WithdrawalCredentials:      make([]byte, 32),
			},
			{
				ActivationEpoch:            helpers.ActivationExitEpoch(0),
				ActivationEligibilityEpoch: 1,
				PublicKey:                  pubKey(1),
				WithdrawalCredentials:      make([]byte, 32),
			},
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
		},
	})
	require.NoError(t, err)
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}
	res, err := bs.GetValidatorQueue(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	// We verify the keys are properly sorted by the validators' activation eligibility epoch.
	wanted := [][]byte{
		pubKey(1),
		pubKey(2),
		pubKey(3),
	}
	activeValidatorCount, err := helpers.ActiveValidatorCount(headState, helpers.CurrentEpoch(headState))
	require.NoError(t, err)
	wantChurn, err := helpers.ValidatorChurnLimit(activeValidatorCount)
	require.NoError(t, err)
	assert.Equal(t, wantChurn, res.ChurnLimit)
	assert.DeepEqual(t, wanted, res.ActivationPublicKeys)
	wantedActiveIndices := []uint64{2, 1, 0}
	assert.DeepEqual(t, wantedActiveIndices, res.ActivationValidatorIndices)
}

func TestServer_GetValidatorQueue_ExitedValidatorLeavesQueue(t *testing.T) {
	validators := []*ethpb.Validator{
		{
			ActivationEpoch:   0,
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey:         []byte("1"),
		},
		{
			ActivationEpoch:   0,
			ExitEpoch:         4,
			WithdrawableEpoch: 6,
			PublicKey:         []byte("2"),
		},
	}

	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetValidators(validators))
	require.NoError(t, headState.SetFinalizedCheckpoint(&ethpb.Checkpoint{Epoch: 0}))
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	// First we check if validator with index 1 is in the exit queue.
	res, err := bs.GetValidatorQueue(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	wanted := [][]byte{
		[]byte("2"),
	}
	activeValidatorCount, err := helpers.ActiveValidatorCount(headState, helpers.CurrentEpoch(headState))
	require.NoError(t, err)
	wantChurn, err := helpers.ValidatorChurnLimit(activeValidatorCount)
	require.NoError(t, err)
	assert.Equal(t, wantChurn, res.ChurnLimit)
	assert.DeepEqual(t, wanted, res.ExitPublicKeys)
	wantedExitIndices := []uint64{1}
	assert.DeepEqual(t, wantedExitIndices, res.ExitValidatorIndices)

	// Now, we move the state.slot past the exit epoch of the validator, and now
	// the validator should no longer exist in the queue.
	require.NoError(t, headState.SetSlot(helpers.StartSlot(validators[1].ExitEpoch+1)))
	res, err = bs.GetValidatorQueue(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(res.ExitPublicKeys))
}

func TestServer_GetValidatorQueue_PendingExit(t *testing.T) {
	headState, err := stateTrie.InitializeFromProto(&pbp2p.BeaconState{
		Validators: []*ethpb.Validator{
			{
				ActivationEpoch:       0,
				ExitEpoch:             4,
				WithdrawableEpoch:     3,
				PublicKey:             pubKey(3),
				WithdrawalCredentials: make([]byte, 32),
			},
			{
				ActivationEpoch:       0,
				ExitEpoch:             4,
				WithdrawableEpoch:     2,
				PublicKey:             pubKey(2),
				WithdrawalCredentials: make([]byte, 32),
			},
			{
				ActivationEpoch:       0,
				ExitEpoch:             4,
				WithdrawableEpoch:     1,
				PublicKey:             pubKey(1),
				WithdrawalCredentials: make([]byte, 32),
			},
		},
		FinalizedCheckpoint: &ethpb.Checkpoint{
			Epoch: 0,
		},
	})
	require.NoError(t, err)
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}
	res, err := bs.GetValidatorQueue(context.Background(), &ptypes.Empty{})
	require.NoError(t, err)
	// We verify the keys are properly sorted by the validators' withdrawable epoch.
	wanted := [][]byte{
		pubKey(1),
		pubKey(2),
		pubKey(3),
	}
	activeValidatorCount, err := helpers.ActiveValidatorCount(headState, helpers.CurrentEpoch(headState))
	require.NoError(t, err)
	wantChurn, err := helpers.ValidatorChurnLimit(activeValidatorCount)
	require.NoError(t, err)
	assert.Equal(t, wantChurn, res.ChurnLimit)
	assert.DeepEqual(t, wanted, res.ExitPublicKeys)
}

func TestServer_GetValidatorParticipation_CannotRequestFutureEpoch(t *testing.T) {
	db, _ := dbTest.SetupDB(t)

	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: false})
	defer resetCfg()

	ctx := context.Background()
	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetSlot(0))
	bs := &Server{
		BeaconDB: db,
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, cache.NewStateSummaryCache()),
	}

	wanted := "Cannot retrieve information about an epoch"
	_, err := bs.GetValidatorParticipation(
		ctx,
		&ethpb.GetValidatorParticipationRequest{
			QueryFilter: &ethpb.GetValidatorParticipationRequest_Epoch{
				Epoch: helpers.SlotToEpoch(bs.GenesisTimeFetcher.CurrentSlot()) + 1,
			},
		},
	)
	assert.ErrorContains(t, wanted, err)
}

func TestServer_GetValidatorParticipation_PrevEpoch(t *testing.T) {
	db, sc := dbTest.SetupDB(t)
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	ctx := context.Background()
	validatorCount := uint64(100)

	validators := make([]*ethpb.Validator, validatorCount)
	balances := make([]uint64, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			PublicKey:             bytesutil.ToBytes(uint64(i), 48),
			WithdrawalCredentials: make([]byte, 32),
			ExitEpoch:             params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:      params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	atts := []*pbp2p.PendingAttestation{{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: make([]byte, 32),
			Source: &ethpb.Checkpoint{
				Root: make([]byte, 32),
			},
			Target: &ethpb.Checkpoint{
				Root: make([]byte, 32),
			},
		},
		InclusionDelay: 1,
	}}
	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetSlot(params.BeaconConfig().SlotsPerEpoch))
	require.NoError(t, headState.SetValidators(validators))
	require.NoError(t, headState.SetBalances(balances))
	require.NoError(t, headState.SetPreviousEpochAttestations(atts))

	b := testutil.NewBeaconBlock()
	b.Block.Slot = params.BeaconConfig().SlotsPerEpoch
	require.NoError(t, db.SaveBlock(ctx, b))
	bRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, headState, bRoot))

	m := &mock.ChainService{State: headState}
	bs := &Server{
		BeaconDB:           db,
		HeadFetcher:        m,
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
	}

	res, err := bs.GetValidatorParticipation(ctx, &ethpb.GetValidatorParticipationRequest{QueryFilter: &ethpb.GetValidatorParticipationRequest_Epoch{Epoch: 0}})
	require.NoError(t, err)

	wanted := &ethpb.ValidatorParticipation{EligibleEther: validatorCount * params.BeaconConfig().MaxEffectiveBalance,
		VotedEther:              params.BeaconConfig().EffectiveBalanceIncrement,
		GlobalParticipationRate: float32(params.BeaconConfig().EffectiveBalanceIncrement) / float32(validatorCount*params.BeaconConfig().MaxEffectiveBalance)}
	assert.DeepEqual(t, wanted, res.Participation, "Incorrect validator participation respond")
}

func TestServer_GetValidatorParticipation_DoesntExist(t *testing.T) {
	db, sc := dbTest.SetupDB(t)
	ctx := context.Background()

	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetSlot(params.BeaconConfig().SlotsPerEpoch))

	b := testutil.NewBeaconBlock()
	b.Block.Slot = params.BeaconConfig().SlotsPerEpoch
	require.NoError(t, db.SaveBlock(ctx, b))
	bRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	require.NoError(t, db.SaveState(ctx, headState, bRoot))

	m := &mock.ChainService{State: headState}
	bs := &Server{
		BeaconDB:           db,
		HeadFetcher:        m,
		GenesisTimeFetcher: &mock.ChainService{},
		StateGen:           stategen.New(db, sc),
	}

	wanted := "Participation information for epoch 0 is not yet available"
	_, err = bs.GetValidatorParticipation(ctx, &ethpb.GetValidatorParticipationRequest{
		QueryFilter: &ethpb.GetValidatorParticipationRequest_Epoch{Epoch: 0},
	})
	if err != nil && !strings.Contains(err.Error(), wanted) {
		t.Errorf("Expected error %v, received %v", wanted, err)
	}
}

func TestGetValidatorPerformance_Syncing(t *testing.T) {
	ctx := context.Background()

	bs := &Server{
		SyncChecker: &mockSync.Sync{IsSyncing: true},
	}

	wanted := "Syncing to latest head, not ready to respond"
	_, err := bs.GetValidatorPerformance(ctx, nil)
	assert.ErrorContains(t, wanted, err)
}

func TestGetValidatorPerformance_OK(t *testing.T) {
	ctx := context.Background()
	epoch := uint64(1)
	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetSlot(helpers.StartSlot(epoch+1)))
	atts := make([]*pb.PendingAttestation, 3)
	for i := 0; i < len(atts); i++ {
		atts[i] = &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			InclusionDelay:  1,
		}
	}
	require.NoError(t, headState.SetPreviousEpochAttestations(atts))
	defaultBal := params.BeaconConfig().MaxEffectiveBalance
	extraBal := params.BeaconConfig().MaxEffectiveBalance + params.BeaconConfig().GweiPerEth
	balances := []uint64{defaultBal, extraBal, extraBal + params.BeaconConfig().GweiPerEth}
	require.NoError(t, headState.SetBalances(balances))
	publicKey1 := bytesutil.ToBytes48([]byte{1})
	publicKey2 := bytesutil.ToBytes48([]byte{2})
	publicKey3 := bytesutil.ToBytes48([]byte{3})
	validators := []*ethpb.Validator{
		{
			PublicKey:       publicKey1[:],
			ActivationEpoch: 5,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
		},
		{
			PublicKey:        publicKey2[:],
			EffectiveBalance: defaultBal,
			ActivationEpoch:  0,
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
		},
		{
			PublicKey:        publicKey3[:],
			EffectiveBalance: defaultBal,
			ActivationEpoch:  0,
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
		},
	}
	require.NoError(t, headState.SetValidators(validators))
	require.NoError(t, headState.SetBalances([]uint64{100, 101, 102}))
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
		GenesisTimeFetcher: &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*int64(headState.Slot()*params.BeaconConfig().SecondsPerSlot)) * time.Second)},
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
	}
	farFuture := params.BeaconConfig().FarFutureEpoch
	want := &ethpb.ValidatorPerformanceResponse{
		PublicKeys:                    [][]byte{publicKey2[:], publicKey3[:]},
		CurrentEffectiveBalances:      []uint64{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance},
		InclusionSlots:                []uint64{farFuture, farFuture},
		InclusionDistances:            []uint64{farFuture, farFuture},
		CorrectlyVotedSource:          []bool{false, false},
		CorrectlyVotedTarget:          []bool{false, false},
		CorrectlyVotedHead:            []bool{false, false},
		BalancesBeforeEpochTransition: []uint64{101, 102},
		BalancesAfterEpochTransition:  []uint64{0, 0},
		MissingValidators:             [][]byte{publicKey1[:]},
	}

	res, err := bs.GetValidatorPerformance(ctx, &ethpb.ValidatorPerformanceRequest{
		PublicKeys: [][]byte{publicKey1[:], publicKey3[:], publicKey2[:]},
	})
	require.NoError(t, err)
	if !proto.Equal(want, res) {
		t.Errorf("Wanted %v\nReceived %v", want, res)
	}
}

func TestGetValidatorPerformance_Indices(t *testing.T) {
	ctx := context.Background()
	epoch := uint64(1)
	defaultBal := params.BeaconConfig().MaxEffectiveBalance
	extraBal := params.BeaconConfig().MaxEffectiveBalance + params.BeaconConfig().GweiPerEth
	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetSlot(helpers.StartSlot(epoch+1)))
	balances := []uint64{defaultBal, extraBal, extraBal + params.BeaconConfig().GweiPerEth}
	require.NoError(t, headState.SetBalances(balances))
	publicKey1 := bytesutil.ToBytes48([]byte{1})
	publicKey2 := bytesutil.ToBytes48([]byte{2})
	publicKey3 := bytesutil.ToBytes48([]byte{3})
	validators := []*ethpb.Validator{
		{
			PublicKey:       publicKey1[:],
			ActivationEpoch: 5,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
		},
		{
			PublicKey:        publicKey2[:],
			EffectiveBalance: defaultBal,
			ActivationEpoch:  0,
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
		},
		{
			PublicKey:        publicKey3[:],
			EffectiveBalance: defaultBal,
			ActivationEpoch:  0,
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
		},
	}
	require.NoError(t, headState.SetValidators(validators))
	bs := &Server{
		HeadFetcher: &mock.ChainService{
			// 10 epochs into the future.
			State: headState,
		},
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
		GenesisTimeFetcher: &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*int64(headState.Slot()*params.BeaconConfig().SecondsPerSlot)) * time.Second)},
	}
	c := headState.Copy()
	vp, bp, err := precompute.New(ctx, c)
	require.NoError(t, err)
	vp, bp, err = precompute.ProcessAttestations(ctx, c, vp, bp)
	require.NoError(t, err)
	c, err = precompute.ProcessRewardsAndPenaltiesPrecompute(c, bp, vp)
	require.NoError(t, err)
	farFuture := params.BeaconConfig().FarFutureEpoch
	want := &ethpb.ValidatorPerformanceResponse{
		PublicKeys:                    [][]byte{publicKey2[:], publicKey3[:]},
		CurrentEffectiveBalances:      []uint64{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance},
		InclusionSlots:                []uint64{farFuture, farFuture},
		InclusionDistances:            []uint64{farFuture, farFuture},
		CorrectlyVotedSource:          []bool{false, false},
		CorrectlyVotedTarget:          []bool{false, false},
		CorrectlyVotedHead:            []bool{false, false},
		BalancesBeforeEpochTransition: []uint64{extraBal, extraBal + params.BeaconConfig().GweiPerEth},
		BalancesAfterEpochTransition:  []uint64{vp[1].AfterEpochTransitionBalance, vp[2].AfterEpochTransitionBalance},
		MissingValidators:             [][]byte{publicKey1[:]},
	}

	res, err := bs.GetValidatorPerformance(ctx, &ethpb.ValidatorPerformanceRequest{
		Indices: []uint64{2, 1, 0},
	})
	require.NoError(t, err)
	if !proto.Equal(want, res) {
		t.Errorf("Wanted %v\nReceived %v", want, res)
	}
}

func TestGetValidatorPerformance_IndicesPubkeys(t *testing.T) {
	ctx := context.Background()
	epoch := uint64(1)
	defaultBal := params.BeaconConfig().MaxEffectiveBalance
	extraBal := params.BeaconConfig().MaxEffectiveBalance + params.BeaconConfig().GweiPerEth
	headState := testutil.NewBeaconState()
	require.NoError(t, headState.SetSlot(helpers.StartSlot(epoch+1)))
	balances := []uint64{defaultBal, extraBal, extraBal + params.BeaconConfig().GweiPerEth}
	require.NoError(t, headState.SetBalances(balances))
	publicKey1 := bytesutil.ToBytes48([]byte{1})
	publicKey2 := bytesutil.ToBytes48([]byte{2})
	publicKey3 := bytesutil.ToBytes48([]byte{3})
	validators := []*ethpb.Validator{
		{
			PublicKey:       publicKey1[:],
			ActivationEpoch: 5,
			ExitEpoch:       params.BeaconConfig().FarFutureEpoch,
		},
		{
			PublicKey:        publicKey2[:],
			EffectiveBalance: defaultBal,
			ActivationEpoch:  0,
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
		},
		{
			PublicKey:        publicKey3[:],
			EffectiveBalance: defaultBal,
			ActivationEpoch:  0,
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
		},
	}
	require.NoError(t, headState.SetValidators(validators))

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			// 10 epochs into the future.
			State: headState,
		},
		SyncChecker:        &mockSync.Sync{IsSyncing: false},
		GenesisTimeFetcher: &mock.ChainService{Genesis: time.Now().Add(time.Duration(-1*int64(headState.Slot()*params.BeaconConfig().SecondsPerSlot)) * time.Second)},
	}
	c := headState.Copy()
	vp, bp, err := precompute.New(ctx, c)
	require.NoError(t, err)
	vp, bp, err = precompute.ProcessAttestations(ctx, c, vp, bp)
	require.NoError(t, err)
	c, err = precompute.ProcessRewardsAndPenaltiesPrecompute(c, bp, vp)
	require.NoError(t, err)
	farFuture := params.BeaconConfig().FarFutureEpoch
	want := &ethpb.ValidatorPerformanceResponse{
		PublicKeys:                    [][]byte{publicKey2[:], publicKey3[:]},
		CurrentEffectiveBalances:      []uint64{params.BeaconConfig().MaxEffectiveBalance, params.BeaconConfig().MaxEffectiveBalance},
		InclusionSlots:                []uint64{farFuture, farFuture},
		InclusionDistances:            []uint64{farFuture, farFuture},
		CorrectlyVotedSource:          []bool{false, false},
		CorrectlyVotedTarget:          []bool{false, false},
		CorrectlyVotedHead:            []bool{false, false},
		BalancesBeforeEpochTransition: []uint64{extraBal, extraBal + params.BeaconConfig().GweiPerEth},
		BalancesAfterEpochTransition:  []uint64{vp[1].AfterEpochTransitionBalance, vp[2].AfterEpochTransitionBalance},
		MissingValidators:             [][]byte{publicKey1[:]},
	}
	// Index 2 and publicKey3 points to the same validator.
	// Should not return duplicates.
	res, err := bs.GetValidatorPerformance(ctx, &ethpb.ValidatorPerformanceRequest{
		PublicKeys: [][]byte{publicKey1[:], publicKey3[:]}, Indices: []uint64{1, 2},
	})
	require.NoError(t, err)
	if !proto.Equal(want, res) {
		t.Errorf("Wanted %v\nReceived %v", want, res)
	}
}

func BenchmarkListValidatorBalances(b *testing.B) {
	b.StopTimer()
	db, _ := dbTest.SetupDB(b)
	ctx := context.Background()

	count := 1000
	setupValidators(b, db, count)

	headState, err := db.HeadState(ctx)
	require.NoError(b, err)

	bs := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}

	req := &ethpb.ListValidatorBalancesRequest{PageSize: 100}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := bs.ListValidatorBalances(ctx, req)
		require.NoError(b, err)
	}
}

func setupValidators(t testing.TB, db db.Database, count int) ([]*ethpb.Validator, []uint64) {
	ctx := context.Background()
	balances := make([]uint64, count)
	validators := make([]*ethpb.Validator, 0, count)
	for i := 0; i < count; i++ {
		pubKey := pubKey(uint64(i))
		balances[i] = uint64(i)
		validators = append(validators, &ethpb.Validator{
			PublicKey:             pubKey,
			WithdrawalCredentials: make([]byte, 32),
		})
	}
	blk := testutil.NewBeaconBlock().Block
	blockRoot, err := blk.HashTreeRoot()
	require.NoError(t, err)
	s := testutil.NewBeaconState()
	require.NoError(t, s.SetValidators(validators))
	require.NoError(t, s.SetBalances(balances))
	require.NoError(t, db.SaveState(context.Background(), s, blockRoot))
	require.NoError(t, db.SaveHeadBlockRoot(ctx, blockRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, blockRoot))
	return validators, balances
}

func TestServer_GetIndividualVotes_RequestFutureSlot(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()
	ds := &Server{GenesisTimeFetcher: &mock.ChainService{}}
	req := &ethpb.IndividualVotesRequest{
		Epoch: helpers.SlotToEpoch(ds.GenesisTimeFetcher.CurrentSlot()) + 1,
	}
	wanted := "Cannot retrieve information about an epoch in the future"
	_, err := ds.GetIndividualVotes(context.Background(), req)
	assert.ErrorContains(t, wanted, err)
}

func TestServer_GetIndividualVotes_ValidatorsDontExist(t *testing.T) {
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	params.UseMinimalConfig()
	defer params.UseMainnetConfig()
	db, sc := dbTest.SetupDB(t)
	ctx := context.Background()

	validators := uint64(64)
	stateWithValidators, _ := testutil.DeterministicGenesisState(t, validators)
	beaconState := testutil.NewBeaconState()
	require.NoError(t, beaconState.SetValidators(stateWithValidators.Validators()))
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))

	b := testutil.NewBeaconBlock()
	b.Block.Slot = params.BeaconConfig().SlotsPerEpoch
	require.NoError(t, db.SaveBlock(ctx, b))
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	gen := stategen.New(db, sc)
	require.NoError(t, gen.SaveState(ctx, gRoot, beaconState))
	require.NoError(t, db.SaveState(ctx, beaconState, gRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	bs := &Server{
		StateGen:           gen,
		GenesisTimeFetcher: &mock.ChainService{},
	}

	// Test non exist public key.
	res, err := bs.GetIndividualVotes(ctx, &ethpb.IndividualVotesRequest{
		PublicKeys: [][]byte{{'a'}},
		Epoch:      0,
	})
	require.NoError(t, err)
	wanted := &ethpb.IndividualVotesRespond{
		IndividualVotes: []*ethpb.IndividualVotesRespond_IndividualVote{
			{PublicKey: []byte{'a'}, ValidatorIndex: ^uint64(0)},
		},
	}
	assert.DeepEqual(t, wanted, res, "Unexpected response")

	// Test non-existent validator index.
	res, err = bs.GetIndividualVotes(ctx, &ethpb.IndividualVotesRequest{
		Indices: []uint64{100},
		Epoch:   0,
	})
	require.NoError(t, err)
	wanted = &ethpb.IndividualVotesRespond{
		IndividualVotes: []*ethpb.IndividualVotesRespond_IndividualVote{
			{ValidatorIndex: 100},
		},
	}
	assert.DeepEqual(t, wanted, res, "Unexpected response")

	// Test both.
	res, err = bs.GetIndividualVotes(ctx, &ethpb.IndividualVotesRequest{
		PublicKeys: [][]byte{{'a'}, {'b'}},
		Indices:    []uint64{100, 101},
		Epoch:      0,
	})
	require.NoError(t, err)
	wanted = &ethpb.IndividualVotesRespond{
		IndividualVotes: []*ethpb.IndividualVotesRespond_IndividualVote{
			{PublicKey: []byte{'a'}, ValidatorIndex: ^uint64(0)},
			{PublicKey: []byte{'b'}, ValidatorIndex: ^uint64(0)},
			{ValidatorIndex: 100},
			{ValidatorIndex: 101},
		},
	}
	assert.DeepEqual(t, wanted, res, "Unexpected response")
}

func TestServer_GetIndividualVotes_Working(t *testing.T) {
	helpers.ClearCache()
	resetCfg := featureconfig.InitWithReset(&featureconfig.Flags{NewStateMgmt: true})
	defer resetCfg()

	params.UseMinimalConfig()
	defer params.UseMainnetConfig()
	db, sc := dbTest.SetupDB(t)
	ctx := context.Background()

	validators := uint64(64)
	stateWithValidators, _ := testutil.DeterministicGenesisState(t, validators)
	beaconState := testutil.NewBeaconState()
	require.NoError(t, beaconState.SetValidators(stateWithValidators.Validators()))
	require.NoError(t, beaconState.SetSlot(params.BeaconConfig().SlotsPerEpoch))

	bf := []byte{0xff}
	att1 := testutil.NewAttestation()
	att1.AggregationBits = bf
	att2 := testutil.NewAttestation()
	att2.AggregationBits = bf
	rt := [32]byte{'A'}
	att1.Data.Target.Root = rt[:]
	att1.Data.BeaconBlockRoot = rt[:]
	br := beaconState.BlockRoots()
	newRt := [32]byte{'B'}
	br[0] = newRt[:]
	require.NoError(t, beaconState.SetBlockRoots(br))
	att2.Data.Target.Root = rt[:]
	att2.Data.BeaconBlockRoot = newRt[:]
	err := beaconState.SetPreviousEpochAttestations([]*pb.PendingAttestation{
		{Data: att1.Data, AggregationBits: bf, InclusionDelay: 1},
	})
	require.NoError(t, err)
	err = beaconState.SetCurrentEpochAttestations([]*pb.PendingAttestation{
		{Data: att2.Data, AggregationBits: bf, InclusionDelay: 1},
	})
	require.NoError(t, err)

	b := testutil.NewBeaconBlock()
	b.Block.Slot = params.BeaconConfig().SlotsPerEpoch
	require.NoError(t, db.SaveBlock(ctx, b))
	gRoot, err := stateutil.BlockRoot(b.Block)
	require.NoError(t, err)
	gen := stategen.New(db, sc)
	require.NoError(t, gen.SaveState(ctx, gRoot, beaconState))
	require.NoError(t, db.SaveState(ctx, beaconState, gRoot))
	require.NoError(t, db.SaveGenesisBlockRoot(ctx, gRoot))
	bs := &Server{
		StateGen:           gen,
		GenesisTimeFetcher: &mock.ChainService{},
	}

	res, err := bs.GetIndividualVotes(ctx, &ethpb.IndividualVotesRequest{
		Indices: []uint64{0, 1},
		Epoch:   0,
	})
	require.NoError(t, err)
	wanted := &ethpb.IndividualVotesRespond{
		IndividualVotes: []*ethpb.IndividualVotesRespond_IndividualVote{
			{ValidatorIndex: 0, PublicKey: beaconState.Validators()[0].PublicKey, IsActiveInCurrentEpoch: true, IsActiveInPreviousEpoch: true,
				CurrentEpochEffectiveBalanceGwei: params.BeaconConfig().MaxEffectiveBalance, InclusionSlot: params.BeaconConfig().FarFutureEpoch, InclusionDistance: params.BeaconConfig().FarFutureEpoch},
			{ValidatorIndex: 1, PublicKey: beaconState.Validators()[1].PublicKey, IsActiveInCurrentEpoch: true, IsActiveInPreviousEpoch: true,
				CurrentEpochEffectiveBalanceGwei: params.BeaconConfig().MaxEffectiveBalance, InclusionSlot: params.BeaconConfig().FarFutureEpoch, InclusionDistance: params.BeaconConfig().FarFutureEpoch},
		},
	}
	assert.DeepEqual(t, wanted, res, "Unexpected response")
}
