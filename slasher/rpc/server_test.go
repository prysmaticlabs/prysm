package rpc

import (
	"context"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/params"
	testDB "github.com/prysmaticlabs/prysm/slasher/db/testing"
	"github.com/prysmaticlabs/prysm/slasher/db/types"
)

func TestServer_IsSlashableBlock(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()
	slasherServer := &Server{
		SlasherDB: db,
	}
	psr := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      1,
				StateRoot: []byte("A"),
			},
		},
		ValidatorIndex: 1,
	}
	psr2 := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      1,
				StateRoot: []byte("B"),
			},
		},
		ValidatorIndex: 1,
	}

	if _, err := slasherServer.IsSlashableBlock(ctx, psr); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableBlock(ctx, psr2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	want := &ethpb.ProposerSlashing{
		ProposerIndex: psr.ValidatorIndex,
		Header_1:      psr2.BlockHeader,
		Header_2:      psr.BlockHeader,
	}

	if len(sr.ProposerSlashing) != 1 {
		t.Errorf("Should return 1 slashing proof: %v", sr)
	}
	if !proto.Equal(sr.ProposerSlashing[0], want) {
		t.Errorf("wanted slashing proof: %v got: %v", want, sr.ProposerSlashing[0])

	}

}

func TestServer_IsNotSlashableBlock(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	defer testDB.TeardownSlasherDB(t, db)

	slasherServer := &Server{
		SlasherDB: db,
	}
	psr := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      1,
				StateRoot: []byte("A"),
			},
		},
		ValidatorIndex: 1,
	}
	psr2 := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      65,
				StateRoot: []byte("B"),
			},
		},
		ValidatorIndex: 1,
	}
	ctx := context.Background()

	if _, err := slasherServer.IsSlashableBlock(ctx, psr); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableBlock(ctx, psr2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr.ProposerSlashing) != 0 {
		t.Errorf("Should return 0 slashing proof: %v", sr)
	}

}

func TestServer_DoubleBlock(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: db,
	}
	psr := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      1,
				StateRoot: []byte("A"),
			},
		},
		ValidatorIndex: 1,
	}

	if _, err := slasherServer.IsSlashableBlock(ctx, psr); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableBlock(ctx, psr)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr.ProposerSlashing) != 0 {
		t.Errorf("Should return 0 slashing proof: %v", sr)
	}

}

func TestServer_SameSlotSlashable(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()

	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: db,
	}
	psr := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      1,
				StateRoot: []byte("A"),
			},
		},
		ValidatorIndex: 1,
	}
	psr2 := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.SignedBeaconBlockHeader{
			Header: &ethpb.BeaconBlockHeader{
				Slot:      1,
				StateRoot: []byte("B"),
			},
		},
		ValidatorIndex: 1,
	}
	want := &ethpb.ProposerSlashing{
		ProposerIndex: psr.ValidatorIndex,
		Header_1:      psr2.BlockHeader,
		Header_2:      psr.BlockHeader,
	}

	if _, err := slasherServer.IsSlashableBlock(ctx, psr); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableBlock(ctx, psr2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr.ProposerSlashing) != 1 {
		t.Errorf("Should return 1 slashing proof: %v", sr)
	}
	if !proto.Equal(sr.ProposerSlashing[0], want) {
		t.Errorf("wanted slashing proof: %v got: %v", want, sr.ProposerSlashing[0])

	}
	if err := slasherServer.SlasherDB.SaveProposerSlashing(ctx, types.Active, sr.ProposerSlashing[0]); err != nil {
		t.Errorf("Could not call db method: %v", err)
	}
	if sr, err = slasherServer.ProposerSlashings(ctx, &slashpb.SlashingStatusRequest{Status: slashpb.SlashingStatusRequest_Active}); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	ar, err := slasherServer.AttesterSlashings(ctx, &slashpb.SlashingStatusRequest{Status: slashpb.SlashingStatusRequest_Active})
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(ar.AttesterSlashing) > 0 {
		t.Errorf("Attester slashings with status 'active' should not be present in testDB.")
	}
	emptySlashingResponse, err := slasherServer.ProposerSlashings(ctx, &slashpb.SlashingStatusRequest{Status: slashpb.SlashingStatusRequest_Included})
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(emptySlashingResponse.ProposerSlashing) > 0 {
		t.Error("Proposer slashings with status 'included' should not be present in db")
	}
	if !proto.Equal(sr.ProposerSlashing[0], want) {
		t.Errorf("Wanted slashing proof: %v got: %v", want, sr.ProposerSlashing[0])
	}
}

func TestServer_SlashDoubleAttestation(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()

	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: db,
	}
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

	if _, err := slasherServer.IsSlashableAttestation(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableAttestation(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr.AttesterSlashing) != 1 {
		t.Errorf("Should return 1 slashing proof: %v", sr)
	}
	if !proto.Equal(sr.AttesterSlashing[0], want) {
		t.Errorf("Wanted slashing proof: %v got: %v", want, sr.AttesterSlashing[0])

	}
}

