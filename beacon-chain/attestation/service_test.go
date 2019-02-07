package attestation

import (
	"context"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestUpdateLatestAttestation_Ok(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	if err := beaconDB.SaveState(&pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{{Pubkey: []byte{'A'}}},
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})

	attestation := &pb.Attestation{
		AggregationBitfield: []byte{0x80},
		Data: &pb.AttestationData{
			Slot: 5,
		},
	}

	if err := service.updateLatestAttestation(attestation); err != nil {
		t.Fatalf("could not update latest attestation: %v", err)
	}
	pubkey := bytesutil.ToBytes48([]byte{'A'})
	if service.LatestAttestation[pubkey].Data.Slot !=
		attestation.Data.Slot {
		t.Errorf("Incorrect slot stored, wanted: %d, got: %d",
			attestation.Data.Slot, service.LatestAttestation[pubkey].Data.Slot)
	}

	attestation.Data.Slot = 100
	if err := service.updateLatestAttestation(attestation); err != nil {
		t.Fatalf("could not update latest attestation: %v", err)
	}
	if service.LatestAttestation[pubkey].Data.Slot !=
		attestation.Data.Slot {
		t.Errorf("Incorrect slot stored, wanted: %d, got: %d",
			attestation.Data.Slot, service.LatestAttestation[pubkey].Data.Slot)
	}
}

func TestAttestationPool_Ok(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	if err := beaconDB.SaveState(&pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{{Pubkey: []byte{'A'}}},
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})
	attestation := &pb.Attestation{
		AggregationBitfield: []byte{0x80},
		Data:                &pb.AttestationData{},
	}

	exitRoutine := make(chan bool)
	go func() {
		service.attestationPool()
		<-exitRoutine
	}()
	service.incomingChan <- attestation

	service.cancel()
	exitRoutine <- true

	testutil.AssertLogsContain(t, hook, "Updated attestation pool for attestation")
}
