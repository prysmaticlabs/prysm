package attestations

import (
	"context"
	"flag"
	"strconv"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"

	"github.com/gogo/protobuf/proto"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/detection"
	"github.com/urfave/cli"
)

type spanMapTestStruct struct {
	validatorIdx        uint64
	sourceEpoch         uint64
	targetEpoch         uint64
	slashingTargetEpoch uint64
	resultSpanMap       *slashpb.EpochSpanMap
}

var spanTestsMax []spanMapTestStruct
var spanTestsMin []spanMapTestStruct

func init() {
	// Test data following example of a max span by https://github.com/protolambda
	// from here: https://github.com/protolambda/eth2-surround/blob/master/README.md#min-max-surround
	spanTestsMax = []spanMapTestStruct{
		{
			validatorIdx:        0,
			sourceEpoch:         3,
			targetEpoch:         6,
			slashingTargetEpoch: 0,
			resultSpanMap: &slashpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
					4: {MinEpochSpan: 0, MaxEpochSpan: 2},
					5: {MinEpochSpan: 0, MaxEpochSpan: 1},
				},
			},
		},
		{
			validatorIdx:        0,
			sourceEpoch:         8,
			targetEpoch:         18,
			slashingTargetEpoch: 0,
			resultSpanMap: &slashpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
					4:  {MinEpochSpan: 0, MaxEpochSpan: 2},
					5:  {MinEpochSpan: 0, MaxEpochSpan: 1},
					9:  {MinEpochSpan: 0, MaxEpochSpan: 9},
					10: {MinEpochSpan: 0, MaxEpochSpan: 8},
					11: {MinEpochSpan: 0, MaxEpochSpan: 7},
					12: {MinEpochSpan: 0, MaxEpochSpan: 6},
					13: {MinEpochSpan: 0, MaxEpochSpan: 5},
					14: {MinEpochSpan: 0, MaxEpochSpan: 4},
					15: {MinEpochSpan: 0, MaxEpochSpan: 3},
					16: {MinEpochSpan: 0, MaxEpochSpan: 2},
					17: {MinEpochSpan: 0, MaxEpochSpan: 1},
				},
			},
		},
		{
			validatorIdx:        0,
			sourceEpoch:         4,
			targetEpoch:         12,
			slashingTargetEpoch: 0,
			resultSpanMap: &slashpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
					4:  {MinEpochSpan: 0, MaxEpochSpan: 2},
					5:  {MinEpochSpan: 0, MaxEpochSpan: 7},
					6:  {MinEpochSpan: 0, MaxEpochSpan: 6},
					7:  {MinEpochSpan: 0, MaxEpochSpan: 5},
					8:  {MinEpochSpan: 0, MaxEpochSpan: 4},
					9:  {MinEpochSpan: 0, MaxEpochSpan: 9},
					10: {MinEpochSpan: 0, MaxEpochSpan: 8},
					11: {MinEpochSpan: 0, MaxEpochSpan: 7},
					12: {MinEpochSpan: 0, MaxEpochSpan: 6},
					13: {MinEpochSpan: 0, MaxEpochSpan: 5},
					14: {MinEpochSpan: 0, MaxEpochSpan: 4},
					15: {MinEpochSpan: 0, MaxEpochSpan: 3},
					16: {MinEpochSpan: 0, MaxEpochSpan: 2},
					17: {MinEpochSpan: 0, MaxEpochSpan: 1},
				},
			},
		},
		{
			validatorIdx:        0,
			sourceEpoch:         10,
			targetEpoch:         15,
			slashingTargetEpoch: 18,
			resultSpanMap: &slashpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
					4:  {MinEpochSpan: 0, MaxEpochSpan: 2},
					5:  {MinEpochSpan: 0, MaxEpochSpan: 7},
					6:  {MinEpochSpan: 0, MaxEpochSpan: 6},
					7:  {MinEpochSpan: 0, MaxEpochSpan: 5},
					8:  {MinEpochSpan: 0, MaxEpochSpan: 4},
					9:  {MinEpochSpan: 0, MaxEpochSpan: 9},
					10: {MinEpochSpan: 0, MaxEpochSpan: 8},
					11: {MinEpochSpan: 0, MaxEpochSpan: 7},
					12: {MinEpochSpan: 0, MaxEpochSpan: 6},
					13: {MinEpochSpan: 0, MaxEpochSpan: 5},
					14: {MinEpochSpan: 0, MaxEpochSpan: 4},
					15: {MinEpochSpan: 0, MaxEpochSpan: 3},
					16: {MinEpochSpan: 0, MaxEpochSpan: 2},
					17: {MinEpochSpan: 0, MaxEpochSpan: 1},
				},
			},
		},
	}

	spanTestsMin = []spanMapTestStruct{
		{
			validatorIdx:        0,
			sourceEpoch:         4,
			targetEpoch:         6,
			slashingTargetEpoch: 0,
			resultSpanMap: &slashpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
					1: {MinEpochSpan: 5, MaxEpochSpan: 0},
					2: {MinEpochSpan: 4, MaxEpochSpan: 0},
					3: {MinEpochSpan: 3, MaxEpochSpan: 0},
				},
			},
		},
		{
			validatorIdx:        0,
			sourceEpoch:         13,
			targetEpoch:         18,
			slashingTargetEpoch: 0,
			resultSpanMap: &slashpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
					1:  {MinEpochSpan: 5, MaxEpochSpan: 0},
					2:  {MinEpochSpan: 4, MaxEpochSpan: 0},
					3:  {MinEpochSpan: 3, MaxEpochSpan: 0},
					4:  {MinEpochSpan: 14, MaxEpochSpan: 0},
					5:  {MinEpochSpan: 13, MaxEpochSpan: 0},
					6:  {MinEpochSpan: 12, MaxEpochSpan: 0},
					7:  {MinEpochSpan: 11, MaxEpochSpan: 0},
					8:  {MinEpochSpan: 10, MaxEpochSpan: 0},
					9:  {MinEpochSpan: 9, MaxEpochSpan: 0},
					10: {MinEpochSpan: 8, MaxEpochSpan: 0},
					11: {MinEpochSpan: 7, MaxEpochSpan: 0},
					12: {MinEpochSpan: 6, MaxEpochSpan: 0},
				},
			},
		},
		{
			validatorIdx:        0,
			sourceEpoch:         11,
			targetEpoch:         15,
			slashingTargetEpoch: 0,
			resultSpanMap: &slashpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
					1:  {MinEpochSpan: 5, MaxEpochSpan: 0},
					2:  {MinEpochSpan: 4, MaxEpochSpan: 0},
					3:  {MinEpochSpan: 3, MaxEpochSpan: 0},
					4:  {MinEpochSpan: 11, MaxEpochSpan: 0},
					5:  {MinEpochSpan: 10, MaxEpochSpan: 0},
					6:  {MinEpochSpan: 9, MaxEpochSpan: 0},
					7:  {MinEpochSpan: 8, MaxEpochSpan: 0},
					8:  {MinEpochSpan: 7, MaxEpochSpan: 0},
					9:  {MinEpochSpan: 6, MaxEpochSpan: 0},
					10: {MinEpochSpan: 5, MaxEpochSpan: 0},
					11: {MinEpochSpan: 7, MaxEpochSpan: 0},
					12: {MinEpochSpan: 6, MaxEpochSpan: 0},
				},
			},
		},
		{
			validatorIdx:        0,
			sourceEpoch:         10,
			targetEpoch:         20,
			slashingTargetEpoch: 15,
			resultSpanMap: &slashpb.EpochSpanMap{
				EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
					1:  {MinEpochSpan: 5, MaxEpochSpan: 0},
					2:  {MinEpochSpan: 4, MaxEpochSpan: 0},
					3:  {MinEpochSpan: 3, MaxEpochSpan: 0},
					4:  {MinEpochSpan: 11, MaxEpochSpan: 0},
					5:  {MinEpochSpan: 10, MaxEpochSpan: 0},
					6:  {MinEpochSpan: 9, MaxEpochSpan: 0},
					7:  {MinEpochSpan: 8, MaxEpochSpan: 0},
					8:  {MinEpochSpan: 7, MaxEpochSpan: 0},
					9:  {MinEpochSpan: 6, MaxEpochSpan: 0},
					10: {MinEpochSpan: 5, MaxEpochSpan: 0},
					11: {MinEpochSpan: 7, MaxEpochSpan: 0},
					12: {MinEpochSpan: 6, MaxEpochSpan: 0},
				},
			},
		},
	}
}

