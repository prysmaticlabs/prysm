package beacon

import (
	"bytes"
	"context"
	"strings"
	"testing"

	chainMock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	rpchelpers "github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/eth/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/statefetcher"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	v1 "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/eth/v1"
	"github.com/prysmaticlabs/prysm/v3/proto/migration"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

func TestGetValidator(t *testing.T) {
	ctx := context.Background()
	db := dbTest.SetupDB(t)

	var st state.BeaconState
	st, _ = util.DeterministicGenesisState(t, 8192)

	t.Run("Head Get Validator by index", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		resp, err := s.GetValidator(ctx, &ethpb.StateValidatorRequest{
			StateId:     []byte("head"),
			ValidatorId: []byte("15"),
		})
		require.NoError(t, err)
		assert.Equal(t, types.ValidatorIndex(15), resp.Data.Index)
	})

	t.Run("Head Get Validator by pubkey", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		pubKey := st.PubkeyAtIndex(types.ValidatorIndex(20))
		resp, err := s.GetValidator(ctx, &ethpb.StateValidatorRequest{
			StateId:     []byte("head"),
			ValidatorId: pubKey[:],
		})
		require.NoError(t, err)
		assert.Equal(t, types.ValidatorIndex(20), resp.Data.Index)
		assert.Equal(t, true, bytes.Equal(pubKey[:], resp.Data.Validator.Pubkey))
	})

	t.Run("Validator ID required", func(t *testing.T) {
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher: &chainMock.ChainService{},
			BeaconDB:    db,
		}
		_, err := s.GetValidator(ctx, &ethpb.StateValidatorRequest{
			StateId: []byte("head"),
		})
		require.ErrorContains(t, "Validator ID is required", err)
	})

	t.Run("execution optimistic", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		chainService := &chainMock.ChainService{Optimistic: true}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}
		resp, err := s.GetValidator(ctx, &ethpb.StateValidatorRequest{
			StateId:     []byte("head"),
			ValidatorId: []byte("15"),
		})
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
}

func TestListValidators(t *testing.T) {
	ctx := context.Background()
	db := dbTest.SetupDB(t)
	var st state.BeaconState
	st, _ = util.DeterministicGenesisState(t, 8192)

	t.Run("Head List All Validators", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, len(resp.Data), 8192)
		for _, val := range resp.Data {
			assert.Equal(t, ethpb.ValidatorStatus_ACTIVE_ONGOING, val.Status)
		}
	})

	t.Run("Head List Validators by index", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		ids := [][]byte{[]byte("15"), []byte("26"), []byte("400")}
		idNums := []types.ValidatorIndex{15, 26, 400}
		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Id:      ids,
		})
		require.NoError(t, err)
		for i, val := range resp.Data {
			assert.Equal(t, idNums[i], val.Index)
			assert.Equal(t, ethpb.ValidatorStatus_ACTIVE_ONGOING, val.Status)
		}
	})

	t.Run("Head List Validators by pubkey", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}
		idNums := []types.ValidatorIndex{20, 66, 90, 100}
		pubkey1 := st.PubkeyAtIndex(types.ValidatorIndex(20))
		pubkey2 := st.PubkeyAtIndex(types.ValidatorIndex(66))
		pubkey3 := st.PubkeyAtIndex(types.ValidatorIndex(90))
		pubkey4 := st.PubkeyAtIndex(types.ValidatorIndex(100))
		pubKeys := [][]byte{pubkey1[:], pubkey2[:], pubkey3[:], pubkey4[:]}
		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Id:      pubKeys,
		})
		require.NoError(t, err)
		for i, val := range resp.Data {
			assert.Equal(t, idNums[i], val.Index)
			assert.Equal(t, true, bytes.Equal(pubKeys[i], val.Validator.Pubkey))
			assert.Equal(t, ethpb.ValidatorStatus_ACTIVE_ONGOING, val.Status)
		}
	})

	t.Run("Head List Validators by both index and pubkey", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		idNums := []types.ValidatorIndex{20, 90, 170, 129}
		pubkey1 := st.PubkeyAtIndex(types.ValidatorIndex(20))
		pubkey2 := st.PubkeyAtIndex(types.ValidatorIndex(90))
		pubkey3 := st.PubkeyAtIndex(types.ValidatorIndex(170))
		pubkey4 := st.PubkeyAtIndex(types.ValidatorIndex(129))
		pubkeys := [][]byte{pubkey1[:], pubkey2[:], pubkey3[:], pubkey4[:]}
		ids := [][]byte{pubkey1[:], []byte("90"), pubkey3[:], []byte("129")}
		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Id:      ids,
		})
		require.NoError(t, err)
		for i, val := range resp.Data {
			assert.Equal(t, idNums[i], val.Index)
			assert.Equal(t, true, bytes.Equal(pubkeys[i], val.Validator.Pubkey))
			assert.Equal(t, ethpb.ValidatorStatus_ACTIVE_ONGOING, val.Status)
		}
	})

	t.Run("Unknown public key is ignored", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		existingKey := st.PubkeyAtIndex(types.ValidatorIndex(1))
		pubkeys := [][]byte{existingKey[:], []byte(strings.Repeat("f", 48))}
		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Id:      pubkeys,
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Data))
		assert.Equal(t, types.ValidatorIndex(1), resp.Data[0].Index)
	})

	t.Run("Unknown index is ignored", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		ids := [][]byte{[]byte("1"), []byte("99999")}
		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Id:      ids,
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(resp.Data))
		assert.Equal(t, types.ValidatorIndex(1), resp.Data[0].Index)
	})

	t.Run("execution optimistic", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		chainService := &chainMock.ChainService{Optimistic: true}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}
		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
}

