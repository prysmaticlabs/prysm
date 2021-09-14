package kv

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/encoding/bytes"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

func unmarshalIndexedAttestation(ctx context.Context, enc []byte) (*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.unmarshalIndexedAttestation")
	defer span.End()
	protoIdxAtt := &ethpb.IndexedAttestation{}
	err := proto.Unmarshal(enc, protoIdxAtt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoded indexed attestation")
	}
	return protoIdxAtt, nil
}

// IndexedAttestationsForTarget accepts a target epoch and returns a list of
// indexed attestations.
// Returns nil if the indexed attestation does not exist with that target epoch.
func (s *Store) IndexedAttestationsForTarget(ctx context.Context, targetEpoch types.Epoch) ([]*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.IndexedAttestationsForTarget")
	defer span.End()
	var idxAtts []*ethpb.IndexedAttestation
	key := bytes.Bytes8(uint64(targetEpoch))
	err := s.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		for k, enc := c.Seek(key); k != nil && bytes.Equal(k[:8], key); k, enc = c.Next() {
			idxAtt, err := unmarshalIndexedAttestation(ctx, enc)
			if err != nil {
				return err
			}
			idxAtts = append(idxAtts, idxAtt)
		}
		return nil
	})
	return idxAtts, err
}

// IndexedAttestationsWithPrefix accepts a target epoch and signature bytes to find all attestations with the requested prefix.
// Returns nil if the indexed attestation does not exist with that target epoch.
func (s *Store) IndexedAttestationsWithPrefix(ctx context.Context, targetEpoch types.Epoch, sigBytes []byte) ([]*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.IndexedAttestationsWithPrefix")
	defer span.End()
	var idxAtts []*ethpb.IndexedAttestation
	key := encodeEpochSig(targetEpoch, sigBytes)
	err := s.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		for k, enc := c.Seek(key); k != nil && bytes.Equal(k[:len(key)], key); k, enc = c.Next() {
			idxAtt, err := unmarshalIndexedAttestation(ctx, enc)
			if err != nil {
				return err
			}
			idxAtts = append(idxAtts, idxAtt)
		}
		return nil
	})
	return idxAtts, err
}

// HasIndexedAttestation accepts an attestation and returns true if it exists in the DB.
func (s *Store) HasIndexedAttestation(ctx context.Context, att *ethpb.IndexedAttestation) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.HasIndexedAttestation")
	defer span.End()
	key := encodeEpochSig(att.Data.Target.Epoch, att.Signature)
	var hasAttestation bool
	// #nosec G104
	err := s.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		enc := bucket.Get(key)
		if enc == nil {
			return nil
		}
		hasAttestation = true
		return nil
	})

	return hasAttestation, err
}

// SaveIndexedAttestation accepts an indexed attestation and writes it to the DB.
func (s *Store) SaveIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.SaveIndexedAttestation")
	defer span.End()
	key := encodeEpochSig(idxAttestation.Data.Target.Epoch, idxAttestation.Signature)
	enc, err := proto.Marshal(idxAttestation)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}
	err = s.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		// if data is in s skip put and index functions
		val := bucket.Get(key)
		if val != nil {
			return nil
		}
		if err := bucket.Put(key, enc); err != nil {
			return errors.Wrap(err, "failed to save indexed attestation into historical bucket")
		}

		return err
	})
	return err
}

// SaveIndexedAttestations accepts multiple indexed attestations and writes them to the DB.
func (s *Store) SaveIndexedAttestations(ctx context.Context, idxAttestations []*ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.SaveIndexedAttestations")
	defer span.End()
	keys := make([][]byte, len(idxAttestations))
	marshaledAtts := make([][]byte, len(idxAttestations))
	for i, att := range idxAttestations {
		enc, err := proto.Marshal(att)
		if err != nil {
			return errors.Wrap(err, "failed to marshal")
		}
		keys[i] = encodeEpochSig(att.Data.Target.Epoch, att.Signature)
		marshaledAtts[i] = enc
	}

	err := s.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		for i, key := range keys {
			// if data is in s skip put and index functions
			val := bucket.Get(key)
			if val != nil {
				continue
			}
			if err := bucket.Put(key, marshaledAtts[i]); err != nil {
				return errors.Wrap(err, "failed to save indexed attestation into historical bucket")
			}
		}
		return nil
	})
	return err
}

// DeleteIndexedAttestation deletes a indexed attestation using the slot and its root as keys in their respective buckets.
func (s *Store) DeleteIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.DeleteIndexedAttestation")
	defer span.End()
	key := encodeEpochSig(idxAttestation.Data.Target.Epoch, idxAttestation.Signature)
	return s.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		enc := bucket.Get(key)
		if enc == nil {
			return nil
		}
		if err := bucket.Delete(key); err != nil {
			return errors.Wrap(err, "failed to delete indexed attestation from historical bucket")
		}
		return nil
	})
}

// PruneAttHistory removes all attestations from the DB older than the pruning epoch age.
func (s *Store) PruneAttHistory(ctx context.Context, currentEpoch, pruningEpochAge types.Epoch) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.pruneAttHistory")
	defer span.End()
	pruneFromEpoch := int64(currentEpoch) - int64(pruningEpochAge)
	if pruneFromEpoch <= 0 {
		return nil
	}

	return s.update(func(tx *bolt.Tx) error {
		attBucket := tx.Bucket(historicIndexedAttestationsBucket)
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		max := bytes.Bytes8(uint64(pruneFromEpoch))
		for k, _ := c.First(); k != nil && bytes.Compare(k[:8], max) <= 0; k, _ = c.Next() {
			if err := attBucket.Delete(k); err != nil {
				return errors.Wrap(err, "failed to delete indexed attestation from historical bucket")
			}
		}
		return nil
	})
}

// LatestIndexedAttestationsTargetEpoch returns latest target epoch in db
// returns 0 if there is no indexed attestations in db.
func (s *Store) LatestIndexedAttestationsTargetEpoch(ctx context.Context) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.LatestIndexedAttestationsTargetEpoch")
	defer span.End()
	var lt uint64
	err := s.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		k, _ := c.Last()
		if k == nil {
			return nil
		}
		lt = bytes.FromBytes8(k[:8])
		return nil
	})
	return lt, err
}
