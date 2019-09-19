package archiver

import (
	"context"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-bitfield"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
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
	svc, db := setupService(t)
	defer dbutil.TeardownDB(t, db)
	svc.headFetcher = &mock.ChainService{
		State: &pb.BeaconState{Slot: 1},
	}
	headRoot := [32]byte{1, 2, 3}
	triggerNewHeadEvent(t, svc, headRoot)
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("%#x", headRoot))
	testutil.AssertLogsContain(t, hook, "New chain head event")
}

func TestArchiverService_OnlyArchiveAtEpochEnd(t *testing.T) {
	hook := logTest.NewGlobal()
	svc, db := setupService(t)
	defer dbutil.TeardownDB(t, db)
	// The head state is NOT an epoch end.
	svc.headFetcher = &mock.ChainService{
		State: &pb.BeaconState{Slot: params.BeaconConfig().SlotsPerEpoch - 3},
	}
	triggerNewHeadEvent(t, svc, [32]byte{})

	// The context should have been canceled.
	if svc.ctx.Err() != context.Canceled {
		t.Error("context was not canceled")
	}
	testutil.AssertLogsContain(t, hook, "New chain head event")
	// The service should ONLY log any archival logs if we receive a
	// head slot that is an epoch end.
	testutil.AssertLogsDoNotContain(t, hook, "Successfully archived validator participation during epoch")
}

func TestArchiverService_ComputesAndSavesParticipation(t *testing.T) {
	hook := logTest.NewGlobal()
	validatorCount := uint64(100)
	headState := setupState(t, validatorCount)
	svc, db := setupService(t)
	defer dbutil.TeardownDB(t, db)
	svc.headFetcher = &mock.ChainService{
		State: headState,
	}
	triggerNewHeadEvent(t, svc, [32]byte{})

	attestedBalance := uint64(1)
	wanted := &ethpb.ValidatorParticipation{
		Epoch:                   helpers.SlotToEpoch(headState.Slot),
		VotedEther:              attestedBalance,
		EligibleEther:           validatorCount * params.BeaconConfig().MaxEffectiveBalance,
		GlobalParticipationRate: float32(attestedBalance) / float32(validatorCount*params.BeaconConfig().MaxEffectiveBalance),
	}

	retrieved, err := svc.beaconDB.ArchivedValidatorParticipation(svc.ctx, wanted.Epoch)
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(wanted, retrieved) {
		t.Errorf("Wanted participation for epoch %d %v, retrieved %v", wanted.Epoch, wanted, retrieved)
	}
	testutil.AssertLogsContain(t, hook, "archived validator participation")
}

func TestArchiverService_SavesIndicesAndBalances(t *testing.T) {
	hook := logTest.NewGlobal()
	validatorCount := uint64(100)
	headState := setupState(t, validatorCount)
	svc, db := setupService(t)
	defer dbutil.TeardownDB(t, db)
	svc.headFetcher = &mock.ChainService{
		State: headState,
	}
	triggerNewHeadEvent(t, svc, [32]byte{})

	retrieved, err := svc.beaconDB.ArchivedBalances(svc.ctx, helpers.CurrentEpoch(headState))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(headState.Balances, retrieved) {
		t.Errorf(
			"Wanted balances for epoch %d %v, retrieved %v",
			helpers.CurrentEpoch(headState),
			headState.Balances,
			retrieved,
		)
	}
	testutil.AssertLogsContain(t, hook, "archived validator balances and active indices")
}

func TestArchiverService_SavesCommitteeInfo(t *testing.T) {
	hook := logTest.NewGlobal()
	validatorCount := uint64(100)
	headState := setupState(t, validatorCount)
	svc, db := setupService(t)
	defer dbutil.TeardownDB(t, db)
	svc.headFetcher = &mock.ChainService{
		State: headState,
	}
	triggerNewHeadEvent(t, svc, [32]byte{})

	currentEpoch := helpers.CurrentEpoch(headState)
	startShard, err := helpers.StartShard(headState, currentEpoch)
	if err != nil {
		t.Fatal(err)
	}
	committeeCount, err := helpers.CommitteeCount(headState, currentEpoch)
	if err != nil {
		t.Fatal(err)
	}
	seed, err := helpers.Seed(headState, currentEpoch)
	if err != nil {
		t.Fatal(err)
	}
	wanted := &ethpb.ArchivedCommitteeInfo{
		Seed:           seed[:],
		StartShard:     startShard,
		CommitteeCount: committeeCount,
	}

	retrieved, err := svc.beaconDB.ArchivedCommitteeInfo(svc.ctx, helpers.CurrentEpoch(headState))
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(wanted, retrieved) {
		t.Errorf(
			"Wanted committee info for epoch %d %v, retrieved %v",
			helpers.CurrentEpoch(headState),
			wanted,
			retrieved,
		)
	}
	testutil.AssertLogsContain(t, hook, "archived validator balances and active indices")
}

func setupState(t *testing.T, validatorCount uint64) *pb.BeaconState {
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
	return &pb.BeaconState{
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
}

func setupService(t *testing.T) (*Service, db.Database) {
	db := dbutil.SetupDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		beaconDB:        db,
		ctx:             ctx,
		cancel:          cancel,
		newHeadRootChan: make(chan [32]byte, 0),
		newHeadNotifier: &mock.ChainService{},
	}, db
}

func triggerNewHeadEvent(t *testing.T, svc *Service, headRoot [32]byte) {
	exitRoutine := make(chan bool)
	go func() {
		svc.run()
		<-exitRoutine
	}()

	svc.newHeadRootChan <- headRoot
	if err := svc.Stop(); err != nil {
		t.Fatal(err)
	}
	exitRoutine <- true

	// The context should have been canceled.
	if svc.ctx.Err() != context.Canceled {
		t.Error("context was not canceled")
	}
}
