package slasher

import (
	"context"
	"fmt"
	"testing"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func Test_processQueuedAttestations(t *testing.T) {
	type args struct {
		attestationQueue []*slashertypes.IndexedAttestationWrapper
		currentEpoch     types.Epoch
	}
	tests := []struct {
		name                 string
		args                 args
		shouldNotBeSlashable bool
	}{
		{
			name: "Detects surrounding vote (source 1, target 2), (source 0, target 3)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(t, 1, 2, []uint64{0, 1}, nil),
					createAttestationWrapper(t, 0, 3, []uint64{0, 1}, nil),
				},
				currentEpoch: 4,
			},
		},
		{
			name: "Detects surrounding vote (source 50, target 51), (source 0, target 1000)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(t, 50, 51, []uint64{0}, nil),
					createAttestationWrapper(t, 0, 1000, []uint64{0}, nil),
				},
				currentEpoch: 1000,
			},
		},
		{
			name: "Detects surrounded vote (source 0, target 3), (source 1, target 2)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(t, 0, 3, []uint64{0, 1}, nil),
					createAttestationWrapper(t, 1, 2, []uint64{0, 1}, nil),
				},
				currentEpoch: 4,
			},
		},
		{
			name: "Detects double vote, (source 1, target 2), (source 0, target 2)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(t, 1, 2, []uint64{0, 1}, nil),
					createAttestationWrapper(t, 0, 2, []uint64{0, 1}, nil),
				},
				currentEpoch: 4,
			},
		},
		{
			name: "Not slashable, surrounding but non-overlapping attesting indices within same validator chunk index",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(t, 1, 2, []uint64{0}, nil),
					createAttestationWrapper(t, 0, 3, []uint64{1}, nil),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, surrounded but non-overlapping attesting indices within same validator chunk index",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(t, 0, 3, []uint64{0, 1}, nil),
					createAttestationWrapper(t, 1, 2, []uint64{2, 3}, nil),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, surrounding but non-overlapping attesting indices in different validator chunk index",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(t, 0, 3, []uint64{0}, nil),
					createAttestationWrapper(t, 1, 2, []uint64{1000000}, nil),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, surrounded but non-overlapping attesting indices in different validator chunk index",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(t, 0, 3, []uint64{0}, nil),
					createAttestationWrapper(t, 1, 2, []uint64{1000000}, nil),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 1, target 2), (source 2, target 3)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(t, 1, 2, []uint64{0, 1}, nil),
					createAttestationWrapper(t, 2, 3, []uint64{0, 1}, nil),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 0, target 3), (source 2, target 4)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(t, 0, 3, []uint64{0, 1}, nil),
					createAttestationWrapper(t, 2, 4, []uint64{0, 1}, nil),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 0, target 2), (source 0, target 3)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(t, 0, 2, []uint64{0, 1}, nil),
					createAttestationWrapper(t, 0, 3, []uint64{0, 1}, nil),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 0, target 3), (source 0, target 2)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(t, 0, 3, []uint64{0, 1}, nil),
					createAttestationWrapper(t, 0, 2, []uint64{0, 1}, nil),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := logTest.NewGlobal()
			defer hook.Reset()
			beaconDB := dbtest.SetupDB(t)
			ctx, cancel := context.WithCancel(context.Background())

			currentTime := time.Now()
			totalSlots := uint64(tt.args.currentEpoch) * uint64(params.BeaconConfig().SlotsPerEpoch)
			secondsSinceGenesis := time.Duration(totalSlots * params.BeaconConfig().SecondsPerSlot)
			genesisTime := currentTime.Add(-secondsSinceGenesis * time.Second)

			s := &Service{
				serviceCfg: &ServiceConfig{
					Database:              beaconDB,
					StateNotifier:         &mock.MockStateNotifier{},
					AttesterSlashingsFeed: new(event.Feed),
				},
				params:      DefaultParams(),
				attsQueue:   newAttestationsQueue(),
				genesisTime: genesisTime,
			}
			currentSlotChan := make(chan types.Slot)
			exitChan := make(chan struct{})
			go func() {
				s.processQueuedAttestations(ctx, currentSlotChan)
				exitChan <- struct{}{}
			}()
			s.attsQueue.extend(tt.args.attestationQueue)
			slot, err := helpers.StartSlot(tt.args.currentEpoch)
			require.NoError(t, err)
			currentSlotChan <- slot
			cancel()
			<-exitChan
			if tt.shouldNotBeSlashable {
				require.LogsDoNotContain(t, hook, "Attester slashing detected")
			} else {
				require.LogsContain(t, hook, "Attester slashing detected")
			}
		})
	}
}

