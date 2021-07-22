package kv

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	slashpb "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
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
			enc, err := json.Marshal(value.(map[uint64]*slashpb.HighestAttestation))
			if err != nil {
				return errors.Wrap(err, "failed to marshal")
			}
			dbKey := highestAttSetkeyBytes(key.(uint64))
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

// EnableHighestAttestationCache used to enable or disable highest attestation cache in tests.
func (s *Store) EnableHighestAttestationCache(enable bool) {
	s.highestAttCacheEnabled = enable
}

// HighestAttestation returns the highest calculated attestation for a validatorID
func (s *Store) HighestAttestation(ctx context.Context, validatorID uint64) (*slashpb.HighestAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.HighestAttestation")
	defer span.End()

	if s.highestAttCacheEnabled {
		h, ok := s.highestAttestationCache.Get(highestAttSetkey(validatorID))
		if ok && h[validatorID] != nil {
			return h[validatorID], nil
		}
	}

	key := highestAttSetkeyBytes(validatorID)
	var highestAtt *slashpb.HighestAttestation
	err := s.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(highestAttestationBucket)
		if enc := b.Get(key); enc != nil {
			set := map[uint64]*slashpb.HighestAttestation{}
			err := json.Unmarshal(enc, &set)
			if err != nil {
				return err
			}
			highestAtt = set[validatorID]
		}
		return nil
	})

	return highestAtt, err
}

// SaveHighestAttestation saves highest attestation for a validatorID.
func (s *Store) SaveHighestAttestation(ctx context.Context, highest *slashpb.HighestAttestation) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SaveHighestAttestation")
	defer span.End()

	if s.highestAttCacheEnabled {
		s.highestAttestationCache.Set(highestAttSetkey(highest.ValidatorId), highest)
		return nil
	}

	key := highestAttSetkeyBytes(highest.ValidatorId)
	set := map[uint64]*slashpb.HighestAttestation{}
	err := s.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(highestAttestationBucket)
		if enc := b.Get(key); enc != nil {
			err := json.Unmarshal(enc, &set)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	set[highest.ValidatorId] = highest
	enc, err := json.Marshal(set)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}
	err = s.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(highestAttestationBucket)
		if err := bucket.Put(key, enc); err != nil {
			return errors.Wrap(err, "failed to add highest attestation to slasher s.")
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func highestAttSetkeyBytes(validatorID uint64) []byte {
	return bytesutil.Uint64ToBytesBigEndian(highestAttSetkey(validatorID))
}

// divide validators by id into 1k-ish buckets (0-1000,1001-1999, etc).
func highestAttSetkey(validatorID uint64) uint64 {
	return validatorID / 1000
}
