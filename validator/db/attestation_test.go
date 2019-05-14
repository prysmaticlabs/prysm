package db

import (
	"testing"

	pbp2p "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestCreateAttestation(t *testing.T) {
	attestation := &pbp2p.Attestation{Data: &pbp2p.AttestationData{Slot: 42}}
	attestationEnc, err := attestation.Marshal()
	att, err := createAttestation(attestationEnc)
	if err != nil {
		t.Fatalf("failed to unmarshal encoding: %v", err)
	}
	if att.Data.Slot != 42 {
		t.Fatal("incorrect attestation marshal/unmarshal")
	}
}

func TestSaveAndGetAttestation(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	fork := &pbp2p.Fork{}
	pubKey := getRandPubKey(t)
	attestation := &pbp2p.Attestation{Data: &pbp2p.AttestationData{Slot: 42}}

	err := db.SaveAttestation(fork, pubKey, attestation)
	if err != nil {
		t.Fatalf("can't save attestation: %v", err)
	}
	loadedAttestation, err := db.GetAttestation(fork, pubKey, attestation.Data.Slot/params.BeaconConfig().SlotsPerEpoch)
	if err != nil {
		t.Fatalf("can't read attestation: %v", err)
	}

	if loadedAttestation.Data.Slot != 42 {
		t.Fatalf("read the wrong attestation")
	}
}

func TestGetMaxAttestationEpoch(t *testing.T) {
	db := setupDB(t)
	defer teardownDB(t, db)

	fork := &pbp2p.Fork{}
	pubKey := getRandPubKey(t)
	// if there were no saves, then 0 is returned
	maxAttestationEpoch, err := db.getMaxAttestationEpoch(pubKey)
	if err != nil {
		t.Fatalf("can't get max attestation epoch: %v", err)
	}
	if maxAttestationEpoch != 0 {
		t.Fatalf("getMaxAttestationEpoch for new key return not 0")
	}

	// for multiple saves, the maximum epoch is returned
	attestation := &pbp2p.Attestation{Data: &pbp2p.AttestationData{Slot: 1}}
	err = db.SaveAttestation(fork, pubKey, attestation)
	if err != nil {
		t.Fatalf("can't get max attestation epoch: %v", err)
	}
	attestation = &pbp2p.Attestation{Data: &pbp2p.AttestationData{Slot: 10 * params.BeaconConfig().SlotsPerEpoch}}
	err = db.SaveAttestation(fork, pubKey, attestation)
	if err != nil {
		t.Fatalf("can't get max attestation epoch: %v", err)
	}
	maxAttestationEpoch, err = db.getMaxAttestationEpoch(pubKey)
	if err != nil {
		t.Fatalf("can't get max attestation epoch: %v", err)
	}
	if maxAttestationEpoch != 10 {
		t.Fatalf("getMaxAttestationEpoch for new key return not 0")
	}

	// maximum epoch returns to independence from the order of save
	attestation = &pbp2p.Attestation{Data: &pbp2p.AttestationData{Slot: 5 * params.BeaconConfig().SlotsPerEpoch}}
	err = db.SaveAttestation(fork, pubKey, attestation)
	if err != nil {
		t.Fatalf("can't get max attestation epoch: %v", err)
	}
	maxAttestationEpoch, err = db.getMaxAttestationEpoch(pubKey)
	if err != nil {
		t.Fatalf("can't get max attestation epoch: %v", err)
	}
	if maxAttestationEpoch != 10 {
		t.Fatalf("getMaxAttestationEpoch for new key return not 0")
	}
}
