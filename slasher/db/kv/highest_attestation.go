package kv

import (
	"context"
	"fmt"
	json "github.com/json-iterator/go"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

type HighestAttestation struct {
	HighestSourceEpoch uint64
	HighestTargetEpoch uint64
}

func (db *Store) HighestAttestation(ctx context.Context, validatorID uint64) (sourceEpoch uint64, targetEpoch uint64, err error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.IndexedAttestationsForTarget")
	defer span.End()
	key := []byte(fmt.Sprintf("%d", validatorID))
	idxAtt := &HighestAttestation{HighestSourceEpoch:0,HighestTargetEpoch:0} // default
	err = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(HighestAttestationBucket)
		if enc := b.Get(key); enc != nil {
			err := json.Unmarshal(enc, &idxAtt)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return idxAtt.HighestSourceEpoch, idxAtt.HighestTargetEpoch, err
}

func (db *Store) SaveHighestAttestation(ctx context.Context, validatorID uint64, sourceEpoch uint64, targetEpoch uint64) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SavePubKey")
	defer span.End()

	enc, err := json.Marshal(&HighestAttestation{HighestTargetEpoch:targetEpoch, HighestSourceEpoch:sourceEpoch})
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}

	key := []byte(fmt.Sprintf("%d", validatorID))
	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(HighestAttestationBucket)
		if err := bucket.Put(key, enc); err != nil {
			return errors.Wrap(err, "failed to add highest attestation to slasher db.")
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
