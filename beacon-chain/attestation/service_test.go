package attestation

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/internal"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

var _ = TargetHandler(&Service{})

func TestUpdateLatestAttestation_UpdatesLatest(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()

	var validators []*ethpb.Validator
	for i := 0; i < 64; i++ {
		validators = append(validators, &ethpb.Validator{
			PublicKey:       []byte{byte(i)},
			ActivationEpoch: 0,
			ExitEpoch:       10,
		})
	}

	beaconState := &pb.BeaconState{
		Slot:             1,
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	block := &ethpb.BeaconBlock{
		Slot: 1,
	}
	if err := beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.UpdateChainHead(ctx, block, beaconState); err != nil {
		t.Fatal(err)
	}
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})

	attestation := &ethpb.Attestation{
		AggregationBits: bitfield.Bitlist{0x03},
		Data: &ethpb.AttestationData{
			Crosslink: &ethpb.Crosslink{
				Shard: 1,
			},
			Target: &ethpb.Checkpoint{},
			Source: &ethpb.Checkpoint{},
		},
	}

	if err := service.UpdateLatestAttestation(ctx, attestation); err != nil {
		t.Fatalf("could not update latest attestation: %v", err)
	}
	pubkey := bytesutil.ToBytes48(beaconState.Validators[10].PublicKey)
	if service.store.m[pubkey].Data.Crosslink.Shard !=
		attestation.Data.Crosslink.Shard {
		t.Errorf("Incorrect shard stored, wanted: %d, got: %d",
			attestation.Data.Crosslink.Shard, service.store.m[pubkey].Data.Crosslink.Shard)
	}

	beaconState = &pb.BeaconState{
		Slot:             36,
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}
	if err := beaconDB.UpdateChainHead(ctx, block, beaconState); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	attestation.Data.Crosslink.Shard = 36
	if err := service.UpdateLatestAttestation(ctx, attestation); err != nil {
		t.Fatalf("could not update latest attestation: %v", err)
	}
	if service.store.m[pubkey].Data.Crosslink.Shard !=
		attestation.Data.Crosslink.Shard {
		t.Errorf("Incorrect shard stored, wanted: %d, got: %d",
			attestation.Data.Crosslink.Shard, service.store.m[pubkey].Data.Crosslink.Shard)
	}
}

func TestAttestationPool_UpdatesAttestationPool(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()

	var validators []*ethpb.Validator
	for i := 0; i < 64; i++ {
		validators = append(validators, &ethpb.Validator{
			PublicKey:       []byte{byte(i)},
			ActivationEpoch: 0,
			ExitEpoch:       10,
		})
	}
	beaconState := &pb.BeaconState{
		Slot:       1,
		Validators: validators,
	}
	block := &ethpb.BeaconBlock{
		Slot: 1,
	}
	if err := beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.UpdateChainHead(ctx, block, beaconState); err != nil {
		t.Fatal(err)
	}

	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})
	attestation := &ethpb.Attestation{
		AggregationBits: bitfield.Bitlist{0x80, 0x01},
		Data: &ethpb.AttestationData{
			Crosslink: &ethpb.Crosslink{
				Shard: 1,
			},
		},
	}

	if err := service.handleAttestation(context.Background(), attestation); err != nil {
		t.Error(err)
	}
}

func TestLatestAttestationTarget_CantGetAttestation(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()

	if err := beaconDB.SaveState(ctx, &pb.BeaconState{
		Validators: []*ethpb.Validator{{}},
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})
	headState, err := beaconDB.HeadState(ctx)
	if err != nil {
		t.Fatal(err)
	}

	index := uint64(100)
	want := fmt.Sprintf("invalid validator index %d", index)
	if _, err := service.LatestAttestationTarget(headState, index); !strings.Contains(err.Error(), want) {
		t.Errorf("Wanted error to contain %s, received %v", want, err)
	}
}

func TestLatestAttestationTarget_ReturnsLatestAttestedBlock(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()

	pubKey := []byte{'A'}
	if err := beaconDB.SaveState(ctx, &pb.BeaconState{
		Validators: []*ethpb.Validator{{PublicKey: pubKey}},
	}); err != nil {
		t.Fatalf("could not save state: %v", err)
	}

	block := &ethpb.BeaconBlock{Slot: 999}
	if err := beaconDB.SaveBlock(block); err != nil {
		t.Fatalf("could not save block: %v", err)
	}
	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		log.Fatalf("could not hash block: %v", err)
	}
	if err := beaconDB.SaveAttestationTarget(ctx, &pb.AttestationTarget{
		Slot:            block.Slot,
		BeaconBlockRoot: blockRoot[:],
		ParentRoot:      []byte{},
	}); err != nil {
		log.Fatalf("could not save att target: %v", err)
	}

	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})

	attestation := &ethpb.Attestation{
		Data: &ethpb.AttestationData{
			BeaconBlockRoot: blockRoot[:],
		}}
	pubKey48 := bytesutil.ToBytes48(pubKey)
	service.store.m[pubKey48] = attestation

	headState, err := beaconDB.HeadState(ctx)
	if err != nil {
		t.Fatal(err)
	}

	latestAttestedTarget, err := service.LatestAttestationTarget(headState, 0)
	if err != nil {
		t.Fatalf("Could not get latest attestation: %v", err)
	}

	if !bytes.Equal(blockRoot[:], latestAttestedTarget.BeaconBlockRoot) {
		t.Errorf("Wanted: %v, got: %v", blockRoot[:], latestAttestedTarget.BeaconBlockRoot)
	}
}

