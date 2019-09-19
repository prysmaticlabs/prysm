package archiver

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
}

func TestArchiverService_ReceivesNewChainHeadEvent(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx, cancel := context.WithCancel(context.Background())
	svc := &Service{
		ctx:             ctx,
		cancel:          cancel,
		newHeadRootChan: make(chan [32]byte, 0),
		newHeadNotifier: &mock.ChainService{},
	}
	exitRoutine := make(chan bool)
	go func() {
		svc.run()
		<-exitRoutine
	}()

	svc.newHeadRootChan <- [32]byte{1, 2, 3}
	if err := svc.Stop(); err != nil {
		t.Fatal(err)
	}
	exitRoutine <- true

	// The context should have been canceled.
	if svc.ctx.Err() != context.Canceled {
		t.Error("context was not canceled")
	}
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("%#x", [32]byte{1, 2, 3}))
	testutil.AssertLogsContain(t, hook, "New chain head event")
}

func TestArchiverService_ComputesAndSavesParticipation(t *testing.T) {
	hook := logTest.NewGlobal()
	db := dbutil.SetupDB(t)
	defer dbutil.TeardownDB(t, db)
	attestedBalance := uint64(1)
	validatorCount := uint64(100)

	validators := make([]*ethpb.Validator, validatorCount)
	balances := make([]uint64, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	atts := []*pb.PendingAttestation{{Data: &ethpb.AttestationData{Crosslink: &ethpb.Crosslink{Shard: 0}, Target: &ethpb.Checkpoint{}}}}
	var crosslinks []*ethpb.Crosslink
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		crosslinks = append(crosslinks, &ethpb.Crosslink{
			StartEpoch: 0,
			DataRoot:   []byte{'A'},
		})
	}

	// We initialize a head state that has attestations from participated
	// validators in a simulated fashion.
	headState := &pb.BeaconState{
		Slot:                       (2 * params.BeaconConfig().SlotsPerEpoch) - 1,
		Validators:                 validators,
		Balances:                   balances,
		BlockRoots:                 make([][]byte, 128),
		Slashings:                  []uint64{0, 1e9, 1e9},
		RandaoMixes:                make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:           make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CompactCommitteesRoots:     make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CurrentCrosslinks:          crosslinks,
		CurrentEpochAttestations:   atts,
		FinalizedCheckpoint:        &ethpb.Checkpoint{},
		JustificationBits:          bitfield.Bitvector4{0x00},
		CurrentJustifiedCheckpoint: &ethpb.Checkpoint{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	headFetcher := &mock.ChainService{
		State: headState,
	}
	svc := &Service{
		beaconDB:        db,
		ctx:             ctx,
		cancel:          cancel,
		newHeadRootChan: make(chan [32]byte, 0),
		newHeadNotifier: &mock.ChainService{},
		headFetcher:     headFetcher,
	}
	exitRoutine := make(chan bool)
	go func() {
		svc.run()
		<-exitRoutine
	}()

	// Upon receiving a new head state,
	svc.newHeadRootChan <- [32]byte{}
	if err := svc.Stop(); err != nil {
		t.Fatal(err)
	}
	exitRoutine <- true

	// The context should have been canceled.
	if svc.ctx.Err() != context.Canceled {
		t.Error("context was not canceled")
	}

	wanted := &ethpb.ValidatorParticipation{
		Epoch:                   helpers.SlotToEpoch(headState.Slot),
		VotedEther:              attestedBalance,
		EligibleEther:           validatorCount * params.BeaconConfig().MaxEffectiveBalance,
		GlobalParticipationRate: float32(attestedBalance) / float32(validatorCount*params.BeaconConfig().MaxEffectiveBalance),
	}

	retrieved, err := svc.beaconDB.ArchivedValidatorParticipation(ctx, wanted.Epoch)
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(wanted, retrieved) {
		t.Errorf("Wanted participation for epoch %d %v, retrieved %v", wanted.Epoch, wanted, retrieved)
	}
	testutil.AssertLogsContain(t, hook, "archived validator participation")
}

func TestArchiverService_OnlyArchiveAtEpochEnd(t *testing.T) {
	hook := logTest.NewGlobal()
	ctx, cancel := context.WithCancel(context.Background())
	svc := &Service{
		ctx:             ctx,
		cancel:          cancel,
		newHeadRootChan: make(chan [32]byte, 0),
		newHeadNotifier: &mock.ChainService{},
	}
	exitRoutine := make(chan bool)
	go func() {
		svc.run()
		<-exitRoutine
	}()

	// The head state is NOT an epoch end.
	svc.headFetcher = &mock.ChainService{
		State: &pb.BeaconState{Slot: params.BeaconConfig().SlotsPerEpoch - 3},
	}
	svc.newHeadRootChan <- [32]byte{}
	if err := svc.Stop(); err != nil {
		t.Fatal(err)
	}
	exitRoutine <- true

	// The context should have been canceled.
	if svc.ctx.Err() != context.Canceled {
		t.Error("context was not canceled")
	}
	testutil.AssertLogsContain(t, hook, "New chain head event")
	// The service should ONLY log any archival logs if we receive a
	// head slot that is an epoch end.
	testutil.AssertLogsDoNotContain(t, hook, "Successfully archived validator participation during epoch")
}