func TestListValidators_Status(t *testing.T) {
	ctx := context.Background()
	db := dbTest.SetupDB(t)

	var st state.BeaconState
	st, _ = util.DeterministicGenesisState(t, 8192)

	farFutureEpoch := params.BeaconConfig().FarFutureEpoch
	validators := []*eth.Validator{
		// Pending initialized.
		{
			ActivationEpoch:            farFutureEpoch,
			ActivationEligibilityEpoch: farFutureEpoch,
		},
		// Pending queued.
		{
			ActivationEpoch:            10,
			ActivationEligibilityEpoch: 4,
		},
		// Active ongoing.
		{
			ActivationEpoch: 0,
			ExitEpoch:       farFutureEpoch,
		},
		// Active slashed.
		{
			ActivationEpoch: 0,
			ExitEpoch:       30,
			Slashed:         true,
		},
		// Active exiting.
		{
			ActivationEpoch: 3,
			ExitEpoch:       30,
			Slashed:         false,
		},
		// Exit slashed (at epoch 35).
		{
			ActivationEpoch:   3,
			ExitEpoch:         30,
			WithdrawableEpoch: 40,
			Slashed:           true,
		},
		// Exit unslashed (at epoch 35).
		{
			ActivationEpoch:   3,
			ExitEpoch:         30,
			WithdrawableEpoch: 40,
			Slashed:           false,
		},
		// Withdrawable (at epoch 45).
		{
			ActivationEpoch:   3,
			ExitEpoch:         30,
			WithdrawableEpoch: 40,
			EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
			Slashed:           false,
		},
		// Withdrawal done (at epoch 45).
		{
			ActivationEpoch:   3,
			ExitEpoch:         30,
			WithdrawableEpoch: 40,
			EffectiveBalance:  0,
			Slashed:           false,
		},
	}
	for _, validator := range validators {
		require.NoError(t, st.AppendValidator(validator))
		require.NoError(t, st.AppendBalance(params.BeaconConfig().MaxEffectiveBalance))
	}

	t.Run("Head List All ACTIVE Validators", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: st},
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Status:  []ethpb.ValidatorStatus{ethpb.ValidatorStatus_ACTIVE},
		})
		require.NoError(t, err)
		assert.Equal(t, len(resp.Data), 8192+2 /* 2 active */)
		for _, datum := range resp.Data {
			readOnlyVal, err := v1.NewValidator(migration.V1ValidatorToV1Alpha1(datum.Validator))
			require.NoError(t, err)
			status, err := rpchelpers.ValidatorStatus(readOnlyVal, 0)
			require.NoError(t, err)
			require.Equal(
				t,
				true,
				status == ethpb.ValidatorStatus_ACTIVE,
			)
			require.Equal(
				t,
				true,
				datum.Status == ethpb.ValidatorStatus_ACTIVE_ONGOING ||
					datum.Status == ethpb.ValidatorStatus_ACTIVE_EXITING ||
					datum.Status == ethpb.ValidatorStatus_ACTIVE_SLASHED,
			)
		}
	})

	t.Run("Head List All ACTIVE_ONGOING Validators", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: st},
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Status:  []ethpb.ValidatorStatus{ethpb.ValidatorStatus_ACTIVE_ONGOING},
		})
		require.NoError(t, err)
		assert.Equal(t, len(resp.Data), 8192+1 /* 1 active_ongoing */)
		for _, datum := range resp.Data {
			readOnlyVal, err := v1.NewValidator(migration.V1ValidatorToV1Alpha1(datum.Validator))
			require.NoError(t, err)
			status, err := rpchelpers.ValidatorSubStatus(readOnlyVal, 0)
			require.NoError(t, err)
			require.Equal(
				t,
				true,
				status == ethpb.ValidatorStatus_ACTIVE_ONGOING,
			)
			require.Equal(
				t,
				true,
				datum.Status == ethpb.ValidatorStatus_ACTIVE_ONGOING,
			)
		}
	})

	require.NoError(t, st.SetSlot(params.BeaconConfig().SlotsPerEpoch*35))
	t.Run("Head List All EXITED Validators", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: st},
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Status:  []ethpb.ValidatorStatus{ethpb.ValidatorStatus_EXITED},
		})
		require.NoError(t, err)
		assert.Equal(t, 4 /* 4 exited */, len(resp.Data))
		for _, datum := range resp.Data {
			readOnlyVal, err := v1.NewValidator(migration.V1ValidatorToV1Alpha1(datum.Validator))
			require.NoError(t, err)
			status, err := rpchelpers.ValidatorStatus(readOnlyVal, 35)
			require.NoError(t, err)
			require.Equal(
				t,
				true,
				status == ethpb.ValidatorStatus_EXITED,
			)
			require.Equal(
				t,
				true,
				datum.Status == ethpb.ValidatorStatus_EXITED_UNSLASHED || datum.Status == ethpb.ValidatorStatus_EXITED_SLASHED,
			)
		}
	})

	t.Run("Head List All PENDING_INITIALIZED and EXITED_UNSLASHED Validators", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: st},
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Status:  []ethpb.ValidatorStatus{ethpb.ValidatorStatus_PENDING_INITIALIZED, ethpb.ValidatorStatus_EXITED_UNSLASHED},
		})
		require.NoError(t, err)
		assert.Equal(t, 4 /* 4 exited */, len(resp.Data))
		for _, datum := range resp.Data {
			readOnlyVal, err := v1.NewValidator(migration.V1ValidatorToV1Alpha1(datum.Validator))
			require.NoError(t, err)
			status, err := rpchelpers.ValidatorSubStatus(readOnlyVal, 35)
			require.NoError(t, err)
			require.Equal(
				t,
				true,
				status == ethpb.ValidatorStatus_PENDING_INITIALIZED || status == ethpb.ValidatorStatus_EXITED_UNSLASHED,
			)
			require.Equal(
				t,
				true,
				datum.Status == ethpb.ValidatorStatus_PENDING_INITIALIZED || datum.Status == ethpb.ValidatorStatus_EXITED_UNSLASHED,
			)
		}
	})

	t.Run("Head List All PENDING and EXITED Validators", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &statefetcher.StateProvider{
				ChainInfoFetcher: &chainMock.ChainService{State: st},
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		resp, err := s.ListValidators(ctx, &ethpb.StateValidatorsRequest{
			StateId: []byte("head"),
			Status:  []ethpb.ValidatorStatus{ethpb.ValidatorStatus_PENDING, ethpb.ValidatorStatus_EXITED_SLASHED},
		})
		require.NoError(t, err)
		assert.Equal(t, 2 /* 1 pending, 1 exited */, len(resp.Data))
		for _, datum := range resp.Data {
			readOnlyVal, err := v1.NewValidator(migration.V1ValidatorToV1Alpha1(datum.Validator))
			require.NoError(t, err)
			status, err := rpchelpers.ValidatorStatus(readOnlyVal, 35)
			require.NoError(t, err)
			subStatus, err := rpchelpers.ValidatorSubStatus(readOnlyVal, 35)
			require.NoError(t, err)
			require.Equal(
				t,
				true,
				status == ethpb.ValidatorStatus_PENDING || subStatus == ethpb.ValidatorStatus_EXITED_SLASHED,
			)
			require.Equal(
				t,
				true,
				datum.Status == ethpb.ValidatorStatus_PENDING_INITIALIZED || datum.Status == ethpb.ValidatorStatus_EXITED_SLASHED,
			)
		}
	})
}
func TestListValidatorBalances(t *testing.T) {
	ctx := context.Background()
	db := dbTest.SetupDB(t)

	var st state.BeaconState
	count := uint64(8192)
	st, _ = util.DeterministicGenesisState(t, count)
	balances := make([]uint64, count)
	for i := uint64(0); i < count; i++ {
		balances[i] = i
	}
	require.NoError(t, st.SetBalances(balances))

	t.Run("Head List Validators Balance by index", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		ids := [][]byte{[]byte("15"), []byte("26"), []byte("400")}
		idNums := []types.ValidatorIndex{15, 26, 400}
		resp, err := s.ListValidatorBalances(ctx, &ethpb.ValidatorBalancesRequest{
			StateId: []byte("head"),
			Id:      ids,
		})
		require.NoError(t, err)
		for i, val := range resp.Data {
			assert.Equal(t, idNums[i], val.Index)
			assert.Equal(t, balances[val.Index], val.Balance)
		}
	})

	t.Run("Head List Validators Balance by pubkey", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}
		idNums := []types.ValidatorIndex{20, 66, 90, 100}
		pubkey1 := st.PubkeyAtIndex(types.ValidatorIndex(20))
		pubkey2 := st.PubkeyAtIndex(types.ValidatorIndex(66))
		pubkey3 := st.PubkeyAtIndex(types.ValidatorIndex(90))
		pubkey4 := st.PubkeyAtIndex(types.ValidatorIndex(100))
		pubKeys := [][]byte{pubkey1[:], pubkey2[:], pubkey3[:], pubkey4[:]}
		resp, err := s.ListValidatorBalances(ctx, &ethpb.ValidatorBalancesRequest{
			StateId: []byte("head"),
			Id:      pubKeys,
		})
		require.NoError(t, err)
		for i, val := range resp.Data {
			assert.Equal(t, idNums[i], val.Index)
			assert.Equal(t, balances[val.Index], val.Balance)
		}
	})

	t.Run("Head List Validators Balance by both index and pubkey", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		idNums := []types.ValidatorIndex{20, 90, 170, 129}
		pubkey1 := st.PubkeyAtIndex(types.ValidatorIndex(20))
		pubkey3 := st.PubkeyAtIndex(types.ValidatorIndex(170))
		ids := [][]byte{pubkey1[:], []byte("90"), pubkey3[:], []byte("129")}
		resp, err := s.ListValidatorBalances(ctx, &ethpb.ValidatorBalancesRequest{
			StateId: []byte("head"),
			Id:      ids,
		})
		require.NoError(t, err)
		for i, val := range resp.Data {
			assert.Equal(t, idNums[i], val.Index)
			assert.Equal(t, balances[val.Index], val.Balance)
		}
	})

	t.Run("execution optimistic", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		chainService := &chainMock.ChainService{Optimistic: true}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		ids := [][]byte{[]byte("15"), []byte("26"), []byte("400")}
		resp, err := s.ListValidatorBalances(ctx, &ethpb.ValidatorBalancesRequest{
			StateId: []byte("head"),
			Id:      ids,
		})
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
}

