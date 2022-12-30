package beacon

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	grpcutil "github.com/prysmaticlabs/prysm/v3/api/grpc"
	mock "github.com/prysmaticlabs/prysm/v3/beacon-chain/blockchain/testing"
	dbTest "github.com/prysmaticlabs/prysm/v3/beacon-chain/db/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/operations/synccommittee"
	mockp2p "github.com/prysmaticlabs/prysm/v3/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/prysm/v1alpha1/validator"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpbv2 "github.com/prysmaticlabs/prysm/v3/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/assert"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
	"github.com/prysmaticlabs/prysm/v3/testing/util"
	bytesutil2 "github.com/wealdtech/go-bytesutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func Test_currentCommitteeIndicesFromState(t *testing.T) {
	st, _ := util.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
	vals := st.Validators()
	wantedCommittee := make([][]byte, params.BeaconConfig().SyncCommitteeSize)
	wantedIndices := make([]types.ValidatorIndex, len(wantedCommittee))
	for i := 0; i < len(wantedCommittee); i++ {
		wantedIndices[i] = types.ValidatorIndex(i)
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
	wantedIndices := make([]types.ValidatorIndex, len(wantedCommittee))
	for i := 0; i < len(wantedCommittee); i++ {
		wantedIndices[i] = types.ValidatorIndex(i)
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
	wantedSubcommitteeValidators := make([][]types.ValidatorIndex, 0)

	for i := uint64(0); i < commSize; i += subCommSize {
		sub := make([]types.ValidatorIndex, 0)
		start := i
		end := i + subCommSize
		if end > commSize {
			end = commSize
		}
		for j := start; j < end; j++ {
			sub = append(sub, types.ValidatorIndex(j))
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

	chainService := &mock.ChainService{}
	s := &Server{
		GenesisTimeFetcher: &testutil.MockGenesisTimeFetcher{
			Genesis: time.Now(),
		},
		StateFetcher: &testutil.MockFetcher{
			BeaconState: st,
		},
		HeadFetcher:           chainService,
		OptimisticModeFetcher: chainService,
		BeaconDB:              db,
	}
	req := &ethpbv2.StateSyncCommitteesRequest{StateId: stRoot[:]}
	resp, err := s.ListSyncCommittees(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	committeeVals := resp.Data.Validators
	require.NotNil(t, committeeVals)
	require.Equal(t, params.BeaconConfig().SyncCommitteeSize, uint64(len(committeeVals)), "incorrect committee size")
	for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSize; i++ {
		assert.Equal(t, types.ValidatorIndex(i), committeeVals[i])
	}
	require.NotNil(t, resp.Data.ValidatorAggregates)
	assert.Equal(t, params.BeaconConfig().SyncCommitteeSubnetCount, uint64(len(resp.Data.ValidatorAggregates)))
	for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSubnetCount; i++ {
		vStartIndex := types.ValidatorIndex(params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount * i)
		vEndIndex := types.ValidatorIndex(params.BeaconConfig().SyncCommitteeSize/params.BeaconConfig().SyncCommitteeSubnetCount*(i+1) - 1)
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

		chainService := &mock.ChainService{Optimistic: true}
		s := &Server{
			GenesisTimeFetcher: &testutil.MockGenesisTimeFetcher{
				Genesis: time.Now(),
			},
			StateFetcher: &testutil.MockFetcher{
				BeaconState: st,
			},
			HeadFetcher:           chainService,
			OptimisticModeFetcher: chainService,
			BeaconDB:              db,
		}
		resp, err := s.ListSyncCommittees(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, true, resp.ExecutionOptimistic)
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

func (m *futureSyncMockFetcher) StateBySlot(context.Context, types.Slot) (state.BeaconState, error) {
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
		StateFetcher: &futureSyncMockFetcher{
			BeaconState: st,
		},
		HeadFetcher:           chainService,
		OptimisticModeFetcher: chainService,
		BeaconDB:              db,
	}
	req := &ethpbv2.StateSyncCommitteesRequest{}
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
		assert.Equal(t, types.ValidatorIndex(i), committeeVals[i])
	}
	require.NotNil(t, resp.Data.ValidatorAggregates)
	assert.Equal(t, params.BeaconConfig().SyncCommitteeSubnetCount, uint64(len(resp.Data.ValidatorAggregates)))
	for i := uint64(0); i < params.BeaconConfig().SyncCommitteeSubnetCount; i++ {
		vStartIndex := types.ValidatorIndex(params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount * i)
		vEndIndex := types.ValidatorIndex(params.BeaconConfig().SyncCommitteeSize/params.BeaconConfig().SyncCommitteeSubnetCount*(i+1) - 1)
		j := 0
		for vIndex := vStartIndex; vIndex <= vEndIndex; vIndex++ {
			assert.Equal(t, vIndex, resp.Data.ValidatorAggregates[i].Validators[j])
			j++
		}
	}
}

func TestSubmitPoolSyncCommitteeSignatures(t *testing.T) {
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})
	st, _ := util.DeterministicGenesisStateAltair(t, 10)

	alphaServer := &validator.Server{
		SyncCommitteePool: synccommittee.NewStore(),
		P2P:               &mockp2p.MockBroadcaster{},
		HeadFetcher: &mock.ChainService{
			State: st,
		},
	}
	s := &Server{
		V1Alpha1ValidatorServer: alphaServer,
	}

	t.Run("Ok", func(t *testing.T) {
		root, err := bytesutil2.FromHexString("0x" + strings.Repeat("0", 64))
		require.NoError(t, err)
		sig, err := bytesutil2.FromHexString("0x" + strings.Repeat("0", 192))
		require.NoError(t, err)
		_, err = s.SubmitPoolSyncCommitteeSignatures(ctx, &ethpbv2.SubmitPoolSyncCommitteeSignatures{
			Data: []*ethpbv2.SyncCommitteeMessage{
				{
					Slot:            0,
					BeaconBlockRoot: root,
					ValidatorIndex:  0,
					Signature:       sig,
				},
			},
		})
		assert.NoError(t, err)
	})

	t.Run("Invalid message gRPC header", func(t *testing.T) {
		_, err := s.SubmitPoolSyncCommitteeSignatures(ctx, &ethpbv2.SubmitPoolSyncCommitteeSignatures{
			Data: []*ethpbv2.SyncCommitteeMessage{
				{
					Slot:            0,
					BeaconBlockRoot: nil,
					ValidatorIndex:  0,
					Signature:       nil,
				},
			},
		})
		assert.ErrorContains(t, "One or more messages failed validation", err)
		sts, ok := grpc.ServerTransportStreamFromContext(ctx).(*runtime.ServerTransportStream)
		require.Equal(t, true, ok, "type assertion failed")
		md := sts.Header()
		v, ok := md[strings.ToLower(grpcutil.CustomErrorMetadataKey)]
		require.Equal(t, true, ok, "could not retrieve custom error metadata value")
		assert.DeepEqual(
			t,
			[]string{"{\"failures\":[{\"index\":0,\"message\":\"invalid block root length\"}]}"},
			v,
		)
	})
}

func TestValidateSyncCommitteeMessage(t *testing.T) {
	root, err := bytesutil2.FromHexString("0x" + strings.Repeat("0", 64))
	require.NoError(t, err)
	sig, err := bytesutil2.FromHexString("0x" + strings.Repeat("0", 192))
	require.NoError(t, err)
	t.Run("valid", func(t *testing.T) {
		msg := &ethpbv2.SyncCommitteeMessage{
			Slot:            0,
			BeaconBlockRoot: root,
			ValidatorIndex:  0,
			Signature:       sig,
		}
		err := validateSyncCommitteeMessage(msg)
		assert.NoError(t, err)
	})
	t.Run("invalid block root", func(t *testing.T) {
		msg := &ethpbv2.SyncCommitteeMessage{
			Slot:            0,
			BeaconBlockRoot: []byte("invalid"),
			ValidatorIndex:  0,
			Signature:       sig,
		}
		err := validateSyncCommitteeMessage(msg)
		require.NotNil(t, err)
		assert.ErrorContains(t, "invalid block root length", err)
	})
	t.Run("invalid block root", func(t *testing.T) {
		msg := &ethpbv2.SyncCommitteeMessage{
			Slot:            0,
			BeaconBlockRoot: root,
			ValidatorIndex:  0,
			Signature:       []byte("invalid"),
		}
		err := validateSyncCommitteeMessage(msg)
		require.NotNil(t, err)
		assert.ErrorContains(t, "invalid signature length", err)
	})
}
