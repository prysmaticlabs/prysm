package beacon

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"

	mock "github.com/prysmaticlabs/prysm/v4/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/v4/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	ethpbv2 "github.com/prysmaticlabs/prysm/v4/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/testing/assert"
	"github.com/prysmaticlabs/prysm/v4/testing/require"
	"github.com/prysmaticlabs/prysm/v4/testing/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_currentCommitteeIndicesFromState(t *testing.T) {
	st, _ := util.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
	vals := st.Validators()
	wantedCommittee := make([][]byte, params.BeaconConfig().SyncCommitteeSize)
	wantedIndices := make([]primitives.ValidatorIndex, len(wantedCommittee))
	for i := 0; i < len(wantedCommittee); i++ {
		wantedIndices[i] = primitives.ValidatorIndex(i)
		wantedCommittee[i] = vals[i].PublicKey
	}
	require.NoError(t, st.SetCurrentSyncCommittee(&ethpbalpha.SyncCommittee{
		Pubkeys:         wantedCommittee,
		AggregatePubkey: bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
	}))

	t.Run("OK", func(t *testing.T) {
		indices, committee, err := currentCommitteeIndicesFromState(st)
		require.NoError(t, err)
		require.DeepEqual(t, wantedIndices, indices)
		require.DeepEqual(t, wantedCommittee, committee.Pubkeys)
	})
	t.Run("validator in committee not found in state", func(t *testing.T) {
		wantedCommittee[0] = bytesutil.PadTo([]byte("fakepubkey"), 48)
		require.NoError(t, st.SetCurrentSyncCommittee(&ethpbalpha.SyncCommittee{
			Pubkeys:         wantedCommittee,
			AggregatePubkey: bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
		}))
		_, _, err := currentCommitteeIndicesFromState(st)
		require.ErrorContains(t, "index not found for pubkey", err)
	})
}

func Test_nextCommitteeIndicesFromState(t *testing.T) {
	st, _ := util.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
	vals := st.Validators()
	wantedCommittee := make([][]byte, params.BeaconConfig().SyncCommitteeSize)
	wantedIndices := make([]primitives.ValidatorIndex, len(wantedCommittee))
	for i := 0; i < len(wantedCommittee); i++ {
		wantedIndices[i] = primitives.ValidatorIndex(i)
		wantedCommittee[i] = vals[i].PublicKey
	}
	require.NoError(t, st.SetNextSyncCommittee(&ethpbalpha.SyncCommittee{
		Pubkeys:         wantedCommittee,
		AggregatePubkey: bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
	}))

	t.Run("OK", func(t *testing.T) {
		indices, committee, err := nextCommitteeIndicesFromState(st)
		require.NoError(t, err)
		require.DeepEqual(t, wantedIndices, indices)
		require.DeepEqual(t, wantedCommittee, committee.Pubkeys)
	})
	t.Run("validator in committee not found in state", func(t *testing.T) {
		wantedCommittee[0] = bytesutil.PadTo([]byte("fakepubkey"), 48)
		require.NoError(t, st.SetNextSyncCommittee(&ethpbalpha.SyncCommittee{
			Pubkeys:         wantedCommittee,
			AggregatePubkey: bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
		}))
		_, _, err := nextCommitteeIndicesFromState(st)
		require.ErrorContains(t, "index not found for pubkey", err)
	})
}

