package db

import (
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestSaveAndRemoveAttestations(t *testing.T) {
	db := startInMemoryBeaconDB(t)
	defer db.Close()

	attestation := types.NewAttestation(&pb.AggregatedAttestation{
		Slot:             1,
		Shard:            1,
		AttesterBitfield: []byte{'A'},
	})

	hash := attestation.Key()
	if err := db.SaveAttestation(attestation); err != nil {
		t.Fatalf("unable to save attestation %v", err)
	}

	exist, err := db.HasAttestation(hash)
	if err != nil {
		t.Fatalf("unable to check attestation %v", err)
	}
	if !exist {
		t.Fatal("saved attestation does not exist")
	}

	if err := db.RemoveAttestation(hash); err != nil {
		t.Fatalf("error removing attestation %v", err)
	}

	if _, err := db.GetAttestation(hash); err == nil {
		t.Fatalf("attestation is able to be retrieved")
	}
}

func TestSaveAndRemoveAttestationHashList(t *testing.T) {
	db := startInMemoryBeaconDB(t)
	defer db.Close()

	block := types.NewBlock(&pb.BeaconBlock{
		Slot: 0,
	})
	blockHash, err := block.Hash()
	if err != nil {
		t.Error(err)
	}

	attestation := types.NewAttestation(&pb.AggregatedAttestation{
		Slot:             1,
		Shard:            1,
		AttesterBitfield: []byte{'A'},
	})
	attestationHash := attestation.Key()

	if err := db.SaveAttestationHash(blockHash, attestationHash); err != nil {
		t.Fatalf("unable to save attestation hash %v", err)
	}

	exist, err := db.HasAttestationHash(blockHash, attestationHash)
	if err != nil {
		t.Fatalf("unable to check for attestation hash %v", err)
	}
	if !exist {
		t.Error("saved attestation hash does not exist")
	}

	// Negative test case: try with random attestation, exist should be false.
	exist, err = db.HasAttestationHash(blockHash, [32]byte{'A'})
	if err != nil {
		t.Fatalf("unable to check for attestation hash %v", err)
	}
	if exist {
		t.Error("attestation hash shouldn't have existed")
	}

	// Remove attestation list by deleting the block hash key.
	if err := db.RemoveAttestationHashList(blockHash); err != nil {
		t.Fatalf("remove attestation hash list failed %v", err)
	}

	// Negative test case: try with deleted block hash, this should fail.
	_, err = db.HasAttestationHash(blockHash, attestationHash)
	if err == nil {
		t.Error("attestation hash should't have existed in DB")
	}
}
