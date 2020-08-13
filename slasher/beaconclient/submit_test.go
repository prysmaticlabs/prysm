package beaconclient

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/mock"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestService_SubscribeDetectedProposerSlashings(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)

	bs := Service{
		beaconClient:          client,
		proposerSlashingsFeed: new(event.Feed),
	}

	slashing := &ethpb.ProposerSlashing{
		Header_1: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				ProposerIndex: 5,
				Slot:          5,
			},
			Signature: make([]byte, 96),
		},
		Header_2: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				ProposerIndex: 5,
				Slot:          5,
			},
			Signature: make([]byte, 96),
		},
	}

	exitRoutine := make(chan bool)
	slashingsChan := make(chan *ethpb.ProposerSlashing)
	ctx, cancel := context.WithCancel(context.Background())
	client.EXPECT().SubmitProposerSlashing(gomock.Any(), slashing)
	go func(tt *testing.T) {
		bs.subscribeDetectedProposerSlashings(ctx, slashingsChan)
		<-exitRoutine
	}(t)
	slashingsChan <- slashing
	cancel()
	exitRoutine <- true
	require.LogsContain(t, hook, "Context canceled")
}

func TestService_SubscribeDetectedAttesterSlashings(t *testing.T) {
	hook := logTest.NewGlobal()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock.NewMockBeaconChainClient(ctrl)

	bs := Service{
		beaconClient:          client,
		attesterSlashingsFeed: new(event.Feed),
	}

	slashing := &ethpb.AttesterSlashing{
		Attestation_1: &ethpb.IndexedAttestation{
			AttestingIndices: []uint64{1, 2, 3},
			Data: &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{
					Epoch: 3,
				},
				Target: &ethpb.Checkpoint{
					Epoch: 4,
				},
			},
		},
		Attestation_2: &ethpb.IndexedAttestation{
			AttestingIndices: []uint64{3, 4, 5},
			Data:             nil,
		},
	}

	exitRoutine := make(chan bool)
	slashingsChan := make(chan *ethpb.AttesterSlashing)
	ctx, cancel := context.WithCancel(context.Background())
	client.EXPECT().SubmitAttesterSlashing(gomock.Any(), slashing)
	go func(tt *testing.T) {
		bs.subscribeDetectedAttesterSlashings(ctx, slashingsChan)
		<-exitRoutine
	}(t)
	slashingsChan <- slashing
	cancel()
	exitRoutine <- true
	require.LogsContain(t, hook, "Context canceled")
}
