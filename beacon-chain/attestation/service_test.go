package attestation

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

func TestUpdateLatestAttestation_UpdatesLatest(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()

	var validators []*pb.Validator
	for i := 0; i < 64; i++ {
		validators = append(validators, &pb.Validator{
			Pubkey:          []byte{byte(i)},
			ActivationEpoch: 0,
			ExitEpoch:       10,
		})
	}

	if err := beaconDB.SaveState(&pb.BeaconState{
		ValidatorRegistry: validators,
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})

	attestation := &pb.Attestation{
		AggregationBitfield: []byte{0x80},
		Data: &pb.AttestationData{
			Slot:  1,
			Shard: 1,
		},
	}

	if err := service.updateLatestAttestation(ctx, attestation); err != nil {
		t.Fatalf("could not update latest attestation: %v", err)
	}
	pubkey := bytesutil.ToBytes48([]byte{byte(35)})
	if service.Store[pubkey].Data.Slot !=
		attestation.Data.Slot {
		t.Errorf("Incorrect slot stored, wanted: %d, got: %d",
			attestation.Data.Slot, service.Store[pubkey].Data.Slot)
	}

	attestation.Data.Slot = 100
	attestation.Data.Shard = 36
	if err := service.updateLatestAttestation(ctx, attestation); err != nil {
		t.Fatalf("could not update latest attestation: %v", err)
	}
	if service.Store[pubkey].Data.Slot !=
		attestation.Data.Slot {
		t.Errorf("Incorrect slot stored, wanted: %d, got: %d",
			attestation.Data.Slot, service.Store[pubkey].Data.Slot)
	}
}

func TestAttestationPool_UpdatesAttestationPool(t *testing.T) {
	hook := logTest.NewGlobal()
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)

	var validators []*pb.Validator
	for i := 0; i < 64; i++ {
		validators = append(validators, &pb.Validator{
			Pubkey:          []byte{byte(i)},
			ActivationEpoch: 0,
			ExitEpoch:       10,
		})
	}
	if err := beaconDB.SaveState(&pb.BeaconState{
		ValidatorRegistry: validators,
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})
	attestation := &pb.Attestation{
		AggregationBitfield: []byte{0x80},
		Data: &pb.AttestationData{
			Slot:  1,
			Shard: 1,
		},
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

func TestLatestAttestation_ReturnsLatestAttestation(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()

	pubKey := []byte{'A'}
	if err := beaconDB.SaveState(&pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{{Pubkey: pubKey}},
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})
	pubKey48 := bytesutil.ToBytes48(pubKey)
	attestation := &pb.Attestation{AggregationBitfield: []byte{'B'}}
	service.Store[pubKey48] = attestation

	latestAttestation, err := service.LatestAttestation(ctx, 0)
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
	ctx := context.Background()

	if err := beaconDB.SaveState(&pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{},
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})

	index := uint64(0)
	want := fmt.Sprintf("invalid validator index %d", index)
	if _, err := service.LatestAttestation(ctx, index); !strings.Contains(err.Error(), want) {
		t.Errorf("Wanted error to contain %s, received %v", want, err)
	}
}

func TestLatestAttestation_NoAttestation(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()

	if err := beaconDB.SaveState(&pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{{}},
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})

	index := 0
	want := fmt.Sprintf("validator index %d does not have an attestation", index)
	if _, err := service.LatestAttestation(ctx, uint64(index)); !strings.Contains(err.Error(), want) {
		t.Errorf("Wanted error to contain %s, received %v", want, err)
	}
}

func TestLatestAttestationTarget_CantGetAttestation(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()

	if err := beaconDB.SaveState(&pb.BeaconState{
		ValidatorRegistry: []*pb.Validator{{}},
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})

	index := uint64(100)
	want := fmt.Sprintf("could not get attestation: invalid validator index %d", index)
	if _, err := service.LatestAttestationTarget(ctx, index); !strings.Contains(err.Error(), want) {
		t.Errorf("Wanted error to contain %s, received %v", want, err)
	}
}

func TestLatestAttestationTarget_ReturnsLatestAttestedBlock(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()

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
	blockRoot, err := hashutil.HashBeaconBlock(block)
	if err != nil {
		log.Fatalf("could not hash block: %v", err)
	}

	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})

	attestation := &pb.Attestation{
		Data: &pb.AttestationData{
			BeaconBlockRootHash32: blockRoot[:],
		}}
	pubKey48 := bytesutil.ToBytes48(pubKey)
	service.Store[pubKey48] = attestation

	latestAttestedBlock, err := service.LatestAttestationTarget(ctx, 0)
	if err != nil {
		t.Fatalf("Could not get latest attestation: %v", err)
	}
	if !reflect.DeepEqual(block, latestAttestedBlock) {
		t.Errorf("Wanted: %v, got: %v", block, latestAttestedBlock)
	}
}
