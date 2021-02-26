package slasher

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func Test_processQueuedAttestations(t *testing.T) {
	type args struct {
		attestationQueue []*slashertypes.CompactAttestation
		currentEpoch     types.Epoch
	}
	tests := []struct {
		name                 string
		args                 args
		shouldNotBeSlashable bool
		wantedLogs           []string
	}{
		{
			name: "Detects surrounding vote (source 1, target 2), (source 0, target 3)",
			args: args{
				attestationQueue: []*slashertypes.CompactAttestation{
					{
						AttestingIndices: []uint64{0, 1},
						Source:           1,
						Target:           2,
					},
					{
						AttestingIndices: []uint64{0, 1},
						Source:           0,
						Target:           3,
					},
				},
				currentEpoch: 4,
			},
			wantedLogs: []string{"Attester surrounding vote"},
		},
		{
			name: "Detects surrounding vote (source 50, target 51), (source 0, target 1000)",
			args: args{
				attestationQueue: []*slashertypes.CompactAttestation{
					{
						AttestingIndices: []uint64{0},
						Source:           50,
						Target:           51,
					},
					{
						AttestingIndices: []uint64{0},
						Source:           0,
						Target:           1000,
					},
				},
				currentEpoch: 1000,
			},
			wantedLogs: []string{"Attester surrounding vote"},
		},
		{
			name: "Detects surrounded vote (source 0, target 3), (source 1, target 2)",
			args: args{
				attestationQueue: []*slashertypes.CompactAttestation{
					{
						AttestingIndices: []uint64{0, 1},
						Source:           0,
						Target:           3,
					},
					{
						AttestingIndices: []uint64{0, 1},
						Source:           1,
						Target:           2,
					},
				},
				currentEpoch: 4,
			},
			wantedLogs: []string{"Attester surrounded vote"},
		},
		{
			name: "Not slashable, surrounding but non-overlapping attesting indices within same validator chunk index",
			args: args{
				attestationQueue: []*slashertypes.CompactAttestation{
					{
						AttestingIndices: []uint64{0},
						Source:           1,
						Target:           2,
					},
					{
						AttestingIndices: []uint64{1},
						Source:           0,
						Target:           3,
					},
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, surrounded but non-overlapping attesting indices within same validator chunk index",
			args: args{
				attestationQueue: []*slashertypes.CompactAttestation{
					{
						AttestingIndices: []uint64{0, 1},
						Source:           0,
						Target:           3,
					},
					{
						AttestingIndices: []uint64{2, 3},
						Source:           1,
						Target:           2,
					},
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, surrounding but non-overlapping attesting indices in different validator chunk index",
			args: args{
				attestationQueue: []*slashertypes.CompactAttestation{
					{
						AttestingIndices: []uint64{0},
						Source:           0,
						Target:           3,
					},
					{
						AttestingIndices: []uint64{1000000},
						Source:           1,
						Target:           2,
					},
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, surrounded but non-overlapping attesting indices in different validator chunk index",
			args: args{
				attestationQueue: []*slashertypes.CompactAttestation{
					{
						AttestingIndices: []uint64{0},
						Source:           0,
						Target:           3,
					},
					{
						AttestingIndices: []uint64{1000000},
						Source:           1,
						Target:           2,
					},
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 1, target 2), (source 2, target 3)",
			args: args{
				attestationQueue: []*slashertypes.CompactAttestation{
					{
						AttestingIndices: []uint64{0, 1},
						Source:           1,
						Target:           2,
					},
					{
						AttestingIndices: []uint64{0, 1},
						Source:           2,
						Target:           3,
					},
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, same signing root, (source 1, target 2), (source 1, target 2)",
			args: args{
				attestationQueue: []*slashertypes.CompactAttestation{
					{
						AttestingIndices: []uint64{0, 1},
						Source:           1,
						Target:           2,
						SigningRoot:      [32]byte{1},
					},
					{
						AttestingIndices: []uint64{0, 1},
						Source:           1,
						Target:           2,
						SigningRoot:      [32]byte{1},
					},
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 0, target 3), (source 2, target 4)",
			args: args{
				attestationQueue: []*slashertypes.CompactAttestation{
					{
						AttestingIndices: []uint64{0, 1},
						Source:           0,
						Target:           3,
					},
					{
						AttestingIndices: []uint64{0, 1},
						Source:           2,
						Target:           4,
					},
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 1, target 2), (source 0, target 2)",
			args: args{
				attestationQueue: []*slashertypes.CompactAttestation{
					{
						AttestingIndices: []uint64{0, 1},
						Source:           1,
						Target:           2,
					},
					{
						AttestingIndices: []uint64{0, 1},
						Source:           0,
						Target:           2,
					},
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 0, target 2), (source 0, target 3)",
			args: args{
				attestationQueue: []*slashertypes.CompactAttestation{
					{
						AttestingIndices: []uint64{0, 1},
						Source:           0,
						Target:           2,
					},
					{
						AttestingIndices: []uint64{0, 1},
						Source:           0,
						Target:           3,
					},
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 0, target 3), (source 0, target 2)",
			args: args{
				attestationQueue: []*slashertypes.CompactAttestation{
					{
						AttestingIndices: []uint64{0, 1},
						Source:           0,
						Target:           3,
					},
					{
						AttestingIndices: []uint64{0, 1},
						Source:           0,
						Target:           2,
					},
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
			s := &Service{
				serviceCfg: &ServiceConfig{
					Database: beaconDB,
				},
				params:           DefaultParams(),
				attestationQueue: make([]*slashertypes.CompactAttestation, 0),
			}
			currentSlotChan := make(chan types.Slot)
			exitChan := make(chan struct{})
			go func() {
				s.processQueuedAttestations(ctx, currentSlotChan)
				exitChan <- struct{}{}
			}()
			s.attestationQueue = tt.args.attestationQueue
			slot, err := helpers.StartSlot(tt.args.currentEpoch)
			require.NoError(t, err)
			currentSlotChan <- slot
			cancel()
			<-exitChan
			if tt.shouldNotBeSlashable {
				require.LogsDoNotContain(t, hook, "Slashable offenses found")
			} else {
				for _, wanted := range tt.wantedLogs {
					require.LogsContain(t, hook, wanted)
				}
			}
		})
	}
}

func TestSlasher_receiveAttestations_OK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		serviceCfg: &ServiceConfig{
			IndexedAttsFeed: new(event.Feed),
		},
		indexedAttsChan: make(chan *ethpb.IndexedAttestation),
	}
	exitChan := make(chan struct{})
	go func() {
		s.receiveAttestations(ctx)
		exitChan <- struct{}{}
	}()
	firstIndices := []uint64{1, 2, 3}
	secondIndices := []uint64{4, 5, 6}
	att1 := testutil.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
		AttestingIndices: firstIndices,
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{
				Epoch: 1,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 2,
			},
		},
	})
	att2 := testutil.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
		AttestingIndices: secondIndices,
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{
				Epoch: 1,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 2,
			},
		},
	})
	s.indexedAttsChan <- att1
	s.indexedAttsChan <- att2
	cancel()
	<-exitChan
	sr1, err := att1.Data.HashTreeRoot()
	require.NoError(t, err)
	sr2, err := att2.Data.HashTreeRoot()
	require.NoError(t, err)
	wanted := []*slashertypes.CompactAttestation{
		{
			AttestingIndices: att1.AttestingIndices,
			Source:           att1.Data.Source.Epoch,
			Target:           att1.Data.Target.Epoch,
			SigningRoot:      sr1,
		},
		{
			AttestingIndices: att2.AttestingIndices,
			Source:           att2.Data.Source.Epoch,
			Target:           att2.Data.Target.Epoch,
			SigningRoot:      sr2,
		},
	}
	require.DeepEqual(t, wanted, s.attestationQueue)
}

func TestSlasher_receiveAttestations_OnlyValidAttestations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		serviceCfg: &ServiceConfig{
			IndexedAttsFeed: new(event.Feed),
		},
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
	validAtt := testutil.HydrateIndexedAttestation(&ethpb.IndexedAttestation{
		AttestingIndices: firstIndices,
		Data: &ethpb.AttestationData{
			Source: &ethpb.Checkpoint{
				Epoch: 1,
			},
			Target: &ethpb.Checkpoint{
				Epoch: 2,
			},
		},
	})
	sr, err := validAtt.Data.HashTreeRoot()
	require.NoError(t, err)
	s.indexedAttsChan <- validAtt
	// Send an invalid, bad attestation which will not
	// pass integrity checks at it has invalid attestation data.
	s.indexedAttsChan <- &ethpb.IndexedAttestation{
		AttestingIndices: secondIndices,
	}
	cancel()
	<-exitChan
	// Expect only a single, valid attestation was added to the queue.
	require.Equal(t, 1, len(s.attestationQueue))
	wanted := []*slashertypes.CompactAttestation{
		{
			AttestingIndices: validAtt.AttestingIndices,
			Source:           validAtt.Data.Source.Epoch,
			Target:           validAtt.Data.Target.Epoch,
			SigningRoot:      sr,
		},
	}
	require.DeepEqual(t, wanted, s.attestationQueue)
}

func TestSlasher_receiveBlocks_OK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		serviceCfg: &ServiceConfig{
			BeaconBlocksFeed: new(event.Feed),
		},
		beaconBlocksChan: make(chan *ethpb.BeaconBlockHeader),
	}
	exitChan := make(chan struct{})
	go func() {
		s.receiveBlocks(ctx)
		exitChan <- struct{}{}
	}()
	block1 := testutil.HydrateBeaconHeader(&ethpb.BeaconBlockHeader{
		ProposerIndex: 1,
	})
	block2 := testutil.HydrateBeaconHeader(&ethpb.BeaconBlockHeader{
		ProposerIndex: 2,
	})
	s.beaconBlocksChan <- block1
	s.beaconBlocksChan <- block2
	cancel()
	<-exitChan
	sr1, err := block1.HashTreeRoot()
	require.NoError(t, err)
	sr2, err := block2.HashTreeRoot()
	require.NoError(t, err)
	wanted := []*slashertypes.CompactBeaconBlock{
		{
			ProposerIndex: types.ValidatorIndex(block1.ProposerIndex),
			SigningRoot:   sr1,
		},
		{
			ProposerIndex: types.ValidatorIndex(block2.ProposerIndex),
			SigningRoot:   sr2,
		},
	}
	require.DeepEqual(t, wanted, s.beaconBlocksQueue)
}

func TestService_processQueuedBlocks(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := dbtest.SetupDB(t)
	s := &Service{
		params: DefaultParams(),
		serviceCfg: &ServiceConfig{
			Database: beaconDB,
		},
		beaconBlocksQueue: []*slashertypes.CompactBeaconBlock{
			{
				ProposerIndex: 1,
			},
		},
	}
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
	assert.LogsContain(t, hook, "Epoch reached, processing queued")
}
