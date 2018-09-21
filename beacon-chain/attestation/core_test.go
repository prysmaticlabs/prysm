package attestation

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/blockchain"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/database"
)

type faultyDB struct{}

func (f *faultyDB) Get(k []byte) ([]byte, error) {
	return []byte{}, nil
}

func (f *faultyDB) Has(k []byte) (bool, error) {
	return true, nil
}

func (f *faultyDB) Put(k []byte, v []byte) error {
	return nil
}

func (f *faultyDB) Delete(k []byte) error {
	return nil
}

func (f *faultyDB) Close() {}

func (f *faultyDB) NewBatch() ethdb.Batch {
	return nil
}

func startInMemoryAttestationDB(t *testing.T) (*Handler, *database.DB) {
	config := &database.DBConfig{DataDir: "", Name: "", InMemory: true}
	db, err := database.NewDB(config)
	if err != nil {
		t.Fatalf("unable to setup db: %v", err)
	}
	handler, err := NewHandler(db.DB())
	if err != nil {
		t.Fatalf("unable to setup beacon chain: %v", err)
	}

	return handler, db
}

func TestSaveAndRemoveAttestations(t *testing.T) {
	b, db := startInMemoryAttestationDB(t)
	defer db.Close()

	attestation := types.NewAttestation(&pb.AggregatedAttestation{
		Slot:             1,
		ShardId:          1,
		AttesterBitfield: []byte{'A'},
	})

	hash := attestation.Key()
	if err := b.saveAttestation(attestation); err != nil {
		t.Fatalf("unable to save attestation %v", err)
	}

	exist, err := b.hasAttestation(hash)
	if err != nil {
		t.Fatalf("unable to check attestation %v", err)
	}
	if !exist {
		t.Fatal("saved attestation does not exist")
	}

	// Adding a different attestation with the same key
	newAttestation := types.NewAttestation(&pb.AggregatedAttestation{
		Slot:             2,
		ShardId:          2,
		AttesterBitfield: []byte{'B'},
	})

	key := blockchain.AttestationKey(hash)
	marshalled, err := proto.Marshal(newAttestation.Proto())
	if err != nil {
		t.Fatal(err)
	}

	if err := b.db.Put(key, marshalled); err != nil {
		t.Fatal(err)
	}

	returnedAttestation, err := b.getAttestation(hash)
	if err != nil {
		t.Fatalf("attestation is unable to be retrieved")
	}

	if returnedAttestation.SlotNumber() != newAttestation.SlotNumber() {
		t.Errorf("slotnumber does not match for saved and retrieved attestation")
	}

	if !bytes.Equal(returnedAttestation.AttesterBitfield(), newAttestation.AttesterBitfield()) {
		t.Errorf("attester bitfield does not match for saved and retrieved attester")
	}

	if err := b.removeAttestation(hash); err != nil {
		t.Fatalf("error removing attestation %v", err)
	}

	if _, err := b.getAttestation(hash); err == nil {
		t.Fatalf("attestation is able to be retrieved")
	}
}

func TestSaveAndRemoveAttestationHashList(t *testing.T) {
	b, db := startInMemoryAttestationDB(t)
	defer db.Close()

	block := types.NewBlock(&pb.BeaconBlock{
		SlotNumber: 0,
	})
	blockHash, err := block.Hash()
	if err != nil {
		t.Error(err)
	}

	attestation := types.NewAttestation(&pb.AggregatedAttestation{
		Slot:             1,
		ShardId:          1,
		AttesterBitfield: []byte{'A'},
	})
	attestationHash := attestation.Key()

	if err := b.saveAttestationHash(blockHash, attestationHash); err != nil {
		t.Fatalf("unable to save attestation hash %v", err)
	}

	exist, err := b.hasAttestationHash(blockHash, attestationHash)
	if err != nil {
		t.Fatalf("unable to check for attestation hash %v", err)
	}
	if !exist {
		t.Error("saved attestation hash does not exist")
	}

	// Negative test case: try with random attestation, exist should be false.
	exist, err = b.hasAttestationHash(blockHash, [32]byte{'A'})
	if err != nil {
		t.Fatalf("unable to check for attestation hash %v", err)
	}
	if exist {
		t.Error("attestation hash shouldn't have existed")
	}

	// Remove attestation list by deleting the block hash key.
	if err := b.removeAttestationHashList(blockHash); err != nil {
		t.Fatalf("remove attestation hash list failed %v", err)
	}

	// Negative test case: try with deleted block hash, this should fail.
	_, err = b.hasAttestationHash(blockHash, attestationHash)
	if err == nil {
		t.Error("attestation hash should't have existed in DB")
	}
}
