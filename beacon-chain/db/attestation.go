package db

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/types"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// SaveAttestation puts the attestation record into the beacon chain db.
func (db *BeaconDB) SaveAttestation(attestation *types.Attestation) error {
	hash := attestation.Key()
	encodedState, err := attestation.Marshal()
	if err != nil {
		return err
	}

	return db.update(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationBucket)

		return a.Put(hash[:], encodedState)
	})
}

// GetAttestation retrieves an attestation record from the db using its hash.
func (db *BeaconDB) GetAttestation(hash [32]byte) (*types.Attestation, error) {
	var attestation *types.Attestation
	err := db.view(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationBucket)

		enc := a.Get(hash[:])
		if enc == nil {
			return nil
		}

		var err error
		attestation, err = createAttestation(enc)
		return err
	})

	return attestation, err
}

// HasAttestation checks if the attestation exists.
func (db *BeaconDB) HasAttestation(hash [32]byte) bool {
	exists := false
	db.view(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationBucket)

		exists = a.Get(hash[:]) != nil
		return nil
	})
	return exists
}

func createAttestation(enc []byte) (*types.Attestation, error) {
	protoAttestation := &pb.AggregatedAttestation{}
	err := proto.Unmarshal(enc, protoAttestation)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}

	attestation := types.NewAttestation(protoAttestation)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate a block from the encoding: %v", err)
	}

	return attestation, nil
}
