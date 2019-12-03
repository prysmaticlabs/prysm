package rpc

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashpb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/slasher/db"
)

func TestServer_IsSlashableBlock(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		SlasherDB: dbs,
	}
	psr := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      1,
			StateRoot: []byte("A"),
		},
		ValidatorIndex: 1,
	}
	psr2 := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      1,
			StateRoot: []byte("B"),
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
		t.Errorf("Should return 1 slashaing proof: %v", sr)
	}
	if !proto.Equal(sr.ProposerSlashing[0], want) {
		t.Errorf("wanted slashing proof: %v got: %v", want, sr.ProposerSlashing[0])

	}

}

func TestServer_IsNotSlashableBlock(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)

	slasherServer := &Server{
		SlasherDB: dbs,
	}
	psr := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      1,
			StateRoot: []byte("A"),
		},
		ValidatorIndex: 1,
	}
	psr2 := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      65,
			StateRoot: []byte("B"),
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
		t.Errorf("Should return 0 slashaing proof: %v", sr)
	}

}

func TestServer_DoubleBlock(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: dbs,
	}
	psr := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      1,
			StateRoot: []byte("A"),
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
		t.Errorf("Should return 0 slashaing proof: %v", sr)
	}

}

func TestServer_SameSlotSlashable(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: dbs,
	}
	psr := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      1,
			StateRoot: []byte("A"),
		},
		ValidatorIndex: 1,
	}
	psr2 := &slashpb.ProposerSlashingRequest{
		BlockHeader: &ethpb.BeaconBlockHeader{
			Slot:      1,
			StateRoot: []byte("B"),
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
		t.Errorf("Should return 1 slashaing proof: %v", sr)
	}
	if !proto.Equal(sr.ProposerSlashing[0], want) {
		t.Errorf("wanted slashing proof: %v got: %v", want, sr.ProposerSlashing[0])

	}
}

func TestServer_SlashDoubleAttestation(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: dbs,
	}
	ia1 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig2"),
		Data: &ethpb.AttestationData{
			Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block1"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}
	ia2 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig1"),
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
		t.Errorf("Should return 1 slashaing proof: %v", sr)
	}
	if !proto.Equal(sr.AttesterSlashing[0], want) {
		t.Errorf("Wanted slashing proof: %v got: %v", want, sr.AttesterSlashing[0])

	}
}

func TestServer_SlashTripleAttestation(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: dbs,
	}
	ia1 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig1"),
		Data: &ethpb.AttestationData{
			Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block1"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}
	ia2 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig2"),
		Data: &ethpb.AttestationData{
			Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block2"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}
	ia3 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig3"),
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
		t.Errorf("Should return 1 slashaing proof: %v", sr)
	}
	if !proto.Equal(sr.AttesterSlashing[0], want1) {
		t.Errorf("Wanted slashing proof: %v got: %v", want1, sr.AttesterSlashing[0])

	}
	if !proto.Equal(sr.AttesterSlashing[1], want2) {
		t.Errorf("Wanted slashing proof: %v got: %v", want2, sr.AttesterSlashing[0])

	}
}

func TestServer_DontSlashSameAttestation(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: dbs,
	}
	ia1 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig1"),
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
		t.Errorf("Should not return slashaing proof for same attestation: %v", sr)
	}
}

func TestServer_DontSlashDifferentTargetAttestation(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: dbs,
	}
	ia1 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig2"),
		Data: &ethpb.AttestationData{
			Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block1"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}
	ia2 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig1"),
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
		t.Errorf("Should not return slashaing proof for different epoch attestation: %v", sr)
	}
}

func TestServer_DontSlashSameAttestationData(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: dbs,
	}
	ad := &ethpb.AttestationData{
		Slot:            3*params.BeaconConfig().SlotsPerEpoch + 1,
		CommitteeIndex:  0,
		BeaconBlockRoot: []byte("block1"),
		Source:          &ethpb.Checkpoint{Epoch: 2},
		Target:          &ethpb.Checkpoint{Epoch: 3},
	}
	ia1 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig2"),
		Data:                ad,
	}
	ia2 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig1"),
		Data:                ad,
	}

	if _, err := slasherServer.IsSlashableAttestation(ctx, ia1); err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}
	sr, err := slasherServer.IsSlashableAttestation(ctx, ia2)
	if err != nil {
		t.Errorf("Could not call RPC method: %v", err)
	}

	if len(sr.AttesterSlashing) != 0 {
		t.Errorf("Should not return slashaing proof for same data: %v", sr)
	}
}

func TestServer_SlashSurroundedAttestation(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: dbs,
	}
	ia1 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig2"),
		Data: &ethpb.AttestationData{
			Slot:            4*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block1"),
			Source:          &ethpb.Checkpoint{Epoch: 1},
			Target:          &ethpb.Checkpoint{Epoch: 4},
		},
	}
	ia2 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig1"),
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
		t.Fatalf("Should return 1 slashaing proof: %v", sr.AttesterSlashing)
	}
	if !proto.Equal(sr.AttesterSlashing[0], want) {
		t.Errorf("Wanted slashing proof: %v got: %v", want, sr.AttesterSlashing[0])

	}
}

func TestServer_SlashSurroundAttestation(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: dbs,
	}
	ia1 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig2"),
		Data: &ethpb.AttestationData{
			Slot:            4*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block1"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 3},
		},
	}
	ia2 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig1"),
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
		t.Fatalf("Should return 1 slashaing proof: %v", sr.AttesterSlashing)
	}
	if !proto.Equal(sr.AttesterSlashing[0], want) {
		t.Errorf("Wanted slashing proof: %v got: %v", want, sr.AttesterSlashing[0])

	}
}

func TestServer_DontSlashValidAttestations(t *testing.T) {
	dbs := db.SetupSlasherDB(t)
	defer db.TeardownSlasherDB(t, dbs)
	ctx := context.Background()
	slasherServer := &Server{
		ctx:       ctx,
		SlasherDB: dbs,
	}
	ia1 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig2"),
		Data: &ethpb.AttestationData{
			Slot:            5*params.BeaconConfig().SlotsPerEpoch + 1,
			CommitteeIndex:  0,
			BeaconBlockRoot: []byte("block1"),
			Source:          &ethpb.Checkpoint{Epoch: 2},
			Target:          &ethpb.Checkpoint{Epoch: 4},
		},
	}
	ia2 := &ethpb.IndexedAttestation{
		CustodyBit_0Indices: []uint64{0},
		CustodyBit_1Indices: []uint64{},
		Signature:           []byte("sig1"),
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
		t.Errorf("Should not return slashaing proof for same data: %v", sr)
	}
}