func TestUpdateLatestAttestation_InvalidIndex(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	hook := logTest.NewGlobal()
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()

	var validators []*ethpb.Validator
	for i := 0; i < 64; i++ {
		validators = append(validators, &ethpb.Validator{
			PublicKey:       []byte{byte(i)},
			ActivationEpoch: 0,
			ExitEpoch:       10,
		})
	}

	beaconState := &pb.BeaconState{
		Slot:             1,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		Validators:       validators,
	}
	block := &ethpb.BeaconBlock{
		Slot: 1,
	}
	if err := beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.UpdateChainHead(ctx, block, beaconState); err != nil {
		t.Fatal(err)
	}
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})
	attestation := &ethpb.Attestation{
		AggregationBits: bitfield.Bitlist{0xC0, 0x01},
		Data: &ethpb.AttestationData{
			Crosslink: &ethpb.Crosslink{
				Shard: 1,
			},
			Target: &ethpb.Checkpoint{},
			Source: &ethpb.Checkpoint{},
		},
	}

	wanted := "bitfield points to an invalid index in the committee"

	if err := service.UpdateLatestAttestation(ctx, attestation); err != nil {
		t.Error(err)
	}

	testutil.AssertLogsContain(t, hook, wanted)
}

func TestBatchUpdate_FromSync(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()

	var validators []*ethpb.Validator
	var latestRandaoMixes [][]byte
	var latestActiveIndexRoots [][]byte
	for i := 0; i < 64; i++ {
		validators = append(validators, &ethpb.Validator{
			PublicKey:       []byte{byte(i)},
			ActivationEpoch: 0,
			ExitEpoch:       10,
		})
		latestRandaoMixes = append(latestRandaoMixes, []byte{'A'})
		latestActiveIndexRoots = append(latestActiveIndexRoots, []byte{'B'})
	}

	beaconState := &pb.BeaconState{
		Slot:             1,
		Validators:       validators,
		RandaoMixes:      latestRandaoMixes,
		ActiveIndexRoots: latestActiveIndexRoots,
	}
	block := &ethpb.BeaconBlock{
		Slot: 1,
	}
	if err := beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.UpdateChainHead(ctx, block, beaconState); err != nil {
		t.Fatal(err)
	}
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})
	service.poolLimit = 9
	for i := 0; i < 10; i++ {
		attestation := &ethpb.Attestation{
			AggregationBits: bitfield.Bitlist{0x80},
			Data: &ethpb.AttestationData{
				Target: &ethpb.Checkpoint{Epoch: 2},
				Source: &ethpb.Checkpoint{},
				Crosslink: &ethpb.Crosslink{
					Shard: 1,
				},
			},
		}
		if err := service.handleAttestation(ctx, attestation); err != nil {
			t.Fatalf("could not update latest attestation: %v", err)
		}
	}
	if len(service.pooledAttestations) != 0 {
		t.Errorf("pooled attestations were not cleared out, still %d attestations in pool", len(service.pooledAttestations))
	}
}

func TestUpdateLatestAttestation_BatchUpdate(t *testing.T) {
	beaconDB := internal.SetupDB(t)
	defer internal.TeardownDB(t, beaconDB)
	ctx := context.Background()

	var validators []*ethpb.Validator
	for i := 0; i < 64; i++ {
		validators = append(validators, &ethpb.Validator{
			PublicKey:       []byte{byte(i)},
			ActivationEpoch: 0,
			ExitEpoch:       10,
		})
	}

	beaconState := &pb.BeaconState{
		Slot:             1,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		Validators:       validators,
	}
	block := &ethpb.BeaconBlock{
		Slot: 1,
	}
	if err := beaconDB.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	if err := beaconDB.UpdateChainHead(ctx, block, beaconState); err != nil {
		t.Fatal(err)
	}
	service := NewAttestationService(context.Background(), &Config{BeaconDB: beaconDB})
	attestations := make([]*ethpb.Attestation, 0)
	for i := 0; i < 10; i++ {
		attestations = append(attestations, &ethpb.Attestation{
			AggregationBits: bitfield.Bitlist{0x80, 0x01},
			Data: &ethpb.AttestationData{
				Crosslink: &ethpb.Crosslink{
					Shard: 1,
				},
				Target: &ethpb.Checkpoint{},
				Source: &ethpb.Checkpoint{},
			},
		})
	}

	if err := service.BatchUpdateLatestAttestation(ctx, attestations); err != nil {
		t.Fatalf("could not update latest attestation: %v", err)
	}
}
