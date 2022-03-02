package validator

import (
	"context"
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	opfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/operations/synccommittee"
	mockp2p "github.com/prysmaticlabs/prysm/beacon-chain/p2p/testing"
	fieldparams "github.com/prysmaticlabs/prysm/config/fieldparams"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/testing/assert"
	"github.com/prysmaticlabs/prysm/testing/require"
	"github.com/prysmaticlabs/prysm/testing/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGetSyncMessageBlockRoot_OK(t *testing.T) {
	r := []byte{'a'}
	server := &Server{
		HeadFetcher: &mock.ChainService{Root: r},
		TimeFetcher: &mock.ChainService{Genesis: time.Now()},
	}
	res, err := server.GetSyncMessageBlockRoot(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
	require.DeepEqual(t, r, res.Root)
}

func TestGetSyncMessageBlockRoot_Optimistic(t *testing.T) {
	params.SetupTestConfigCleanup(t)
	cfg := params.BeaconConfig()
	cfg.BellatrixForkEpoch = 0
	params.OverrideBeaconConfig(cfg)

	server := &Server{
		HeadFetcher: &mock.ChainService{Optimistic: true},
		TimeFetcher: &mock.ChainService{Genesis: time.Now()},
	}
	_, err := server.GetSyncMessageBlockRoot(context.Background(), &emptypb.Empty{})
	s, ok := status.FromError(err)
	require.Equal(t, true, ok)
	require.DeepEqual(t, codes.Unavailable, s.Code())
	require.ErrorContains(t, " The node is currently optimistic and cannot serve validators", err)

	server = &Server{
		HeadFetcher: &mock.ChainService{Optimistic: false},
		TimeFetcher: &mock.ChainService{Genesis: time.Now()},
	}
	_, err = server.GetSyncMessageBlockRoot(context.Background(), &emptypb.Empty{})
	require.NoError(t, err)
}

func TestSubmitSyncMessage_OK(t *testing.T) {
	st, _ := util.DeterministicGenesisStateAltair(t, 10)
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
	pubKey := [fieldparams.BLSPubkeyLength]byte{}
	// Request slot 0, should get the index 0 for validator 0.
	res, err := server.GetSyncSubcommitteeIndex(context.Background(), &ethpb.SyncSubcommitteeIndexRequest{
		PublicKey: pubKey[:], Slot: types.Slot(0),
	})
	require.NoError(t, err)
	require.DeepEqual(t, []types.CommitteeIndex{0}, res.Indices)
}

func TestGetSyncCommitteeContribution_FiltersDuplicates(t *testing.T) {
	st, _ := util.DeterministicGenesisStateAltair(t, 10)
	server := &Server{
		SyncCommitteePool: synccommittee.NewStore(),
		P2P:               &mockp2p.MockBroadcaster{},
		HeadFetcher: &mock.ChainService{
			State:                st,
			SyncCommitteeIndices: []types.CommitteeIndex{10},
		},
		TimeFetcher: &mock.ChainService{Genesis: time.Now()},
	}
	secKey, err := bls.RandKey()
	require.NoError(t, err)
	sig := secKey.Sign([]byte{'A'}).Marshal()
	msg := &ethpb.SyncCommitteeMessage{
		Slot:           1,
		ValidatorIndex: 2,
		BlockRoot:      make([]byte, 32),
		Signature:      sig,
	}
	_, err = server.SubmitSyncMessage(context.Background(), msg)
	require.NoError(t, err)
	_, err = server.SubmitSyncMessage(context.Background(), msg)
	require.NoError(t, err)
	val, err := st.ValidatorAtIndex(2)
	require.NoError(t, err)

	contr, err := server.GetSyncCommitteeContribution(context.Background(),
		&ethpb.SyncCommitteeContributionRequest{
			Slot:      1,
			PublicKey: val.PublicKey,
			SubnetId:  1})
	require.NoError(t, err)
	assert.DeepEqual(t, sig, contr.Signature)
}

func TestSubmitSignedContributionAndProof_OK(t *testing.T) {
	server := &Server{
		SyncCommitteePool: synccommittee.NewStore(),
		P2P:               &mockp2p.MockBroadcaster{},
		OperationNotifier: (&mock.ChainService{}).OperationNotifier(),
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

func TestSubmitSignedContributionAndProof_Notification(t *testing.T) {
	server := &Server{
		SyncCommitteePool: synccommittee.NewStore(),
		P2P:               &mockp2p.MockBroadcaster{},
		OperationNotifier: (&mock.ChainService{}).OperationNotifier(),
	}

	// Subscribe to operation notifications.
	opChannel := make(chan *feed.Event, 1024)
	opSub := server.OperationNotifier.OperationFeed().Subscribe(opChannel)
	defer opSub.Unsubscribe()

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

	// Ensure the state notification was broadcast.
	notificationFound := false
	for !notificationFound {
		select {
		case event := <-opChannel:
			if event.Type == opfeed.SyncCommitteeContributionReceived {
				notificationFound = true
				data, ok := event.Data.(*opfeed.SyncCommitteeContributionReceivedData)
				assert.Equal(t, true, ok, "Entity is of the wrong type")
				assert.NotNil(t, data.Contribution)
			}
		case <-opSub.Err():
			t.Error("Subscription to state notifier failed")
			return
		}
	}
}
