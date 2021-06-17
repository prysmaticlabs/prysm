package validator

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/synccommittee"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGetSyncMessageBlockRoot_UseHeadBlockRoot(t *testing.T) {
	testutil.ResetCache()
	headState, _ := testutil.DeterministicGenesisStateAltair(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	require.NoError(t, headState.SetSlot(100))
	r := []byte{'a'}
	server := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
			Root:  r,
		},
	}
	res, err := server.GetSyncMessageBlockRoot(context.Background(), &ethpb.SyncMessageBlockRootRequest{
		Slot: headState.Slot() + 1,
	})
	require.NoError(t, err)
	require.DeepEqual(t, r, res.Root)
}

func TestGetSyncMessageBlockRoot_UseBlockRootInState(t *testing.T) {
	testutil.ResetCache()
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	headState, _ := testutil.DeterministicGenesisStateAltair(t, params.BeaconConfig().MaxValidatorsPerCommittee)
	require.NoError(t, headState.SetSlot(100))
	server := &Server{
		HeadFetcher: &mock.ChainService{
			State: headState,
		},
	}
	slot := types.Slot(200)
	res, err := server.GetSyncMessageBlockRoot(context.Background(), &ethpb.SyncMessageBlockRootRequest{
		Slot: slot,
	})
	require.NoError(t, err)

	r, err := helpers.BlockRootAtSlot(headState, slot-1)
	require.NoError(t, err)

	require.DeepEqual(t, r, res.Root)
}

func TestSubmitSyncMessage_OK(t *testing.T) {
	server := &Server{
		SyncCommitteePool: synccommittee.NewStore(),
		P2P:               &mockp2p.MockBroadcaster{},
	}
	msg := &ethpb.SyncCommitteeMessage{
		Slot:           1,
		ValidatorIndex: 2,
	}
	_, err := server.SubmitSyncMessage(context.Background(), msg)
	require.NoError(t, err)
	savedMsgs, err := server.SyncCommitteePool.SyncCommitteeMessages(1)
	require.NoError(t, err)
	require.DeepEqual(t, []*ethpb.SyncCommitteeMessage{msg}, savedMsgs)
}
