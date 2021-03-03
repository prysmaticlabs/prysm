package slasher

import (
	"context"
	"testing"

	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	dbtest "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/event"
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
		wantedLogs           []string
	}{
		{
			name: "Detects surrounding vote (source 1, target 2), (source 0, target 3)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(1, 2, []uint64{0, 1} /* indices */, nil /* signingRoot */),
					createAttestationWrapper(0, 3, []uint64{0, 1} /* indices */, nil /* signingRoot */),
				},
				currentEpoch: 4,
			},
			wantedLogs: []string{"Attester surrounding vote"},
		},
		{
			name: "Detects surrounding vote (source 50, target 51), (source 0, target 1000)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(50, 51, []uint64{0} /* indices */, nil /* signingRoot */),
					createAttestationWrapper(0, 1000, []uint64{0} /* indices */, nil /* signingRoot */),
				},
				currentEpoch: 1000,
			},
			wantedLogs: []string{"Attester surrounding vote"},
		},
		{
			name: "Detects surrounded vote (source 0, target 3), (source 1, target 2)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(0, 3, []uint64{0, 1} /* indices */, nil /* signingRoot */),
					createAttestationWrapper(1, 2, []uint64{0, 1} /* indices */, nil /* signingRoot */),
				},
				currentEpoch: 4,
			},
			wantedLogs: []string{"Attester surrounded vote"},
		},
		{
			name: "Not slashable, surrounding but non-overlapping attesting indices within same validator chunk index",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(1, 2, []uint64{0} /* indices */, nil /* signingRoot */),
					createAttestationWrapper(0, 3, []uint64{1} /* indices */, nil /* signingRoot */),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, surrounded but non-overlapping attesting indices within same validator chunk index",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(0, 3, []uint64{0, 1} /* indices */, nil /* signingRoot */),
					createAttestationWrapper(1, 2, []uint64{2, 3} /* indices */, nil /* signingRoot */),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, surrounding but non-overlapping attesting indices in different validator chunk index",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(0, 3, []uint64{0} /* indices */, nil /* signingRoot */),
					createAttestationWrapper(1, 2, []uint64{1000000} /* indices */, nil /* signingRoot */),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, surrounded but non-overlapping attesting indices in different validator chunk index",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(0, 3, []uint64{0} /* indices */, nil /* signingRoot */),
					createAttestationWrapper(1, 2, []uint64{1000000} /* indices */, nil /* signingRoot */),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 1, target 2), (source 2, target 3)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(1, 2, []uint64{0, 1} /* indices */, nil /* signingRoot */),
					createAttestationWrapper(2, 3, []uint64{0, 1} /* indices */, nil /* signingRoot */),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, same signing root, (source 1, target 2), (source 1, target 2)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(1, 2, []uint64{0, 1}, []byte{1}),
					createAttestationWrapper(1, 2, []uint64{0, 1}, []byte{1}),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 0, target 3), (source 2, target 4)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(0, 3, []uint64{0, 1} /* indices */, nil /* signingRoot */),
					createAttestationWrapper(2, 4, []uint64{0, 1} /* indices */, nil /* signingRoot */),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 1, target 2), (source 0, target 2)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(1, 2, []uint64{0, 1} /* indices */, nil /* signingRoot */),
					createAttestationWrapper(0, 2, []uint64{0, 1} /* indices */, nil /* signingRoot */),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 0, target 2), (source 0, target 3)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(0, 2, []uint64{0, 1} /* indices */, nil /* signingRoot */),
					createAttestationWrapper(0, 3, []uint64{0, 1} /* indices */, nil /* signingRoot */),
				},
				currentEpoch: 4,
			},
			shouldNotBeSlashable: true,
		},
		{
			name: "Not slashable, (source 0, target 3), (source 0, target 2)",
			args: args{
				attestationQueue: []*slashertypes.IndexedAttestationWrapper{
					createAttestationWrapper(0, 3, []uint64{0, 1} /* indices */, nil /* signingRoot */),
					createAttestationWrapper(0, 2, []uint64{0, 1} /* indices */, nil /* signingRoot */),
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
				attestationQueue: make([]*slashertypes.IndexedAttestationWrapper, 0),
			}
			currentEpochChan := make(chan types.Epoch)
			exitChan := make(chan struct{})
			go func() {
				s.processQueuedAttestations(ctx, currentEpochChan)
				exitChan <- struct{}{}
			}()
			s.attestationQueue = tt.args.attestationQueue
			currentEpochChan <- tt.args.currentEpoch
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
	att1 := createAttestationWrapper(1, 2, firstIndices, nil)
	att2 := createAttestationWrapper(1, 2, secondIndices, nil)
	s.indexedAttsChan <- att1.IndexedAttestation
	s.indexedAttsChan <- att2.IndexedAttestation
	cancel()
	<-exitChan
	wanted := []*slashertypes.IndexedAttestationWrapper{
		att1,
		att2,
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
	validAtt := createAttestationWrapper(1, 2, firstIndices, nil)
	s.indexedAttsChan <- validAtt.IndexedAttestation
	// Send an invalid, bad attestation which will not
	// pass integrity checks at it has invalid attestation data.
	s.indexedAttsChan <- &ethpb.IndexedAttestation{
		AttestingIndices: secondIndices,
	}
	cancel()
	<-exitChan
	// Expect only a single, valid attestation was added to the queue.
	require.Equal(t, 1, len(s.attestationQueue))
	wanted := []*slashertypes.IndexedAttestationWrapper{
		validAtt,
	}
	require.DeepEqual(t, wanted, s.attestationQueue)
}

func TestSlasher_receiveBlocks_OK(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Service{
		serviceCfg: &ServiceConfig{
			BeaconBlocksFeed: new(event.Feed),
		},
		beaconBlocksChan: make(chan *ethpb.SignedBeaconBlockHeader),
	}
	exitChan := make(chan struct{})
	go func() {
		s.receiveBlocks(ctx)
		exitChan <- struct{}{}
	}()

	block1 := createProposalWrapper(0, 1, nil).SignedBeaconBlockHeader
	block2 := createProposalWrapper(0, 2, nil).SignedBeaconBlockHeader
	s.beaconBlocksChan <- block1
	s.beaconBlocksChan <- block2
	cancel()
	<-exitChan
	wanted := []*slashertypes.SignedBlockHeaderWrapper{
		createProposalWrapper(0, block1.Header.ProposerIndex, nil),
		createProposalWrapper(0, block2.Header.ProposerIndex, nil),
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
		beaconBlocksQueue: []*slashertypes.SignedBlockHeaderWrapper{
			createProposalWrapper(0, 1, nil),
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	tickerChan := make(chan types.Epoch)
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