func TestServer_SlashTripleAttestation(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: db,
	}
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

	if _, err := slasherServer.IsSlashableAttestation(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	_, err := slasherServer.IsSlashableAttestation(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableAttestation(ctx, ia3)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(sr.AttesterSlashing) != 2 {
		t.Errorf("Should return 1 slashing proof: %v", sr)
	}
	if !proto.Equal(sr.AttesterSlashing[0], want1) {
		t.Errorf("Wanted slashing proof: %v got: %v", want1, sr.AttesterSlashing[0])

	}
	if !proto.Equal(sr.AttesterSlashing[1], want2) {
		t.Errorf("Wanted slashing proof: %v got: %v", want2, sr.AttesterSlashing[0])

	}
}

func TestServer_DontSlashSameAttestation(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: db,
	}
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

	if _, err := slasherServer.IsSlashableAttestation(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableAttestation(ctx, ia1)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr.AttesterSlashing) != 0 {
		t.Errorf("Should not return slashing proof for same attestation: %v", sr)
	}
}

func TestServer_DontSlashDifferentTargetAttestation(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: db,
	}
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

	if _, err := slasherServer.IsSlashableAttestation(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableAttestation(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr.AttesterSlashing) != 0 {
		t.Errorf("Should not return slashing proof for different epoch attestation: %v", sr)
	}
}

func TestServer_DontSlashSameAttestationData(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: db,
	}
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

	if _, err := slasherServer.IsSlashableAttestation(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableAttestation(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr.AttesterSlashing) != 0 {
		t.Errorf("Should not return slashing proof for same data: %v", sr)
	}
}

func TestServer_SlashSurroundedAttestation(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: db,
	}
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

	if _, err := slasherServer.IsSlashableAttestation(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableAttestation(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(sr.AttesterSlashing) != 1 {
		t.Fatalf("Should return 1 slashing proof: %v", sr.AttesterSlashing)
	}
	if !proto.Equal(sr.AttesterSlashing[0], want) {
		t.Errorf("Wanted slashing proof: %v got: %v", want, sr.AttesterSlashing[0])

	}
}

func TestServer_SlashSurroundAttestation(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: db,
	}
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

	if _, err := slasherServer.IsSlashableAttestation(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableAttestation(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(sr.AttesterSlashing) != 1 {
		t.Fatalf("Should return 1 slashing proof: %v", sr.AttesterSlashing)
	}
	if !proto.Equal(sr.AttesterSlashing[0], want) {
		t.Errorf("Wanted slashing proof: %v got: %v", want, sr.AttesterSlashing[0])

	}
	if err := slasherServer.SlasherDB.SaveAttesterSlashing(ctx, types.Active, sr.AttesterSlashing[0]); err != nil {
		t.Errorf("Could not call db method: %v", err)
	}
	pr, err := slasherServer.ProposerSlashings(ctx, &slashpb.SlashingStatusRequest{Status: slashpb.SlashingStatusRequest_Active})
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(pr.ProposerSlashing) > 0 {
		t.Errorf("Attester slashings with status 'active' should not be present in testDB.")
	}
	if sr, err = slasherServer.AttesterSlashings(ctx, &slashpb.SlashingStatusRequest{Status: slashpb.SlashingStatusRequest_Active}); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	emptySlashingResponse, err := slasherServer.AttesterSlashings(ctx, &slashpb.SlashingStatusRequest{Status: slashpb.SlashingStatusRequest_Included})
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(emptySlashingResponse.AttesterSlashing) > 0 {
		t.Error("Attester slashings with status 'included' should not be present in db")
	}
	if !proto.Equal(sr.AttesterSlashing[0], want) {
		t.Errorf("Wanted slashing proof: %v got: %v", want, sr.AttesterSlashing[0])
	}
}

func TestServer_DontSlashValidAttestations(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: db,
	}
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

	if _, err := slasherServer.IsSlashableAttestation(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableAttestation(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	if len(sr.AttesterSlashing) != 0 {
		t.Errorf("Should not return slashing proof for same data: %v", sr)
	}
}

func TestServer_Store_100_Attestations(t *testing.T) {
	db := testDB.SetupSlasherDB(t, false)
	defer testDB.TeardownSlasherDB(t, db)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: db,
	}
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
		if _, err := slasherServer.IsSlashableAttestation(ctx, ia1); err != nil {
			t.Errorf("Could not call RPC method: %v", err)
		}
	}

	s, err := db.Size()
	if err != nil {
		t.Error(err)
	}
	t.Logf("DB size is: %d", s)
}

func BenchmarkCheckAttestations(b *testing.B) {
	db := testDB.SetupSlasherDB(b, true)
	defer testDB.TeardownSlasherDB(b, db)
	context := context.Background()

	slasherServer := &Server{
		ctx:       context,
		SlasherDB: db,
	}
	var cb []uint64
	for i := uint64(0); i < 100; i++ {
		cb = append(cb, i)
	}
	ia1 := &ethpb.IndexedAttestation{
		AttestingIndices: cb,
		Signature:        make([]byte, 96),
		Data: &ethpb.AttestationData{
			CommitteeIndex:  0,
			BeaconBlockRoot: make([]byte, 32),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 4},
		},
	}
	b.ResetTimer()
	for i := uint64(0); i < uint64(b.N); i++ {
		ia1.Data.Target.Epoch = i + 1
		ia1.Data.Source.Epoch = i
		ia1.Data.Slot = (i + 1) * params.BeaconConfig().SlotsPerEpoch
		root := []byte(strconv.Itoa(int(i)))
		ia1.Data.BeaconBlockRoot = append(root, ia1.Data.BeaconBlockRoot[len(root):]...)
		if _, err := slasherServer.IsSlashableAttestation(context, ia1); err != nil {
			b.Errorf("Could not call RPC method: %v", err)
		}
	}
}
