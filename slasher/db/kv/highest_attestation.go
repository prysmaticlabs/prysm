package kv

import (
	"context"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

func (db *Store) HighestAttestation(ctx context.Context, validatorID uint64) (*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.IndexedAttestationsForTarget")
	defer span.End()
	key,err := db.ValidatorPubKey(ctx, validatorID)
	if err != nil {
		return nil, err
	}
	var idxAtt *ethpb.IndexedAttestation
	err = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(HighestAttestationBucket)
		idxAtt, err = unmarshalIndexedAttestation(ctx, b.Get(key))
		if err != nil {
			return err
		}
		return nil
	})
	return idxAtt, err
}

func (db *Store) SaveHighestAttestation(ctx context.Context, validatorIdx uint64, idxAttestation *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SavePubKey")
	defer span.End()

	enc, err := proto.Marshal(idxAttestation)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}

	key,err := db.ValidatorPubKey(ctx, validatorIdx)
	if err != nil {
		return err
	}
	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(validatorsPublicKeysBucket)
		if err := bucket.Put(key, enc); err != nil {
			return errors.Wrap(err, "failed to add highest attestation to slasher db.")
		}
		return nil
	})
	if err != nil {
		return nil
	}

	return nil
}
