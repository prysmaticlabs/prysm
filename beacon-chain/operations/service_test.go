package operations

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

// Ensure operations service implements intefaces.
var _ = OperationFeeds(&Service{})

type mockBroadcaster struct {
	broadcastCalled bool
}

func (mb *mockBroadcaster) Broadcast(_ context.Context, _ proto.Message) {
	mb.broadcastCalled = true
}

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestStop_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	opsService := NewOpsPoolService(context.Background(), &Config{})

	if err := opsService.Stop(); err != nil {
		t.Fatalf("Unable to stop operation service: %v", err)
	}

	msg := hook.LastEntry().Message
	want := "Stopping service"
	if msg != want {
		t.Errorf("incorrect log, expected %s, got %s", want, msg)
	}

	// The context should have been canceled.
	if opsService.ctx.Err() != context.Canceled {
		t.Error("context was not canceled")
	}
	hook.Reset()
}

func TestServiceStatus_Error(t *testing.T) {
	service := NewOpsPoolService(context.Background(), &Config{})
	if service.Status() != nil {
		t.Errorf("service status should be nil to begin with, got: %v", service.error)
	}
	err := errors.New("error error error")
	service.error = err

	if service.Status() != err {
		t.Error("service status did not return wanted err")
	}
}

func TestRoutineContextClosing_Ok(t *testing.T) {
	hook := logTest.NewGlobal()
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	s := NewOpsPoolService(context.Background(), &Config{BeaconDB: db})

	exitRoutine := make(chan bool)
	go func(tt *testing.T) {
		s.removeOperations()
		s.saveOperations()
		<-exitRoutine
	}(t)
	s.cancel()
	exitRoutine <- true
	testutil.AssertLogsContain(t, hook, "operations service context closed, exiting remove goroutine")
	testutil.AssertLogsContain(t, hook, "operations service context closed, exiting save goroutine")
}

func TestIncomingExits_Ok(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	service := NewOpsPoolService(context.Background(), &Config{BeaconDB: beaconDB})

	exitRoutine := make(chan bool)
	go func() {
		service.saveOperations()
		<-exitRoutine
	}()
	exit := &pb.VoluntaryExit{Epoch: 100}
	hash, err := hashutil.HashProto(exit)
	if err != nil {
		t.Fatalf("Could not hash exit proto: %v", err)
	}

	service.incomingValidatorExits <- exit
	service.cancel()
	exitRoutine <- true

	want := fmt.Sprintf("Exit request %#x saved in DB", hash)
	testutil.AssertLogsContain(t, hook, want)
}

func TestIncomingAttestation_OK(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	broadcaster := &mockBroadcaster{}
	service := NewOpsPoolService(context.Background(), &Config{
		BeaconDB: beaconDB,
		P2P:      broadcaster,
	})

	exitRoutine := make(chan bool)
	go func() {
		service.saveOperations()
		<-exitRoutine
	}()
	attestation := &pb.Attestation{
		AggregationBitfield: []byte{'A'},
		Data: &pb.AttestationData{
			Slot: 100,
		}}
	hash, err := hashutil.HashProto(attestation)
	if err != nil {
		t.Fatalf("Could not hash exit proto: %v", err)
	}

	service.incomingAtt <- attestation
	service.cancel()
	exitRoutine <- true

	want := fmt.Sprintf("Attestation %#x saved in DB", hash)
	testutil.AssertLogsContain(t, hook, want)

	if !broadcaster.broadcastCalled {
		t.Error("Attestation was not broadcasted")
	}
}

func TestRetrieveAttestations_OK(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	service := NewOpsPoolService(context.Background(), &Config{BeaconDB: beaconDB})

	// Save 140 attestations for test. During 1st retrieval we should get slot:0 - slot:128 attestations,
	// 2nd retrieval we should get slot:128 - slot:140 attestations.
	// Max attestation config value is set to 128.
	origAttestations := make([]*pb.Attestation, 140)
	for i := 0; i < len(origAttestations); i++ {
		origAttestations[i] = &pb.Attestation{
			Data: &pb.AttestationData{
				Slot:  uint64(i),
				Shard: uint64(i),
			},
		}
		if err := service.beaconDB.SaveAttestation(context.Background(), origAttestations[i]); err != nil {
			t.Fatalf("Failed to save attestation: %v", err)
		}
	}
	// Test we can retrieve attestations from slot0 - slot127 (Max attestation amount).
	attestations, err := service.PendingAttestations()
	if err != nil {
		t.Fatalf("Could not retrieve attestations: %v", err)
	}
	if !reflect.DeepEqual(attestations, origAttestations[0:params.BeaconConfig().MaxAttestations]) {
		t.Errorf("Retrieved attestations did not match prev generated attestations for the first %d",
			params.BeaconConfig().MaxAttestations)
	}
}