func TestServer_UpdateMaxEpochSpan(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(t, c)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	detector := AttDetector{&detection.SlashingDetector{
		SlasherDB: dbs,
	}}
	for _, tt := range spanTestsMax {
		spanMap, err := detector.slashingDetector.SlasherDB.ValidatorSpansMap(tt.validatorIdx)
		if err != nil {
			t.Fatal(err)
		}
		st, spanMap, err := detector.DetectSurroundedAttestations(ctx, tt.sourceEpoch, tt.targetEpoch, tt.validatorIdx, spanMap)
		if err != nil {
			t.Fatalf("Failed to update span: %v", err)
		}
		if err := detector.slashingDetector.SlasherDB.SaveValidatorSpansMap(tt.validatorIdx, spanMap); err != nil {
			t.Fatalf("Couldnt save span map for validator id: %d", tt.validatorIdx)
		}
		if st != tt.slashingTargetEpoch {
			t.Fatalf("Expected slashing target: %d got: %d", tt.slashingTargetEpoch, st)
		}
		sm, err := detector.slashingDetector.SlasherDB.ValidatorSpansMap(tt.validatorIdx)
		if err != nil {
			t.Fatalf("Failed to retrieve span: %v", err)
		}
		if sm == nil || !proto.Equal(sm, tt.resultSpanMap) {
			t.Fatalf("Get should return validator span map: %v got: %v", tt.resultSpanMap, sm)
		}
	}
}

