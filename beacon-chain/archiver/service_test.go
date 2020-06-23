package archiver

import (
	"context"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	mock "github.com/prysmaticlabs/prysm/beacon-chain/blockchain/testing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch/precompute"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	statefeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	dbutil "github.com/prysmaticlabs/prysm/beacon-chain/db/testing"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func TestArchiverService_ReceivesBlockProcessedEvent(t *testing.T) {
	hook := logTest.NewGlobal()
	svc, _ := setupService(t)
	st := testutil.NewBeaconState()
	if err := st.SetSlot(1); err != nil {
		t.Fatal(err)
	}
	svc.headFetcher = &mock.ChainService{
		State: st,
	}

	event := &feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			BlockRoot: [32]byte{1, 2, 3},
			Verified:  true,
		},
	}
	triggerStateEvent(t, svc, event)
	testutil.AssertLogsContain(t, hook, fmt.Sprintf("%#x", event.Data.(*statefeed.BlockProcessedData).BlockRoot))
	testutil.AssertLogsContain(t, hook, "Received block processed event")
}

func TestArchiverService_OnlyArchiveAtEpochEnd(t *testing.T) {
	hook := logTest.NewGlobal()
	svc, _ := setupService(t)
	// The head state is NOT an epoch end.
	st := testutil.NewBeaconState()
	if err := st.SetSlot(params.BeaconConfig().SlotsPerEpoch - 2); err != nil {
		t.Fatal(err)
	}
	svc.headFetcher = &mock.ChainService{
		State: st,
	}
	event := &feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			BlockRoot: [32]byte{1, 2, 3},
			Verified:  true,
		},
	}
	triggerStateEvent(t, svc, event)

	// The context should have been canceled.
	if svc.ctx.Err() != context.Canceled {
		t.Error("context was not canceled")
	}
	testutil.AssertLogsContain(t, hook, "Received block processed event")
	// The service should ONLY log any archival logs if we receive a
	// head slot that is an epoch end.
	testutil.AssertLogsDoNotContain(t, hook, "Successfully archived")
}

func TestArchiverService_ArchivesEvenThroughSkipSlot(t *testing.T) {
	hook := logTest.NewGlobal()
	svc, _ := setupService(t)
	validatorCount := uint64(100)
	headState, err := setupState(validatorCount)
	if err != nil {
		t.Fatal(err)
	}
	event := &feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			BlockRoot: [32]byte{1, 2, 3},
			Verified:  true,
		},
	}

	exitRoutine := make(chan bool)
	go func() {
		svc.run(svc.ctx)
		<-exitRoutine
	}()

	// Send out an event every slot, skipping the end slot of the epoch.
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch+1; i++ {
		if err := headState.SetSlot(i); err != nil {
			t.Fatal(err)
		}
		svc.headFetcher = &mock.ChainService{
			State: headState,
		}
		if helpers.IsEpochEnd(i) {
			continue
		}
		// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
		for sent := 0; sent == 0; {
			sent = svc.stateNotifier.StateFeed().Send(event)
		}
	}
	if err := svc.Stop(); err != nil {
		t.Fatal(err)
	}
	exitRoutine <- true

	// The context should have been canceled.
	if svc.ctx.Err() != context.Canceled {
		t.Error("context was not canceled")
	}

	testutil.AssertLogsContain(t, hook, "Received block processed event")
	// Even though there was a skip slot, we should still be able to archive
	// upon the next block event afterwards.
	testutil.AssertLogsContain(t, hook, "Successfully archived")
}

func TestArchiverService_ComputesAndSavesParticipation(t *testing.T) {
	hook := logTest.NewGlobal()
	validatorCount := uint64(100)
	headState, err := setupState(validatorCount)
	if err != nil {
		t.Fatal(err)
	}
	svc, _ := setupService(t)
	svc.headFetcher = &mock.ChainService{
		State: headState,
	}
	event := &feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			BlockRoot: [32]byte{1, 2, 3},
			Verified:  true,
		},
	}
	triggerStateEvent(t, svc, event)

	attestedBalance := uint64(1)

	currentEpoch := helpers.CurrentEpoch(headState)
	wanted := &ethpb.ValidatorParticipation{
		VotedEther:              attestedBalance,
		EligibleEther:           validatorCount * params.BeaconConfig().MaxEffectiveBalance,
		GlobalParticipationRate: float32(attestedBalance) / float32(validatorCount*params.BeaconConfig().MaxEffectiveBalance),
	}

	retrieved, err := svc.beaconDB.ArchivedValidatorParticipation(svc.ctx, currentEpoch)
	if err != nil {
		t.Fatal(err)
	}

	if !proto.Equal(wanted, retrieved) {
		t.Errorf("Wanted participation for epoch %d %v, retrieved %v", currentEpoch-1, wanted, retrieved)
	}
	testutil.AssertLogsContain(t, hook, "Successfully archived")
}

