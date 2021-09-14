package beacon

import (
	"context"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/synccommittee"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/prysm/v1alpha1/validator"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/testutil"
	"github.com/prysmaticlabs/prysm/encoding/bytes"
	ethpbv2 "github.com/prysmaticlabs/prysm/proto/eth/v2"
	ethpbalpha "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/grpcutils"
	"github.com/prysmaticlabs/prysm/shared/params"
	sharedtestutil "github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	"google.golang.org/grpc"
)

func Test_currentCommitteeIndicesFromState(t *testing.T) {
	st, _ := sharedtestutil.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
	vals := st.Validators()
	wantedCommittee := make([][]byte, params.BeaconConfig().SyncCommitteeSize)
	wantedIndices := make([]types.ValidatorIndex, len(wantedCommittee))
	for i := 0; i < len(wantedCommittee); i++ {
		wantedIndices[i] = types.ValidatorIndex(i)
		wantedCommittee[i] = vals[i].PublicKey
	}
	require.NoError(t, st.SetCurrentSyncCommittee(&ethpbalpha.SyncCommittee{
		Pubkeys:         wantedCommittee,
		AggregatePubkey: bytes.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
	}))

	t.Run("OK", func(t *testing.T) {
		indices, committee, err := currentCommitteeIndicesFromState(st)
		require.NoError(t, err)
		require.DeepEqual(t, wantedIndices, indices)
		require.DeepEqual(t, wantedCommittee, committee.Pubkeys)
	})
	t.Run("validator in committee not found in state", func(t *testing.T) {
		wantedCommittee[0] = bytes.PadTo([]byte("fakepubkey"), 48)
		require.NoError(t, st.SetCurrentSyncCommittee(&ethpbalpha.SyncCommittee{
			Pubkeys:         wantedCommittee,
			AggregatePubkey: bytes.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
		}))
		_, _, err := currentCommitteeIndicesFromState(st)
		require.ErrorContains(t, "index not found for pubkey", err)
	})
}

func Test_extractSyncSubcommittees(t *testing.T) {
	st, _ := sharedtestutil.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
	vals := st.Validators()
	syncCommittee := make([][]byte, params.BeaconConfig().SyncCommitteeSize)
	for i := 0; i < len(syncCommittee); i++ {
		syncCommittee[i] = vals[i].PublicKey
	}
	require.NoError(t, st.SetCurrentSyncCommittee(&ethpbalpha.SyncCommittee{
		Pubkeys:         syncCommittee,
		AggregatePubkey: bytes.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
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
		syncCommittee[0] = bytes.PadTo([]byte("fakepubkey"), 48)
		require.NoError(t, st.SetCurrentSyncCommittee(&ethpbalpha.SyncCommittee{
			Pubkeys:         syncCommittee,
			AggregatePubkey: bytes.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
		}))
		committee, err := st.CurrentSyncCommittee()
		require.NoError(t, err)
		_, err = extractSyncSubcommittees(st, committee)
		require.ErrorContains(t, "index not found for pubkey", err)
	})
}

func TestListSyncCommittees(t *testing.T) {
	ctx := context.Background()
	st, _ := sharedtestutil.DeterministicGenesisStateAltair(t, params.BeaconConfig().SyncCommitteeSize)
	syncCommittee := make([][]byte, params.BeaconConfig().SyncCommitteeSize)
	vals := st.Validators()
	for i := 0; i < len(syncCommittee); i++ {
		syncCommittee[i] = vals[i].PublicKey
	}
	require.NoError(t, st.SetCurrentSyncCommittee(&ethpbalpha.SyncCommittee{
		Pubkeys:         syncCommittee,
		AggregatePubkey: bytes.PadTo([]byte{}, params.BeaconConfig().BLSPubkeyLength),
	}))
	stRoot, err := st.HashTreeRoot(ctx)
	require.NoError(t, err)

	s := &Server{
		StateFetcher: &testutil.MockFetcher{
			BeaconState: st,
		},
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
}

func TestSubmitPoolSyncCommitteeSignatures(t *testing.T) {
	ctx := grpc.NewContextWithServerTransportStream(context.Background(), &runtime.ServerTransportStream{})
	st, _ := sharedtestutil.DeterministicGenesisStateAltair(t, 10)

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
		_, err := s.SubmitPoolSyncCommitteeSignatures(ctx, &ethpbv2.SubmitPoolSyncCommitteeSignatures{
			Data: []*ethpbv2.SyncCommitteeMessage{
				{
					Slot:            0,
					BeaconBlockRoot: []byte("0x" + strings.Repeat("0", 64)),
					ValidatorIndex:  0,
					Signature:       []byte("0x" + strings.Repeat("0", 192)),
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
		v, ok := md[strings.ToLower(grpcutils.CustomErrorMetadataKey)]
		require.Equal(t, true, ok, "could not retrieve custom error metadata value")
		assert.DeepEqual(
			t,
			[]string{"{\"failures\":[{\"index\":0,\"message\":\"invalid block root format\"}]}"},
			v,
		)
	})
}

func TestValidateSyncCommitteeMessage(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		msg := &ethpbv2.SyncCommitteeMessage{
			Slot:            0,
			BeaconBlockRoot: []byte("0x" + strings.Repeat("0", 64)),
			ValidatorIndex:  0,
			Signature:       []byte("0x" + strings.Repeat("0", 192)),
		}
		err := validateSyncCommitteeMessage(msg)
		assert.NoError(t, err)
	})
	t.Run("invalid block root", func(t *testing.T) {
		msg := &ethpbv2.SyncCommitteeMessage{
			Slot:            0,
			BeaconBlockRoot: []byte("invalid"),
			ValidatorIndex:  0,
			Signature:       []byte("0x" + strings.Repeat("0", 192)),
		}
		err := validateSyncCommitteeMessage(msg)
		require.NotNil(t, err)
		assert.ErrorContains(t, "invalid block root format", err)
	})
	t.Run("invalid block root", func(t *testing.T) {
		msg := &ethpbv2.SyncCommitteeMessage{
			Slot:            0,
			BeaconBlockRoot: []byte("0x" + strings.Repeat("0", 64)),
			ValidatorIndex:  0,
			Signature:       []byte("invalid"),
		}
		err := validateSyncCommitteeMessage(msg)
		require.NotNil(t, err)
		assert.ErrorContains(t, "invalid signature format", err)
	})
}