func TestListCommittees(t *testing.T) {
	ctx := context.Background()
	db := dbTest.SetupDB(t)

	var st state.BeaconState
	st, _ = util.DeterministicGenesisState(t, 8192)
	epoch := slots.ToEpoch(st.Slot())

	t.Run("Head All Committees", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, int(params.BeaconConfig().SlotsPerEpoch)*2, len(resp.Data))
		for _, datum := range resp.Data {
			assert.Equal(t, true, datum.Index == types.CommitteeIndex(0) || datum.Index == types.CommitteeIndex(1))
			assert.Equal(t, epoch, slots.ToEpoch(datum.Slot))
		}
	})

	t.Run("Head All Committees of Epoch 10", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}
		epoch := types.Epoch(10)
		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
			Epoch:   &epoch,
		})
		require.NoError(t, err)
		for _, datum := range resp.Data {
			assert.Equal(t, true, datum.Slot >= 320 && datum.Slot <= 351)
		}
	})

	t.Run("Head All Committees of Slot 4", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		slot := types.Slot(4)
		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
			Slot:    &slot,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(resp.Data))
		index := types.CommitteeIndex(0)
		for _, datum := range resp.Data {
			assert.Equal(t, epoch, slots.ToEpoch(datum.Slot))
			assert.Equal(t, slot, datum.Slot)
			assert.Equal(t, index, datum.Index)
			index++
		}
	})

	t.Run("Head All Committees of Index 1", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		index := types.CommitteeIndex(1)
		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
			Index:   &index,
		})
		require.NoError(t, err)
		assert.Equal(t, int(params.BeaconConfig().SlotsPerEpoch), len(resp.Data))
		slot := types.Slot(0)
		for _, datum := range resp.Data {
			assert.Equal(t, epoch, slots.ToEpoch(datum.Slot))
			assert.Equal(t, slot, datum.Slot)
			assert.Equal(t, index, datum.Index)
			slot++
		}
	})

	t.Run("Head All Committees of Slot 2, Index 1", func(t *testing.T) {
		chainService := &chainMock.ChainService{}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		index := types.CommitteeIndex(1)
		slot := types.Slot(2)
		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
			Slot:    &slot,
			Index:   &index,
		})
		require.NoError(t, err)
		assert.Equal(t, 1, len(resp.Data))
		for _, datum := range resp.Data {
			assert.Equal(t, epoch, slots.ToEpoch(datum.Slot))
			assert.Equal(t, slot, datum.Slot)
			assert.Equal(t, index, datum.Index)
		}
	})

	t.Run("execution optimistic", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		chainService := &chainMock.ChainService{Optimistic: true}
		s := Server{
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}

		resp, err := s.ListCommittees(ctx, &ethpb.StateCommitteesRequest{
			StateId: []byte("head"),
		})
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})
}