func TestArchiverService_SavesIndicesAndBalances(t *testing.T) {
	hook := logTest.NewGlobal()
	validatorCount := uint64(100)
	headState, err := setupState(validatorCount)
	if err != nil {
		t.Fatal(err)
	}
	svc, _ := setupService(t)
	svc.headFetcher = &mock.ChainService{
		State: headState,
	}
	event := &feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			BlockRoot: [32]byte{1, 2, 3},
			Verified:  true,
		},
	}
	triggerStateEvent(t, svc, event)

	retrieved, err := svc.beaconDB.ArchivedBalances(svc.ctx, helpers.CurrentEpoch(headState))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(headState.Balances(), retrieved) {
		t.Errorf(
			"Wanted balances for epoch %d %v, retrieved %v",
			helpers.CurrentEpoch(headState),
			headState.Balances(),
			retrieved,
		)
	}
	testutil.AssertLogsContain(t, hook, "Successfully archived")
}

func TestArchiverService_SavesCommitteeInfo(t *testing.T) {
	hook := logTest.NewGlobal()
	validatorCount := uint64(100)
	headState, err := setupState(validatorCount)
	if err != nil {
		t.Fatal(err)
	}
	svc, _ := setupService(t)
	svc.headFetcher = &mock.ChainService{
		State: headState,
	}
	event := &feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			BlockRoot: [32]byte{1, 2, 3},
			Verified:  true,
		},
	}
	triggerStateEvent(t, svc, event)

	currentEpoch := helpers.CurrentEpoch(headState)
	proposerSeed, err := helpers.Seed(headState, currentEpoch, params.BeaconConfig().DomainBeaconProposer)
	if err != nil {
		t.Fatal(err)
	}
	attesterSeed, err := helpers.Seed(headState, currentEpoch, params.BeaconConfig().DomainBeaconAttester)
	if err != nil {
		t.Fatal(err)
	}
	wanted := &pb.ArchivedCommitteeInfo{
		ProposerSeed: proposerSeed[:],
		AttesterSeed: attesterSeed[:],
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
	testutil.AssertLogsContain(t, hook, "Successfully archived")
}

func TestArchiverService_SavesActivatedValidatorChanges(t *testing.T) {
	hook := logTest.NewGlobal()
	validatorCount := uint64(100)
	headState, err := setupState(validatorCount)
	if err != nil {
		t.Fatal(err)
	}
	svc, beaconDB := setupService(t)
	svc.headFetcher = &mock.ChainService{
		State: headState,
	}
	prevEpoch := helpers.PrevEpoch(headState)
	delayedActEpoch := helpers.ActivationExitEpoch(prevEpoch)
	val1, err := headState.ValidatorAtIndex(4)
	if err != nil {
		t.Fatal(err)
	}
	val1.ActivationEpoch = delayedActEpoch
	val2, err := headState.ValidatorAtIndex(5)
	if err != nil {
		t.Fatal(err)
	}
	val2.ActivationEpoch = delayedActEpoch
	if err := headState.UpdateValidatorAtIndex(4, val1); err != nil {
		t.Fatal(err)
	}
	if err := headState.UpdateValidatorAtIndex(5, val1); err != nil {
		t.Fatal(err)
	}
	event := &feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			BlockRoot: [32]byte{1, 2, 3},
			Verified:  true,
		},
	}
	triggerStateEvent(t, svc, event)

	retrieved, err := beaconDB.ArchivedActiveValidatorChanges(svc.ctx, prevEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if retrieved == nil {
		t.Fatal("Retrieved indices are nil")
	}
	if len(retrieved.Activated) != 98 {
		t.Error("Did not get wanted active length")
	}
	testutil.AssertLogsContain(t, hook, "Successfully archived")
}