func TestRemoveProcessedAttestations_Ok(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	s := NewOpsPoolService(context.Background(), &Config{BeaconDB: db})

	attestations := make([]*pb.Attestation, 10)
	for i := 0; i < len(attestations); i++ {
		attestations[i] = &pb.Attestation{
			Data: &pb.AttestationData{
				Slot:  uint64(i),
				Shard: uint64(i),
			},
		}
		if err := s.beaconDB.SaveAttestation(context.Background(), attestations[i]); err != nil {
			t.Fatalf("Failed to save attestation: %v", err)
		}
	}

	retrievedAtts, err := s.PendingAttestations()
	if err != nil {
		t.Fatalf("Could not retrieve attestations: %v", err)
	}
	if !reflect.DeepEqual(attestations, retrievedAtts) {
		t.Error("Retrieved attestations did not match prev generated attestations")
	}

	if err := s.removePendingAttestations(attestations); err != nil {
		t.Fatalf("Could not remove pending attestations: %v", err)
	}

	retrievedAtts, _ = s.PendingAttestations()
	if len(retrievedAtts) != 0 {
		t.Errorf("Attestation pool should be empty but got a length of %d", len(retrievedAtts))
	}
}

func TestCleanUpAttestations_OlderThanOneEpoch(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	s := NewOpsPoolService(context.Background(), &Config{BeaconDB: db})

	// Construct attestations for slot 0..99.
	slot := uint64(99)
	attestations := make([]*pb.Attestation, slot+1)
	for i := 0; i < len(attestations); i++ {
		attestations[i] = &pb.Attestation{
			Data: &pb.AttestationData{
				Slot:  params.BeaconConfig().GenesisSlot + uint64(i),
				Shard: uint64(i),
			},
		}
		if err := s.beaconDB.SaveAttestation(context.Background(), attestations[i]); err != nil {
			t.Fatalf("Failed to save attestation: %v", err)
		}
	}

	// Assume current slot is 99. All the attestations before (99 - 64) should get removed.
	if err := s.removeEpochOldAttestations(params.BeaconConfig().GenesisSlot + slot); err != nil {
		t.Fatalf("Could not remove old attestations: %v", err)
	}
	attestations, err := s.beaconDB.Attestations()
	if err != nil {
		t.Fatalf("Could not retrieve attestations: %v", err)
	}
	for _, a := range attestations {
		if a.Data.Slot < slot-params.BeaconConfig().SlotsPerEpoch {
			t.Errorf("Attestation slot %d can't be lower than %d",
				a.Data.Slot, slot-params.BeaconConfig().SlotsPerEpoch)
		}
	}
}

func TestReceiveBlkRemoveOps_Ok(t *testing.T) {
	db := internal.SetupDB(t)
	defer internal.TeardownDB(t, db)
	s := NewOpsPoolService(context.Background(), &Config{BeaconDB: db})

	attestations := make([]*pb.Attestation, 10)
	for i := 0; i < len(attestations); i++ {
		attestations[i] = &pb.Attestation{
			Data: &pb.AttestationData{
				Slot:  uint64(i),
				Shard: uint64(i),
			},
		}
		if err := s.beaconDB.SaveAttestation(context.Background(), attestations[i]); err != nil {
			t.Fatalf("Failed to save attestation: %v", err)
		}
	}

	atts, _ := s.PendingAttestations()
	if len(atts) != len(attestations) {
		t.Errorf("Attestation pool should be %d but got a length of %d",
			len(attestations), len(atts))
	}

	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}

	exitRoutine := make(chan bool)
	go func() {
		s.removeOperations()
		exitRoutine <- true
	}()

	s.incomingProcessedBlock <- block
	s.cancel()
	<-exitRoutine

	atts, _ = s.PendingAttestations()
	if len(atts) != 0 {
		t.Errorf("Attestation pool should be empty but got a length of %d", len(atts))
	}
}