func TestServer_UpdateMinEpochSpan(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(t, c)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	detector := AttDetector{&detection.SlashingDetector{
		SlasherDB: dbs,
	}}
	for _, tt := range spanTestsMin {
		spanMap, err := detector.slashingDetector.SlasherDB.ValidatorSpansMap(tt.validatorIdx)
		if err != nil {
			t.Fatal(err)
		}
		st, spanMap, err := detector.DetectSurroundingAttestation(ctx, tt.sourceEpoch, tt.targetEpoch, tt.validatorIdx, spanMap)
		if err != nil {
			t.Fatalf("Failed to update span: %v", err)
		}
		if err := detector.slashingDetector.SlasherDB.SaveValidatorSpansMap(tt.validatorIdx, spanMap); err != nil {
			t.Fatalf("Couldnt save span map for validator id: %d", tt.validatorIdx)
		}
		if st != tt.slashingTargetEpoch {
			t.Fatalf("Expected slashing target: %d got: %d", tt.slashingTargetEpoch, st)
		}
		sm, err := detector.slashingDetector.SlasherDB.ValidatorSpansMap(tt.validatorIdx)
		if err != nil {
			t.Fatalf("Failed to retrieve span: %v", err)
		}
		if sm == nil || !proto.Equal(sm, tt.resultSpanMap) {
			t.Fatalf("Get should return validator span map: %v got: %v", tt.resultSpanMap, sm)
		}
	}
}

func TestServer_FailToUpdate(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(t, c)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	detector := AttDetector{&detection.SlashingDetector{
		SlasherDB: dbs,
	}}
	spanTestsFail := spanMapTestStruct{
		validatorIdx:        0,
		sourceEpoch:         0,
		slashingTargetEpoch: 0,
		targetEpoch:         params.BeaconConfig().WeakSubjectivityPeriod + 1,
		resultSpanMap: &slashpb.EpochSpanMap{
			EpochSpanMap: map[uint64]*slashpb.MinMaxEpochSpan{
				4: {MinEpochSpan: 0, MaxEpochSpan: 2},
				5: {MinEpochSpan: 0, MaxEpochSpan: 1},
			},
		},
	}
	spanMap, err := detector.slashingDetector.SlasherDB.ValidatorSpansMap(spanTestsFail.validatorIdx)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := detector.DetectSurroundingAttestation(ctx, spanTestsFail.sourceEpoch, spanTestsFail.targetEpoch, spanTestsFail.validatorIdx, spanMap); err == nil {
		t.Fatalf("Update should not support diff greater then weak subjectivity period: %v ", params.BeaconConfig().WeakSubjectivityPeriod)
	}
	if _, _, err := detector.DetectSurroundedAttestations(ctx, spanTestsFail.sourceEpoch, spanTestsFail.targetEpoch, spanTestsFail.validatorIdx, spanMap); err == nil {
		t.Fatalf("Update should not support diff greater then weak subjectivity period: %v ", params.BeaconConfig().WeakSubjectivityPeriod)
	}

}