func Test_extractSyncSubcommittees(t *testing.T) {
	st, _ := util.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
	vals := st.Validators()
	syncCommittee := make([][]byte, params.BeaconConfig().SyncCommitteeSize)
	for i := 0; i < len(syncCommittee); i++ {
		syncCommittee[i] = vals[i].PublicKey
	}
	require.NoError(t, st.SetCurrentSyncCommittee(&ethpbalpha.SyncCommittee{
		Pubkeys:         syncCommittee,
		AggregatePubkey: bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
	}))

	commSize := params.BeaconConfig().SyncCommitteeSize
	subCommSize := params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount
	wantedSubcommitteeValidators := make([][]primitives.ValidatorIndex, 0)

	for i := uint64(0); i < commSize; i += subCommSize {
		sub := make([]primitives.ValidatorIndex, 0)
		start := i
		end := i + subCommSize
		if end > commSize {
			end = commSize
		}
		for j := start; j < end; j++ {
			sub = append(sub, primitives.ValidatorIndex(j))
		}
		wantedSubcommitteeValidators = append(wantedSubcommitteeValidators, sub)
	}

	t.Run("OK", func(t *testing.T) {
		committee, err := st.CurrentSyncCommittee()
		require.NoError(t, err)
		subcommittee, err := extractSyncSubcommittees(st, committee)
		require.NoError(t, err)
		for i, got := range subcommittee {
			want := wantedSubcommitteeValidators[i]
			require.DeepEqual(t, want, got.Validators)
		}
	})
	t.Run("validator in subcommittee not found in state", func(t *testing.T) {
		syncCommittee[0] = bytesutil.PadTo([]byte("fakepubkey"), 48)
		require.NoError(t, st.SetCurrentSyncCommittee(&ethpbalpha.SyncCommittee{
			Pubkeys:         syncCommittee,
			AggregatePubkey: bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
		}))
		committee, err := st.CurrentSyncCommittee()
		require.NoError(t, err)
		_, err = extractSyncSubcommittees(st, committee)
		require.ErrorContains(t, "index not found for pubkey", err)
	})
}

