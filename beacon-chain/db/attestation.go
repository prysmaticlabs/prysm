package db

import (
	"context"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"go.opencensus.io/trace"
)

// SaveAttestation puts the attestation record into the beacon chain db.
func (db *BeaconDB) SaveAttestation(ctx context.Context, attestation *pb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "beaconDB.SaveAttestation")
	defer span.End()

	encodedAtt, err := proto.Marshal(attestation)
	if err != nil {
		return err
	}
	hash := hashutil.Hash(encodedAtt)

	return db.update(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationBucket)

		return a.Put(hash[:], encodedAtt)
	})
}

// SaveAttestationTarget puts the attestation target record into the beacon chain db.
func (db *BeaconDB) SaveAttestationTarget(ctx context.Context, attTarget *pb.AttestationTarget) error {
	ctx, span := trace.StartSpan(ctx, "beaconDB.SaveAttestationTarget")
	defer span.End()

	encodedAttTgt, err := proto.Marshal(attTarget)
	if err != nil {
		return err
	}

	return db.update(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationTargetBucket)

		return a.Put(attTarget.BlockRoot, encodedAttTgt)
	})
}

// DeleteAttestation deletes the attestation record into the beacon chain db.
func (db *BeaconDB) DeleteAttestation(attestation *pb.Attestation) error {
	hash, err := hashutil.HashProto(attestation)
	if err != nil {
		return err
	}

	return db.update(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationBucket)

		return a.Delete(hash[:])
	})
}

// Attestation retrieves an attestation record from the db using its hash.
func (db *BeaconDB) Attestation(hash [32]byte) (*pb.Attestation, error) {
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

// Attestations retrieves all the attestation records from the db.
// These are the attestations that have not been seen on the beacon chain.
func (db *BeaconDB) Attestations() ([]*pb.Attestation, error) {
	var attestations []*pb.Attestation
	err := db.view(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationBucket)

		if err := a.ForEach(func(k, v []byte) error {
			attestation, err := createAttestation(v)
			if err != nil {
				return err
			}
			attestations = append(attestations, attestation)
			return nil
		}); err != nil {
			return err
		}
		return nil
	})

	return attestations, err
}

// AttestationTarget retrieves an attestation target record from the db using its hash.
func (db *BeaconDB) AttestationTarget(hash [32]byte) (*pb.AttestationTarget, error) {
	var attTgt *pb.AttestationTarget
	err := db.view(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationTargetBucket)

		enc := a.Get(hash[:])
		if enc == nil {
			return nil
		}

		var err error
		attTgt, err = createAttestationTarget(enc)
		return err
	})

	return attTgt, err
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

func createAttestationTarget(enc []byte) (*pb.AttestationTarget, error) {
	protoAttTgt := &pb.AttestationTarget{}
	if err := proto.Unmarshal(enc, protoAttTgt); err != nil {
		return nil, fmt.Errorf("failed to unmarshal encoding: %v", err)
	}
	return protoAttTgt, nil
}
