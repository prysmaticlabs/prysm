package db

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	att "github.com/prysmaticlabs/prysm/beacon-chain/core/attestations"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

// SaveAttestation puts the attestation record into the beacon chain db.
func (db *BeaconDB) SaveAttestation(attestation *pb.Attestation) error {
	hash := att.Key(attestation.Data)
	encodedState, err := proto.Marshal(attestation)
	if err != nil {
		return err
	}

	return db.update(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationBucket)

		return a.Put(hash[:], encodedState)
	})
}

// GetAttestation retrieves an attestation record from the db using its hash.
func (db *BeaconDB) GetAttestation(hash [32]byte) (*pb.Attestation, error) {
	var attestation *pb.Attestation
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

// SaveLatestAttestationsForValidator puts a list of latest attestations from a
// beacon chain validator into the key value store using the validator index
// to construct the key for retrieval.
func (db *BeaconDB) SaveLatestAttestationsForValidator(validatorIdx uint32, latestAtts *pb.LatestAttestations) error {
	encodedAtts, err := proto.Marshal(latestAtts)
	if err != nil {
		return err
	}

	return db.update(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationBucket)
		key := []byte(fmt.Sprintf("validator-%d-latest-atts", validatorIdx))
		return a.Put(key, encodedAtts)
	})
}

// GetLatestAttestationsForValidator returns the observed attestations by a validator at a
// certain index in the beacon state's registry.
func (db *BeaconDB) GetLatestAttestationsForValidator(validatorIdx uint32) (*pb.LatestAttestations, error) {
	var latestAtts *pb.LatestAttestations
	err := db.view(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationBucket)

		key := []byte(fmt.Sprintf("validator-%d-latest-atts", validatorIdx))
		enc := a.Get(key)
		if enc == nil {
			return nil
		}

		if err := proto.Unmarshal(enc, latestAtts); err != nil {
			return fmt.Errorf("failed to unmarshal encoding: %v", err)
		}
		return nil
	})

	return latestAtts, err
}

// HasAttestation checks if the attestation exists.
func (db *BeaconDB) HasAttestation(hash [32]byte) bool {
	exists := false
	// #nosec G104
	db.view(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationBucket)

		exists = a.Get(hash[:]) != nil
		return nil
	})
	return exists
}

func createAttestation(enc []byte) (*pb.Attestation, error) {
	protoAttestation := &pb.Attestation{}
	if err := proto.Unmarshal(enc, protoAttestation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}
	return protoAttestation, nil
}
