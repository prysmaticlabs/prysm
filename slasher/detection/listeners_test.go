package detection

import (
	"context"
	"io/ioutil"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations"
	"github.com/prysmaticlabs/prysm/slasher/detection/proposals"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

type mockNotifier struct{}

func (m *mockNotifier) BlockFeed() *event.Feed {
	return new(event.Feed)
}

func (m *mockNotifier) AttestationFeed() *event.Feed {
	return new(event.Feed)
}

func (m *mockNotifier) ClientReadyFeed() *event.Feed {
	return new(event.Feed)
}

func TestService_DetectIncomingBlocks(t *testing.T) {
	hook := logTest.NewGlobal()
	db := testDB.SetupSlasherDB(t, false)
	ds := Service{
		notifier:          &mockNotifier{},
		proposalsDetector: proposals.NewProposeDetector(db),
	}
	blk := &ethpb.SignedBeaconBlock{
		Block:     &ethpb.BeaconBlock{Slot: 1},
		Signature: make([]byte, 96),
	}
	exitRoutine := make(chan bool)
	blocksChan := make(chan *ethpb.SignedBeaconBlock)
	ctx, cancel := context.WithCancel(context.Background())
	go func(tt *testing.T) {
		ds.detectIncomingBlocks(ctx, blocksChan)
		<-exitRoutine
	}(t)
	blocksChan <- blk
	cancel()
	exitRoutine <- true
	require.LogsContain(t, hook, "Context canceled")
}

func TestService_DetectIncomingAttestations(t *testing.T) {
	hook := logTest.NewGlobal()
	ds := Service{
		notifier:              &mockNotifier{},
		minMaxSpanDetector:    &attestations.MockSpanDetector{},
		attesterSlashingsFeed: new(event.Feed),
	}
	att := &ethpb.IndexedAttestation{
		Data: &ethpb.AttestationData{
			Slot: 1,
			Source: &ethpb.Checkpoint{
				Epoch: 0,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 1,
			},
		},
	}
	exitRoutine := make(chan bool)
	attsChan := make(chan *ethpb.IndexedAttestation)
	ctx, cancel := context.WithCancel(context.Background())
	go func(tt *testing.T) {
		ds.detectIncomingAttestations(ctx, attsChan)
		<-exitRoutine
	}(t)
	attsChan <- att
	cancel()
	exitRoutine <- true
	require.LogsContain(t, hook, "Context canceled")
}
