package db

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"go.opencensus.io/trace"
)

// SaveAttestations in the db.
func (db *BeaconDB) SaveAttestations(ctx context.Context, atts []*ethpb.Attestation) error {
	for _, a := range atts {
		if err := db.SaveAttestation(ctx, a); err != nil {
			return err
		}
	}
	return nil
}

// SaveAttestation puts the attestation record into the beacon chain db.
func (db *BeaconDB) SaveAttestation(ctx context.Context, attestation *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "beaconDB.SaveAttestation")
	defer span.End()

	encodedAtt, err := proto.Marshal(attestation)
	if err != nil {
		return err
	}

	hash, err := hashutil.HashProto(attestation.Data)
	if err != nil {
		return err
	}

	return db.batch(func(tx *bolt.Tx) error {
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

		return a.Put(attTarget.BeaconBlockRoot, encodedAttTgt)
	})
}

// DeleteAttestation deletes the attestation record into the beacon chain db.
func (db *BeaconDB) DeleteAttestation(_ context.Context, hash [32]byte) error {
	return db.batch(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationBucket)
		return a.Delete(hash[:])
	})
}

// DeleteAttestationDeprecated deletes the attestation record into the beacon chain db.
// DEPRECATED: Use DeleteAttestation.
func (db *BeaconDB) DeleteAttestationDeprecated(attestation *ethpb.Attestation) error {
	hash, err := hashutil.HashProto(attestation.Data)
	if err != nil {
		return err
	}

	return db.DeleteAttestation(context.Background(), hash)
}

// Attestation retrieves an attestation record from the db using the hash of attestation.data.
func (db *BeaconDB) Attestation(_ context.Context, hash [32]byte) (*ethpb.Attestation, error) {
	return db.AttestationDeprecated(hash)
}

// AttestationDeprecated retrieves an attestation record from the db using the hash of attestation.data.
// DEPRECATED: Use Attestation.
func (db *BeaconDB) AttestationDeprecated(hash [32]byte) (*ethpb.Attestation, error) {
	var attestation *ethpb.Attestation
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
func (db *BeaconDB) Attestations(_ context.Context, _ *filters.QueryFilter) ([]*ethpb.Attestation, error) {
	return db.AttestationsDeprecated()
}

// AttestationsDeprecated retrieves all the attestation records from the db.
// These are the attestations that have not been seen on the beacon chain.
// DEPRECATED: Use Attestations.
func (db *BeaconDB) AttestationsDeprecated() ([]*ethpb.Attestation, error) {
	var attestations []*ethpb.Attestation
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

// AttestationTarget retrieves an attestation target record from the db using the hash of attestation.data.
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

// HasAttestation checks if the attestaiton exists.
func (db *BeaconDB) HasAttestation(_ context.Context, hash [32]byte) bool {
	exists := false
	// #nosec G104
	db.view(func(tx *bolt.Tx) error {
		a := tx.Bucket(attestationBucket)

		exists = a.Get(hash[:]) != nil
		return nil
	})
	return exists
}

// HasAttestationDeprecated checks if the attestation exists.
// DEPRECATED: Use HasAttestation.
func (db *BeaconDB) HasAttestationDeprecated(hash [32]byte) bool {
	return db.HasAttestation(context.Background(), hash)
}

func createAttestation(enc []byte) (*ethpb.Attestation, error) {
	protoAttestation := &ethpb.Attestation{}
	if err := proto.Unmarshal(enc, protoAttestation); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoAttestation, nil
}

func createAttestationTarget(enc []byte) (*pb.AttestationTarget, error) {
	protoAttTgt := &pb.AttestationTarget{}
	if err := proto.Unmarshal(enc, protoAttTgt); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoAttTgt, nil
}