func TestArchiverService_SavesSlashedValidatorChanges(t *testing.T) {
	hook := logTest.NewGlobal()
	validatorCount := uint64(100)
	headState, err := setupState(validatorCount)
	if err != nil {
		t.Fatal(err)
	}
	svc, beaconDB := setupService(t)
	svc.headFetcher = &mock.ChainService{
		State: headState,
	}
	prevEpoch := helpers.PrevEpoch(headState)
	val1, err := headState.ValidatorAtIndex(95)
	if err != nil {
		t.Fatal(err)
	}
	val1.Slashed = true
	val2, err := headState.ValidatorAtIndex(96)
	if err != nil {
		t.Fatal(err)
	}
	val2.Slashed = true
	if err := headState.UpdateValidatorAtIndex(95, val1); err != nil {
		t.Fatal(err)
	}
	if err := headState.UpdateValidatorAtIndex(96, val1); err != nil {
		t.Fatal(err)
	}
	event := &feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			BlockRoot: [32]byte{1, 2, 3},
			Verified:  true,
		},
	}
	triggerStateEvent(t, svc, event)

	retrieved, err := beaconDB.ArchivedActiveValidatorChanges(svc.ctx, prevEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if retrieved == nil {
		t.Fatal("Retrieved indices are nil")
	}
	if !reflect.DeepEqual(retrieved.Slashed, []uint64{95, 96}) {
		t.Errorf("Wanted indices 95, 96 slashed, received %v", retrieved.Slashed)
	}
	testutil.AssertLogsContain(t, hook, "Successfully archived")
}

func TestArchiverService_SavesExitedValidatorChanges(t *testing.T) {
	hook := logTest.NewGlobal()
	validatorCount := uint64(100)
	headState, err := setupState(validatorCount)
	if err != nil {
		t.Fatal(err)
	}
	svc, beaconDB := setupService(t)
	svc.headFetcher = &mock.ChainService{
		State: headState,
	}
	prevEpoch := helpers.PrevEpoch(headState)
	val, err := headState.ValidatorAtIndex(95)
	if err != nil {
		t.Fatal(err)
	}
	val.ExitEpoch = prevEpoch
	val.WithdrawableEpoch = prevEpoch + params.BeaconConfig().MinValidatorWithdrawabilityDelay
	if err := headState.UpdateValidatorAtIndex(95, val); err != nil {
		t.Fatal(err)
	}
	event := &feed.Event{
		Type: statefeed.BlockProcessed,
		Data: &statefeed.BlockProcessedData{
			BlockRoot: [32]byte{1, 2, 3},
			Verified:  true,
		},
	}
	triggerStateEvent(t, svc, event)
	testutil.AssertLogsContain(t, hook, "Successfully archived")
	retrieved, err := beaconDB.ArchivedActiveValidatorChanges(svc.ctx, prevEpoch)
	if err != nil {
		t.Fatal(err)
	}
	if retrieved == nil {
		t.Fatal("Retrieved indices are nil")
	}
	if !reflect.DeepEqual(retrieved.Exited, []uint64{95}) {
		t.Errorf("Wanted indices 95 exited, received %v", retrieved.Exited)
	}
}

func setupState(validatorCount uint64) (*stateTrie.BeaconState, error) {
	validators := make([]*ethpb.Validator, validatorCount)
	balances := make([]uint64, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:         params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch: params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance:  params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	atts := []*pb.PendingAttestation{{Data: &ethpb.AttestationData{Target: &ethpb.Checkpoint{}}}}

	// We initialize a head state that has attestations from participated
	// validators in a simulated fashion.
	st := testutil.NewBeaconState()
	if err := st.SetSlot((2 * params.BeaconConfig().SlotsPerEpoch) - 1); err != nil {
		return nil, err
	}
	if err := st.SetValidators(validators); err != nil {
		return nil, err
	}
	if err := st.SetBalances(balances); err != nil {
		return nil, err
	}
	if err := st.SetCurrentEpochAttestations(atts); err != nil {
		return nil, err
	}
	return st, nil
}

func setupService(t *testing.T) (*Service, db.Database) {
	beaconDB, _ := dbutil.SetupDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	validatorCount := uint64(100)
	totalBalance := validatorCount * params.BeaconConfig().MaxEffectiveBalance
	mockChainService := &mock.ChainService{}
	return &Service{
		beaconDB:      beaconDB,
		ctx:           ctx,
		cancel:        cancel,
		stateNotifier: mockChainService.StateNotifier(),
		participationFetcher: &mock.ChainService{
			Balance: &precompute.Balance{ActivePrevEpoch: totalBalance, PrevEpochTargetAttested: 1}},
	}, beaconDB
}

func triggerStateEvent(t *testing.T, svc *Service, event *feed.Event) {
	exitRoutine := make(chan bool)
	go func() {
		svc.run(svc.ctx)
		<-exitRoutine
	}()

	// Send in a loop to ensure it is delivered (busy wait for the service to subscribe to the state feed).
	for sent := 0; sent == 0; {
		sent = svc.stateNotifier.StateFeed().Send(event)
	}
	if err := svc.Stop(); err != nil {
		t.Fatal(err)
	}
	exitRoutine <- true

	// The context should have been canceled.
	if svc.ctx.Err() != context.Canceled {
		t.Error("context was not canceled")
	}
}