func Test_processQueuedAttestations_MultipleChunkIndices(t *testing.T) {
	hook := logTest.NewGlobal()
	defer hook.Reset()

	beaconDB := dbtest.SetupDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	slasherParams := DefaultParams()

	// We process submit attestations from chunk index 0 to chunk index 1.
	// What we want to test here is if we can proceed
	// with processing queued attestations once the chunk index changes.
	// For example, epochs 0 - 15 are chunk 0, epochs 16 - 31 are chunk 1, etc.
	startEpoch := types.Epoch(slasherParams.chunkSize)
	endEpoch := types.Epoch(slasherParams.chunkSize + 1)

	currentTime := time.Now()
	totalSlots := uint64(startEpoch) * uint64(params.BeaconConfig().SlotsPerEpoch)
	secondsSinceGenesis := time.Duration(totalSlots * params.BeaconConfig().SecondsPerSlot)
	genesisTime := currentTime.Add(-secondsSinceGenesis * time.Second)

	s := &Service{
		serviceCfg: &ServiceConfig{
			Database:      beaconDB,
			StateNotifier: &mock.MockStateNotifier{},
		},
		params:      slasherParams,
		attsQueue:   newAttestationsQueue(),
		genesisTime: genesisTime,
	}
	currentSlotChan := make(chan types.Slot)
	exitChan := make(chan struct{})
	go func() {
		s.processQueuedAttestations(ctx, currentSlotChan)
		exitChan <- struct{}{}
	}()

	for i := startEpoch; i <= endEpoch; i++ {
		source := types.Epoch(0)
		target := types.Epoch(0)
		if i != 0 {
			source = i - 1
			target = i
		}
		var sr [32]byte
		copy(sr[:], fmt.Sprintf("%d", i))
		att := createAttestationWrapper(t, source, target, []uint64{0}, sr[:])
		s.attsQueue = newAttestationsQueue()
		s.attsQueue.push(att)
		slot, err := helpers.StartSlot(i)
		require.NoError(t, err)
		currentSlotChan <- slot
	}

	cancel()
	<-exitChan
	require.LogsDoNotContain(t, hook, "Slashable offenses found")
	require.LogsDoNotContain(t, hook, "Could not detect")
}

func Test_processQueuedAttestations_OverlappingChunkIndices(t *testing.T) {
	hook := logTest.NewGlobal()
	defer hook.Reset()

	beaconDB := dbtest.SetupDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	slasherParams := DefaultParams()

	startEpoch := types.Epoch(slasherParams.chunkSize)

	currentTime := time.Now()
	totalSlots := uint64(startEpoch) * uint64(params.BeaconConfig().SlotsPerEpoch)
	secondsSinceGenesis := time.Duration(totalSlots * params.BeaconConfig().SecondsPerSlot)
	genesisTime := currentTime.Add(-secondsSinceGenesis * time.Second)

	s := &Service{
		serviceCfg: &ServiceConfig{
			Database:      beaconDB,
			StateNotifier: &mock.MockStateNotifier{},
		},
		params:      slasherParams,
		attsQueue:   newAttestationsQueue(),
		genesisTime: genesisTime,
	}
	currentSlotChan := make(chan types.Slot)
	exitChan := make(chan struct{})
	go func() {
		s.processQueuedAttestations(ctx, currentSlotChan)
		exitChan <- struct{}{}
	}()

	// We create two attestations fully spanning chunk indices 0 and chunk 1
	att1 := createAttestationWrapper(t, types.Epoch(slasherParams.chunkSize-2), types.Epoch(slasherParams.chunkSize), []uint64{0, 1}, nil)
	att2 := createAttestationWrapper(t, types.Epoch(slasherParams.chunkSize-1), types.Epoch(slasherParams.chunkSize+1), []uint64{0, 1}, nil)

	// We attempt to process the batch.
	s.attsQueue = newAttestationsQueue()
	s.attsQueue.push(att1)
	s.attsQueue.push(att2)
	slot, err := helpers.StartSlot(att2.IndexedAttestation.Data.Target.Epoch)
	require.NoError(t, err)
	currentSlotChan <- slot

	cancel()
	<-exitChan
	require.LogsDoNotContain(t, hook, "Slashable offenses found")
	require.LogsDoNotContain(t, hook, "Could not detect")
}

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
