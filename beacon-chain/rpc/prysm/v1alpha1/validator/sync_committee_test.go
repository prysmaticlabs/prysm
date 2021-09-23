package validator

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/synccommittee"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	testing2 "github.com/prysmaticlabs/prysm/testing"
	"github.com/prysmaticlabs/prysm/testing/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetSyncMessageBlockRoot_OK(t *testing.T) {
	r := []byte{'a'}
	server := &Server{
		HeadFetcher: &mock.ChainService{Root: r},
	}
	res, err := server.GetSyncMessageBlockRoot(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.DeepEqual(t, r, res.Root)
}

func TestSubmitSyncMessage_OK(t *testing.T) {
	st, _ := testing2.DeterministicGenesisStateAltair(t, 10)
	server := &Server{
		SyncCommitteePool: synccommittee.NewStore(),
		P2P:               &mockp2p.MockBroadcaster{},
		HeadFetcher: &mock.ChainService{
			State: st,
		},
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

func TestGetSyncSubcommitteeIndex_Ok(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	params.OverrideBeaconConfig(params.MainnetConfig())
	transition.SkipSlotCache.Disable()
	defer transition.SkipSlotCache.Enable()

	server := &Server{
		HeadFetcher: &mock.ChainService{
			SyncCommitteeIndices: []types.CommitteeIndex{0},
		},
	}
	pubKey := [48]byte{}
	// Request slot 0, should get the index 0 for validator 0.
	res, err := server.GetSyncSubcommitteeIndex(context.Background(), &ethpb.SyncSubcommitteeIndexRequest{
		PublicKey: pubKey[:], Slot: types.Slot(0),
	})
	require.NoError(t, err)
	require.DeepEqual(t, []types.CommitteeIndex{0}, res.Indices)
}

func TestSubmitSignedContributionAndProof_OK(t *testing.T) {
	server := &Server{
		SyncCommitteePool: synccommittee.NewStore(),
		P2P:               &mockp2p.MockBroadcaster{},
	}
	contribution := &ethpb.SignedContributionAndProof{
		Message: &ethpb.ContributionAndProof{
			Contribution: &ethpb.SyncCommitteeContribution{
				Slot:              1,
				SubcommitteeIndex: 2,
			},
		},
	}
	_, err := server.SubmitSignedContributionAndProof(context.Background(), contribution)
	require.NoError(t, err)
	savedMsgs, err := server.SyncCommitteePool.SyncCommitteeContributions(1)
	require.NoError(t, err)
	require.DeepEqual(t, []*ethpb.SyncCommitteeContribution{contribution.Message.Contribution}, savedMsgs)
}
