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
}

func (mb *mockBroadcaster) Broadcast(_ context.Context, _ proto.Message) {
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
	go func() {
		s.removeOperations()
		s.saveOperations()
		<-exitRoutine
	}()
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

	exit := &pb.VoluntaryExit{Epoch: 100}
	if err := service.HandleValidatorExits(context.Background(), exit); err != nil {
		t.Error(err)
	}

	want := fmt.Sprintf("Exit request saved in DB")
	testutil.AssertLogsContain(t, hook, want)
}

func TestIncomingAttestation_OK(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	broadcaster := &mockBroadcaster{}
	service := NewOpsPoolService(context.Background(), &Config{
		BeaconDB: beaconDB,
		P2P:      broadcaster,
	})

	attestation := &pb.Attestation{
		AggregationBitfield: []byte{'A'},
		Data: &pb.AttestationData{
			Slot: 100,
		}}
	if err := service.HandleAttestations(context.Background(), attestation); err != nil {
		t.Error(err)
	}
}

func TestRetrieveAttestations_OK(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	service := NewOpsPoolService(context.Background(), &Config{BeaconDB: beaconDB})

	// Save 140 attestations for test. During 1st retrieval we should get slot:1 - slot:61 attestations.
	// The 1st retrieval is set at slot 64.
	origAttestations := make([]*pb.Attestation, 140)
	for i := 0; i < len(origAttestations); i++ {
		origAttestations[i] = &pb.Attestation{
			Data: &pb.AttestationData{
				Slot:                    params.BeaconConfig().GenesisSlot + uint64(i),
				CrosslinkDataRootHash32: params.BeaconConfig().ZeroHash[:],
			},
		}
		if err := service.beaconDB.SaveAttestation(context.Background(), origAttestations[i]); err != nil {
			t.Fatalf("Failed to save attestation: %v", err)
		}
	}
	if err := beaconDB.SaveState(context.Background(), &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot + 64,
		LatestCrosslinks: []*pb.Crosslink{{
			Epoch:                   params.BeaconConfig().GenesisEpoch,
			CrosslinkDataRootHash32: params.BeaconConfig().ZeroHash[:]}}}); err != nil {
		t.Fatal(err)
	}
	// Test we can retrieve attestations from slot1 - slot61.
	attestations, err := service.PendingAttestations(context.Background())
	if err != nil {
		t.Fatalf("Could not retrieve attestations: %v", err)
	}

	if !reflect.DeepEqual(attestations, origAttestations[1:128]) {
		t.Error("Retrieved attestations did not match")
	}
}

func TestRetrieveAttestations_PruneInvalidAtts(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	service := NewOpsPoolService(context.Background(), &Config{BeaconDB: beaconDB})

	// Save 140 attestations for slots 0 to 139.
	origAttestations := make([]*pb.Attestation, 140)
	for i := 0; i < len(origAttestations); i++ {
		origAttestations[i] = &pb.Attestation{
			Data: &pb.AttestationData{
				Slot:                    params.BeaconConfig().GenesisSlot + uint64(i),
				CrosslinkDataRootHash32: params.BeaconConfig().ZeroHash[:],
			},
		}
		if err := service.beaconDB.SaveAttestation(context.Background(), origAttestations[i]); err != nil {
			t.Fatalf("Failed to save attestation: %v", err)
		}
	}

	// At slot 200 only attestations up to from slot 137 to 139 are valid attestations.
	if err := beaconDB.SaveState(context.Background(), &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot + 200,
		LatestCrosslinks: []*pb.Crosslink{{
			Epoch:                   params.BeaconConfig().GenesisEpoch + 2,
			CrosslinkDataRootHash32: params.BeaconConfig().ZeroHash[:]}}}); err != nil {
		t.Fatal(err)
	}
	attestations, err := service.PendingAttestations(context.Background())
	if err != nil {
		t.Fatalf("Could not retrieve attestations: %v", err)
	}
	if !reflect.DeepEqual(attestations, origAttestations[137:]) {
		t.Error("Incorrect pruned attestations")
	}

	// Verify the invalid attestations are deleted.
	hash, err := hashutil.HashProto(origAttestations[136])
	if err != nil {
		t.Fatal(err)
	}
	if service.beaconDB.HasAttestation(hash) {
		t.Error("Invalid attestation is not deleted")
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
				Slot:                    params.BeaconConfig().GenesisSlot + uint64(i),
				CrosslinkDataRootHash32: params.BeaconConfig().ZeroHash[:],
			},
		}
		if err := s.beaconDB.SaveAttestation(context.Background(), attestations[i]); err != nil {
			t.Fatalf("Failed to save attestation: %v", err)
		}
	}
	if err := db.SaveState(context.Background(), &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot + 15,
		LatestCrosslinks: []*pb.Crosslink{{
			Epoch:                   params.BeaconConfig().GenesisEpoch,
			CrosslinkDataRootHash32: params.BeaconConfig().ZeroHash[:]}}}); err != nil {
		t.Fatal(err)
	}

	retrievedAtts, err := s.PendingAttestations(context.Background())
	if err != nil {
		t.Fatalf("Could not retrieve attestations: %v", err)
	}
	if !reflect.DeepEqual(attestations, retrievedAtts) {
		t.Error("Retrieved attestations did not match prev generated attestations")
	}

	if err := s.removePendingAttestations(attestations); err != nil {
		t.Fatalf("Could not remove pending attestations: %v", err)
	}

	retrievedAtts, _ = s.PendingAttestations(context.Background())
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
				Slot:                    params.BeaconConfig().GenesisSlot + uint64(i),
				CrosslinkDataRootHash32: params.BeaconConfig().ZeroHash[:],
			},
		}
		if err := s.beaconDB.SaveAttestation(context.Background(), attestations[i]); err != nil {
			t.Fatalf("Failed to save attestation: %v", err)
		}
	}

	if err := db.SaveState(context.Background(), &pb.BeaconState{
		Slot: params.BeaconConfig().GenesisSlot + 15,
		LatestCrosslinks: []*pb.Crosslink{{
			Epoch:                   params.BeaconConfig().GenesisEpoch,
			CrosslinkDataRootHash32: params.BeaconConfig().ZeroHash[:]}}}); err != nil {
		t.Fatal(err)
	}

	atts, _ := s.PendingAttestations(context.Background())
	if len(atts) != len(attestations) {
		t.Errorf("Attestation pool should be %d but got a length of %d",
			len(attestations), len(atts))
	}

	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: attestations,
		},
	}

	s.incomingProcessedBlock <- block
	if err := s.handleProcessedBlock(context.Background(), block); err != nil {
		t.Error(err)
	}

	atts, _ = s.PendingAttestations(context.Background())
	if len(atts) != 0 {
		t.Errorf("Attestation pool should be empty but got a length of %d", len(atts))
	}
}
