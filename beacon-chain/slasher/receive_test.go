package slasher

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func TestSlasher_receiveAttestations_OK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		serviceCfg: &ServiceConfig{
			IndexedAttestationsFeed: new(event.Feed),
			StateNotifier:           &mock.MockStateNotifier{},
		},
		indexedAttsChan: make(chan *ethpb.IndexedAttestation),
		attsQueue:       newAttestationsQueue(),
	}
	exitChan := make(chan struct{})
	go func() {
		s.receiveAttestations(ctx)
		exitChan <- struct{}{}
	}()
	firstIndices := []uint64{1, 2, 3}
	secondIndices := []uint64{4, 5, 6}
	att1 := createAttestationWrapper(t, 1, 2, firstIndices, nil)
	att2 := createAttestationWrapper(t, 1, 2, secondIndices, nil)
	s.indexedAttsChan <- att1.IndexedAttestation
	s.indexedAttsChan <- att2.IndexedAttestation
	cancel()
	<-exitChan
	wanted := []*slashertypes.IndexedAttestationWrapper{
		att1,
		att2,
	}
	require.DeepEqual(t, wanted, s.attsQueue.dequeue())
}

func TestSlasher_receiveAttestations_OnlyValidAttestations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		serviceCfg: &ServiceConfig{
			IndexedAttestationsFeed: new(event.Feed),
			StateNotifier:           &mock.MockStateNotifier{},
		},
		attsQueue:       newAttestationsQueue(),
		indexedAttsChan: make(chan *ethpb.IndexedAttestation),
	}
	exitChan := make(chan struct{})
	go func() {
		s.receiveAttestations(ctx)
		exitChan <- struct{}{}
	}()
	firstIndices := []uint64{1, 2, 3}
	secondIndices := []uint64{4, 5, 6}
	// Add a valid attestation.
	validAtt := createAttestationWrapper(t, 1, 2, firstIndices, nil)
	s.indexedAttsChan <- validAtt.IndexedAttestation
	// Send an invalid, bad attestation which will not
	// pass integrity checks at it has invalid attestation data.
	s.indexedAttsChan <- &ethpb.IndexedAttestation{
		AttestingIndices: secondIndices,
	}
	cancel()
	<-exitChan
	// Expect only a single, valid attestation was added to the queue.
	require.Equal(t, 1, s.attsQueue.size())
	wanted := []*slashertypes.IndexedAttestationWrapper{
		validAtt,
	}
	require.DeepEqual(t, wanted, s.attsQueue.dequeue())
}

func TestSlasher_receiveBlocks_OK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		serviceCfg: &ServiceConfig{
			BeaconBlockHeadersFeed: new(event.Feed),
			StateNotifier:          &mock.MockStateNotifier{},
		},
		beaconBlockHeadersChan: make(chan *ethpb.SignedBeaconBlockHeader),
		blksQueue:              newBlocksQueue(),
	}
	exitChan := make(chan struct{})
	go func() {
		s.receiveBlocks(ctx)
		exitChan <- struct{}{}
	}()

	block1 := createProposalWrapper(t, 0, 1, nil).SignedBeaconBlockHeader
	block2 := createProposalWrapper(t, 0, 2, nil).SignedBeaconBlockHeader
	s.beaconBlockHeadersChan <- block1
	s.beaconBlockHeadersChan <- block2
	cancel()
	<-exitChan
	wanted := []*slashertypes.SignedBlockHeaderWrapper{
		createProposalWrapper(t, 0, block1.Header.ProposerIndex, nil),
		createProposalWrapper(t, 0, block2.Header.ProposerIndex, nil),
	}
	require.DeepEqual(t, wanted, s.blksQueue.dequeue())
}

func TestService_processQueuedBlocks(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbtest.SetupDB(t)
	s := &Service{
		params: DefaultParams(),
		serviceCfg: &ServiceConfig{
			Database:      beaconDB,
			StateNotifier: &mock.MockStateNotifier{},
		},
		blksQueue: newBlocksQueue(),
	}
	s.blksQueue.extend([]*slashertypes.SignedBlockHeaderWrapper{
		createProposalWrapper(t, 0, 1, nil),
	})
	ctx, cancel := context.WithCancel(context.Background())
	tickerChan := make(chan types.Slot)
	exitChan := make(chan struct{})
	go func() {
		s.processQueuedBlocks(ctx, tickerChan)
		exitChan <- struct{}{}
	}()

	// Send a value over the ticker.
	tickerChan <- 0
	cancel()
	<-exitChan
	assert.LogsContain(t, hook, "New slot, processing queued")
}
