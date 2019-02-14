package attestation

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/ssz"

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
	if service.store[pubkey].Data.Slot !=
		attestation.Data.Slot {
		t.Errorf("Incorrect slot stored, wanted: %d, got: %d",
			attestation.Data.Slot, service.store[pubkey].Data.Slot)
	}

	attestation.Data.Slot = 100
	if err := service.updateLatestAttestation(attestation); err != nil {
		t.Fatalf("could not update latest attestation: %v", err)
	}
	if service.store[pubkey].Data.Slot !=
		attestation.Data.Slot {
		t.Errorf("Incorrect slot stored, wanted: %d, got: %d",
			attestation.Data.Slot, service.store[pubkey].Data.Slot)
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

func TestLatestAttestation_Ok(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	pubKey := []byte{'A'}
	if err := beaconDB.SaveState(&pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{{Pubkey: pubKey}},
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})
	pubKey48 := bytesutil.ToBytes48(pubKey)
	attestation := &pb.Attestation{AggregationBitfield: []byte{'B'}}
	service.store[pubKey48] = attestation

	latestAttestation, err := service.LatestAttestation(0)
	if err != nil {
		t.Fatalf("Could not get latest attestation: %v", err)
	}
	if !reflect.DeepEqual(attestation, latestAttestation) {
		t.Errorf("Wanted: %v, got: %v", attestation, latestAttestation)
	}
}

func TestLatestAttestation_InvalidIndex(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	if err := beaconDB.SaveState(&pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{},
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})

	index := 0
	want := fmt.Sprintf("invalid validator index %d", index)
	if _, err := service.LatestAttestation(index); !strings.Contains(err.Error(), want) {
		t.Errorf("Wanted error to contain %s, received %v", want, err)
	}
}

func TestLatestAttestation_NoAttestation(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	if err := beaconDB.SaveState(&pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{{}},
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})

	index := 0
	want := fmt.Sprintf("validator index %d does not have an attestation", index)
	if _, err := service.LatestAttestation(index); !strings.Contains(err.Error(), want) {
		t.Errorf("Wanted error to contain %s, received %v", want, err)
	}
}

func TestLatestAttestationTarget_CantGetAttestation(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	if err := beaconDB.SaveState(&pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{{}},
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})

	index := 100
	want := fmt.Sprintf("could not get attestation: invalid validator index %d", index)
	if _, err := service.LatestAttestationTarget(index); !strings.Contains(err.Error(), want) {
		t.Errorf("Wanted error to contain %s, received %v", want, err)
	}
}

func TestLatestAttestationTarget_Ok(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

	pubKey := []byte{'A'}
	if err := beaconDB.SaveState(&pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{{Pubkey: pubKey}},
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	block := &pb.BeaconBlock{Slot: 999}
	if err := beaconDB.SaveBlock(block); err != nil {
		t.Fatalf("could not save block: %v", err)
	}
	blockRoot, err := ssz.TreeHash(block)
	if err != nil {
		log.Fatalf("could not hash block: %v", err)
	}

	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})

	attestation := &pb.Attestation{
		Data: &pb.AttestationData{
			BeaconBlockRootHash32: blockRoot[:],
		}}
	pubKey48 := bytesutil.ToBytes48(pubKey)
	service.store[pubKey48] = attestation

	latestAttestedBlock, err := service.LatestAttestationTarget(0)
	if err != nil {
		t.Fatalf("Could not get latest attestation: %v", err)
	}
	if !reflect.DeepEqual(block, latestAttestedBlock) {
		t.Errorf("Wanted: %v, got: %v", block, latestAttestedBlock)
	}
}