func TestServer_SlashDoubleAttestation(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(t, c)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	detector := AttDetector{&detection.SlashingDetector{
		SlasherDB: dbs,
	}}
	ia1 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig2"),
		Data: &ethpb.AttestationData{
			Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block1"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}
	ia2 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig1"),
		Data: &ethpb.AttestationData{
			Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block2"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}
	want := &ethpb.AttesterSlashing{
		Attestation_1: ia2,
		Attestation_2: ia1,
	}

	if _, err := detector.DetectAttestationForSlashings(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if err := detector.slashingDetector.SlasherDB.SaveIndexedAttestation(ia1); err != nil {
		t.Fatalf("Save indexed attestation failed: %v", err)
	}
	sr, err := detector.DetectAttestationForSlashings(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr) != 1 {
		t.Errorf("Should return 1 slashing proof: %v", sr)
	}
	if !proto.Equal(sr[0], want) {
		t.Errorf("Wanted slashing proof: %v got: %v", want, sr[0])

	}
}

func TestServer_SlashTripleAttestation(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(t, c)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	detector := AttDetector{&detection.SlashingDetector{
		SlasherDB: dbs,
	}}
	ia1 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig1"),
		Data: &ethpb.AttestationData{
			Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block1"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}
	ia2 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig2"),
		Data: &ethpb.AttestationData{
			Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block2"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}
	ia3 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig3"),
		Data: &ethpb.AttestationData{
			Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block3"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}
	want1 := &ethpb.AttesterSlashing{
		Attestation_1: ia3,
		Attestation_2: ia1,
	}
	want2 := &ethpb.AttesterSlashing{
		Attestation_1: ia3,
		Attestation_2: ia2,
	}

	if _, err := detector.DetectAttestationForSlashings(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if err := detector.slashingDetector.SlasherDB.SaveIndexedAttestation(ia1); err != nil {
		t.Fatalf("Save indexed attestation failed: %v", err)
	}
	_, err := detector.DetectAttestationForSlashings(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if err := detector.slashingDetector.SlasherDB.SaveIndexedAttestation(ia2); err != nil {
		t.Fatalf("Save indexed attestation failed: %v", err)
	}
	sr, err := detector.DetectAttestationForSlashings(ctx, ia3)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(sr) != 2 {
		t.Errorf("Should return 1 slashing proof: %v", sr)
	}
	if !proto.Equal(sr[0], want1) {
		t.Errorf("Wanted slashing proof: %v got: %v", want1, sr[0])

	}
	if !proto.Equal(sr[1], want2) {
		t.Errorf("Wanted slashing proof: %v got: %v", want2, sr[0])

	}
}

func TestServer_DontSlashSameAttestation(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(t, c)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	detector := AttDetector{&detection.SlashingDetector{
		SlasherDB: dbs,
	}}
	ia1 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig1"),
		Data: &ethpb.AttestationData{
			Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block1"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}

	if _, err := detector.DetectAttestationForSlashings(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if err := detector.slashingDetector.SlasherDB.SaveIndexedAttestation(ia1); err != nil {
		t.Fatalf("Save indexed attestation failed: %v", err)
	}
	sr, err := detector.DetectAttestationForSlashings(ctx, ia1)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr) != 0 {
		t.Errorf("Should not return slashing proof for same attestation: %v", sr)
	}
}

func TestServer_DontSlashDifferentTargetAttestation(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(t, c)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	detector := AttDetector{&detection.SlashingDetector{
		SlasherDB: dbs,
	}}
	ia1 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig2"),
		Data: &ethpb.AttestationData{
			Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block1"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}
	ia2 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig1"),
		Data: &ethpb.AttestationData{
			Slot:            4*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block2"),
			Source:          &ethpb.Checkpoint{Epoch: 3},
			Target:          &ethpb.Checkpoint{Epoch: 4},
		},
	}

	if _, err := detector.DetectAttestationForSlashings(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if err := detector.slashingDetector.SlasherDB.SaveIndexedAttestation(ia1); err != nil {
		t.Fatalf("Save indexed attestation failed: %v", err)
	}
	sr, err := detector.DetectAttestationForSlashings(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr) != 0 {
		t.Errorf("Should not return slashing proof for different epoch attestation: %v", sr)
	}
}

func TestServer_DontSlashSameAttestationData(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(t, c)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	detector := AttDetector{&detection.SlashingDetector{
		SlasherDB: dbs,
	}}
	ad := &ethpb.AttestationData{
		Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
		CommitteeIndex:  0,
		BeaconBlockRoot: []byte("block1"),
		Source:          &ethpb.Checkpoint{Epoch: 2},
		Target:          &ethpb.Checkpoint{Epoch: 3},
	}
	ia1 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig2"),
		Data:             ad,
	}
	ia2 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig1"),
		Data:             ad,
	}

	if _, err := detector.DetectAttestationForSlashings(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if err := detector.slashingDetector.SlasherDB.SaveIndexedAttestation(ia1); err != nil {
		t.Fatalf("Save indexed attestation failed: %v", err)
	}
	sr, err := detector.DetectAttestationForSlashings(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr) != 0 {
		t.Errorf("Should not return slashing proof for same data: %v", sr)
	}
}

func TestServer_SlashSurroundedAttestation(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(t, c)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	detector := AttDetector{&detection.SlashingDetector{
		SlasherDB: dbs,
	}}
	ia1 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig2"),
		Data: &ethpb.AttestationData{
			Slot:            4*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block1"),
			Source:          &ethpb.Checkpoint{Epoch: 1},
			Target:          &ethpb.Checkpoint{Epoch: 4},
		},
	}
	ia2 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig1"),
		Data: &ethpb.AttestationData{
			Slot:            4*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block2"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}
	want := &ethpb.AttesterSlashing{
		Attestation_1: ia2,
		Attestation_2: ia1,
	}

	if _, err := detector.DetectAttestationForSlashings(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if err := detector.slashingDetector.SlasherDB.SaveIndexedAttestation(ia1); err != nil {
		t.Fatalf("Save indexed attestation failed: %v", err)
	}
	sr, err := detector.DetectAttestationForSlashings(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(sr) != 1 {
		t.Fatalf("Should return 1 slashing proof: %v", sr)
	}
	if !proto.Equal(sr[0], want) {
		t.Errorf("Wanted slashing proof: %v got: %v", want, sr[0])

	}
}

func TestServer_SlashSurroundAttestation(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(t, c)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	detector := AttDetector{&detection.SlashingDetector{
		SlasherDB: dbs,
	}}
	ia1 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig2"),
		Data: &ethpb.AttestationData{
			Slot:            4*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block1"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}
	ia2 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig1"),
		Data: &ethpb.AttestationData{
			Slot:            4*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block2"),
			Source:          &ethpb.Checkpoint{Epoch: 1},
			Target:          &ethpb.Checkpoint{Epoch: 4},
		},
	}
	want := &ethpb.AttesterSlashing{
		Attestation_1: ia2,
		Attestation_2: ia1,
	}

	if _, err := detector.DetectAttestationForSlashings(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if err := detector.slashingDetector.SlasherDB.SaveIndexedAttestation(ia1); err != nil {
		t.Fatalf("Save indexed attestation failed: %v", err)
	}
	sr, err := detector.DetectAttestationForSlashings(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(sr) != 1 {
		t.Fatalf("Should return 1 slashing proof: %v", sr)
	}
	if !proto.Equal(sr[0], want) {
		t.Errorf("Wanted slashing proof: %v got: %v", want, sr[0])

	}
}

func TestServer_DontSlashValidAttestations(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(t, c)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	detector := AttDetector{&detection.SlashingDetector{
		SlasherDB: dbs,
	}}
	ia1 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig2"),
		Data: &ethpb.AttestationData{
			Slot:            5*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block1"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 4},
		},
	}
	ia2 := &ethpb.IndexedAttestation{
		AttestingIndices: []uint64{0},
		Signature:        []byte("sig1"),
		Data: &ethpb.AttestationData{
			Slot:            5*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block2"),
			Source:          &ethpb.Checkpoint{Epoch: 3},
			Target:          &ethpb.Checkpoint{Epoch: 5},
		},
	}

	if _, err := detector.DetectAttestationForSlashings(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if err := detector.slashingDetector.SlasherDB.SaveIndexedAttestation(ia1); err != nil {
		t.Fatalf("Save indexed attestation failed: %v", err)
	}
	sr, err := detector.DetectAttestationForSlashings(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(sr) != 0 {
		t.Errorf("Should not return slashing proof for same data: %v", sr)
	}
}

func TestServer_Store_100_Attestations(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	c := cli.NewContext(app, set, nil)
	dbs := db.SetupSlasherDB(t, c)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	detector := AttDetector{&detection.SlashingDetector{
		SlasherDB: dbs,
	}}
	var cb []uint64
	for i := uint64(0); i < 100; i++ {
		cb = append(cb, i)
	}
	ia1 := &ethpb.IndexedAttestation{
		AttestingIndices: cb,
		Data: &ethpb.AttestationData{
			CommitteeIndex:  0,
			BeaconBlockRoot: make([]byte, 32),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 4},
		},
	}
	for i := uint64(0); i < 100; i++ {
		ia1.Data.Target.Epoch = i + 1
		ia1.Data.Source.Epoch = i
		t.Logf("In Loop: %d", i)
		ia1.Data.Slot = (i + 1) * params.BeaconConfig().SlotsPerEpoch
		root := []byte(strconv.Itoa(int(i)))
		ia1.Data.BeaconBlockRoot = append(root, ia1.Data.BeaconBlockRoot[len(root):]...)
		if _, err := detector.DetectAttestationForSlashings(ctx, ia1); err != nil {
			t.Errorf("Could not call RPC method: %v", err)
		}
	}

	s, err := dbs.Size()
	if err != nil {
		t.Error(err)
	}
	t.Logf("DB size is: %d", s)

}