func TestListSyncCommittees(t *testing.T) {
	ctx := context.Background()
	st, _ := util.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
	syncCommittee := make([][]byte, params.BeaconConfig().SyncCommitteeSize)
	vals := st.Validators()
	for i := 0; i < len(syncCommittee); i++ {
		syncCommittee[i] = vals[i].PublicKey
	}
	require.NoError(t, st.SetCurrentSyncCommittee(&ethpbalpha.SyncCommittee{
		Pubkeys:         syncCommittee,
		AggregatePubkey: bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
	}))
	stRoot, err := st.HashTreeRoot(ctx)
	require.NoError(t, err)
	db := dbTest.SetupDB(t)

	stSlot := st.Slot()
	chainService := &mock.ChainService{Slot: &stSlot}
	s := &Server{
		GenesisTimeFetcher: &testutil.MockGenesisTimeFetcher{
			Genesis: time.Now(),
		},
		Stater: &testutil.MockStater{
			BeaconState: st,
		},
		HeadFetcher:           chainService,
		OptimisticModeFetcher: chainService,
		FinalizationFetcher:   chainService,
		BeaconDB:              db,
		ChainInfoFetcher:      chainService,
	}
	req := &ethpbv2.StateSyncCommitteesRequest{StateId: stRoot[:]}
	resp, err := s.ListSyncCommittees(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	committeeVals := resp.Data.Validators
	require.NotNil(t, committeeVals)
	require.Equal(t, params.BeaconConfig().SyncCommitteeSize, uint64(len(committeeVals)), "incorrect committee size")
	for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSize; i++ {
		assert.Equal(t, primitives.ValidatorIndex(i), committeeVals[i])
	}
	require.NotNil(t, resp.Data.ValidatorAggregates)
	assert.Equal(t, params.BeaconConfig().SyncCommitteeSubnetCount, uint64(len(resp.Data.ValidatorAggregates)))
	for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSubnetCount; i++ {
		vStartIndex := primitives.ValidatorIndex(params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount * i)
		vEndIndex := primitives.ValidatorIndex(params.BeaconConfig().SyncCommitteeSize/params.BeaconConfig().SyncCommitteeSubnetCount*(i+1) - 1)
		j := 0
		for vIndex := vStartIndex; vIndex <= vEndIndex; vIndex++ {
			assert.Equal(t, vIndex, resp.Data.ValidatorAggregates[i].Validators[j])
			j++
		}
	}

	t.Run("execution optimistic", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		stSlot := st.Slot()
		chainService := &mock.ChainService{Optimistic: true, Slot: &stSlot}
		s := &Server{
			GenesisTimeFetcher: &testutil.MockGenesisTimeFetcher{
				Genesis: time.Now(),
			},
			Stater: &testutil.MockStater{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
			BeaconDB:              db,
			ChainInfoFetcher:      chainService,
		}
		resp, err := s.ListSyncCommittees(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
	})

	t.Run("finalized", func(t *testing.T) {
		parentRoot := [32]byte{'a'}
		blk := util.NewBeaconBlock()
		blk.Block.ParentRoot = parentRoot[:]
		root, err := blk.Block.HashTreeRoot()
		require.NoError(t, err)
		util.SaveBlock(t, ctx, db, blk)
		require.NoError(t, db.SaveGenesisBlockRoot(ctx, root))

		headerRoot, err := st.LatestBlockHeader().HashTreeRoot()
		require.NoError(t, err)
		stSlot := st.Slot()
		chainService := &mock.ChainService{
			FinalizedRoots: map[[32]byte]bool{
				headerRoot: true,
			},
			Slot: &stSlot,
		}
		s := &Server{
			GenesisTimeFetcher: &testutil.MockGenesisTimeFetcher{
				Genesis: time.Now(),
			},
			Stater: &testutil.MockStater{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			FinalizationFetcher:   chainService,
			BeaconDB:              db,
			ChainInfoFetcher:      chainService,
		}
		resp, err := s.ListSyncCommittees(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, true, resp.Finalized)
	})
}

type futureSyncMockFetcher struct {
	BeaconState     state.BeaconState
	BeaconStateRoot []byte
}

func (m *futureSyncMockFetcher) State(_ context.Context, stateId []byte) (state.BeaconState, error) {
	expectedRequest := []byte(strconv.FormatUint(uint64(0), 10))
	res := bytes.Compare(stateId, expectedRequest)
	if res != 0 {
		return nil, status.Errorf(
			codes.Internal,
			"Requested wrong epoch for next sync committee, expected: %#x, received: %#x",
			expectedRequest,
			stateId,
		)
	}
	return m.BeaconState, nil
}
func (m *futureSyncMockFetcher) StateRoot(context.Context, []byte) ([]byte, error) {
	return m.BeaconStateRoot, nil
}

func (m *futureSyncMockFetcher) StateBySlot(context.Context, primitives.Slot) (state.BeaconState, error) {
	return m.BeaconState, nil
}

func TestListSyncCommitteesFuture(t *testing.T) {
	ctx := context.Background()
	st, _ := util.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
	syncCommittee := make([][]byte, params.BeaconConfig().SyncCommitteeSize)
	vals := st.Validators()
	for i := 0; i < len(syncCommittee); i++ {
		syncCommittee[i] = vals[i].PublicKey
	}
	require.NoError(t, st.SetNextSyncCommittee(&ethpbalpha.SyncCommittee{
		Pubkeys:         syncCommittee,
		AggregatePubkey: bytesutil.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
	}))
	db := dbTest.SetupDB(t)

	chainService := &mock.ChainService{}
	s := &Server{
		GenesisTimeFetcher: &testutil.MockGenesisTimeFetcher{
			Genesis: time.Now(),
		},
		Stater: &futureSyncMockFetcher{
			BeaconState: st,
		},
		HeadFetcher:           chainService,
		OptimisticModeFetcher: chainService,
		FinalizationFetcher:   chainService,
		BeaconDB:              db,
	}
	req := &ethpbv2.StateSyncCommitteesRequest{StateId: []byte("head")}
	epoch := 2 * params.BeaconConfig().EpochsPerSyncCommitteePeriod
	req.Epoch = &epoch
	_, err := s.ListSyncCommittees(ctx, req)
	require.ErrorContains(t, "Could not fetch sync committee too far in the future", err)

	epoch = 2*params.BeaconConfig().EpochsPerSyncCommitteePeriod - 1
	resp, err := s.ListSyncCommittees(ctx, req)
	require.NoError(t, err)

	require.NotNil(t, resp.Data)
	committeeVals := resp.Data.Validators
	require.NotNil(t, committeeVals)
	require.Equal(t, params.BeaconConfig().SyncCommitteeSize, uint64(len(committeeVals)), "incorrect committee size")
	for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSize; i++ {
		assert.Equal(t, primitives.ValidatorIndex(i), committeeVals[i])
	}
	require.NotNil(t, resp.Data.ValidatorAggregates)
	assert.Equal(t, params.BeaconConfig().SyncCommitteeSubnetCount, uint64(len(resp.Data.ValidatorAggregates)))
	for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSubnetCount; i++ {
		vStartIndex := primitives.ValidatorIndex(params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount * i)
		vEndIndex := primitives.ValidatorIndex(params.BeaconConfig().SyncCommitteeSize/params.BeaconConfig().SyncCommitteeSubnetCount*(i+1) - 1)
		j := 0
		for vIndex := vStartIndex; vIndex <= vEndIndex; vIndex++ {
			assert.Equal(t, vIndex, resp.Data.ValidatorAggregates[i].Validators[j])
			j++
		}
	}
}
