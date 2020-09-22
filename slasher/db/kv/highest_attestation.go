package kv

import (
	"context"
	"fmt"
	json "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/slasher/detection/attestations/types"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

func persistHighestAttestationCacheOnEviction(db *Store) func(key interface{}, value interface{}) {
	// We use a closure here so we can access the database itself
	// on the eviction of a span map from the cache. The function has the signature
	// required by the ristretto cache OnEvict method.
	// See https://godoc.org/github.com/dgraph-io/ristretto#Config.
	return func(key interface{}, value interface{}) {
		log.Tracef("Evicting highest attestation for validator: %d", key.(uint64))
		err := db.update(func(tx *bolt.Tx) error {
			enc, err := json.Marshal(value.(*types.HighestAttestation))
			if err != nil {
				return errors.Wrap(err, "failed to marshal")
			}

			dbKey := []byte(fmt.Sprintf("%d", key.(uint64)))

			bucket := tx.Bucket(highestAttestationBucket)
			if err := bucket.Put(dbKey, enc); err != nil {
				return errors.Wrap(err, "failed to add highest attestation to slasher db.")
			}
			return nil
		})
		if err != nil {
			log.Errorf("Failed to save highest attestation to db on cache eviction: %v", err)
		}
	}
}

// EnableSpanCache used to enable or disable span map cache in tests.
func (db *Store) EnableHighestAttestationCache(enable bool) {
	db.highestAttCacheEnabled = enable
}

func (db *Store) HighestAttestation(ctx context.Context, validatorID uint64) (*types.HighestAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.IndexedAttestationsForTarget")
	defer span.End()

	if db.highestAttCacheEnabled {
		h, ok := db.highestAttestationCache.Get(validatorID)
		if ok {
			return h, nil
		}
		return nil, nil // default
	}

	key := []byte(fmt.Sprintf("%d", validatorID))
	highestAtt := &types.HighestAttestation{HighestSourceEpoch: 0,HighestTargetEpoch:0} // default
	err := db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(highestAttestationBucket)
		if enc := b.Get(key); enc != nil {
			err := json.Unmarshal(enc, &highestAtt)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return highestAtt, err
}

func (db *Store) SaveHighestAttestation(ctx context.Context, validatorID uint64, highest *types.HighestAttestation) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SavePubKey")
	defer span.End()

	if db.highestAttCacheEnabled {
		db.highestAttestationCache.Set(validatorID, highest)
		return nil
	}

	enc, err := json.Marshal(highest)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}

	key := []byte(fmt.Sprintf("%d", validatorID))
	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(highestAttestationBucket)
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
