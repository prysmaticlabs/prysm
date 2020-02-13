package kv

import (
	"bytes"
	"context"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

func unmarshalIdxAtt(ctx context.Context, enc []byte) (*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.unmarshalIdxAtt")
	defer span.End()
	protoIdxAtt := &ethpb.IndexedAttestation{}
	err := proto.Unmarshal(enc, protoIdxAtt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoded indexed attestation")
	}
	return protoIdxAtt, nil
}

func unmarshalIdxAttKeys(ctx context.Context, enc []byte) ([][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.unmarshalCompressedIdxAttList")
	defer span.End()
	uint64Length := 8
	keyLength := params.BeaconConfig().BLSSignatureLength + uint64Length
	if len(enc)%keyLength != 0 {
		return nil, fmt.Errorf("data length in keys array: %d is not a multiple of keys length: %d ", len(enc), keyLength)
	}
	keys := make([][]byte, len(enc)/keyLength)
	for i := range keys {
		keys[i] = enc[i*keyLength : (i+1)*keyLength]
	}
	return keys, nil
}

// IdxAttsForTargetFromID accepts a epoch and validator index and returns a list of
// indexed attestations from that validator for the given target epoch.
// Returns nil if the indexed attestation does not exist.
func (db *Store) IdxAttsForTargetFromID(ctx context.Context, targetEpoch uint64, validatorID uint64) ([]*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.IdxAttsForTargetFromID")
	defer span.End()
	var idxAtts []*ethpb.IndexedAttestation

	err := db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(epochValidatorIdxAttsBucket)
		key := encodeEpochValidatorID(targetEpoch, validatorID)
		enc := bucket.Get(key)
		if enc == nil {
			return nil
		}
		IdxAttsList, err := unmarshalIdxAttKeys(ctx, enc)
		if err != nil {
			return err
		}

		for _, key := range IdxAttsList {
			idxAttBucket := tx.Bucket(historicIndexedAttestationsBucket)
			enc = idxAttBucket.Get(key)
			if enc == nil {
				continue
			}
			att, err := unmarshalIdxAtt(ctx, enc)
			if err != nil {
				return err
			}
			idxAtts = append(idxAtts, att)
		}
		return nil
	})
	return idxAtts, err
}

// IdxAttsForTarget accepts a target epoch and returns a list of
// indexed attestations.
// Returns nil if the indexed attestation does not exist with that target epoch.
func (db *Store) IdxAttsForTarget(ctx context.Context, targetEpoch uint64) ([]*ethpb.IndexedAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.IdxAttsForTarget")
	defer span.End()
	var idxAtts []*ethpb.IndexedAttestation
	key := bytesutil.Bytes8(targetEpoch)
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		for k, enc := c.Seek(key); k != nil && bytes.Equal(k[:8], key); k, _ = c.Next() {
			idxAtt, err := unmarshalIdxAtt(ctx, enc)
			if err != nil {
				return err
			}
			idxAtts = append(idxAtts, idxAtt)
		}
		return nil
	})
	return idxAtts, err
}

// LatestIndexedAttestationsTargetEpoch returns latest target epoch in db
// returns 0 if there is no indexed attestations in db.
func (db *Store) LatestIndexedAttestationsTargetEpoch(ctx context.Context) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.LatestIndexedAttestationsTargetEpoch")
	defer span.End()
	var lt uint64
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		k, _ := c.Last()
		if k == nil {
			return nil
		}
		lt = bytesutil.FromBytes8(k[:8])
		return nil
	})
	return lt, err
}

// LatestValidatorIdx returns latest validator id in db
// returns 0 if there is no validators in db.
func (db *Store) LatestValidatorIdx(ctx context.Context) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.LatestValidatorIdx")
	defer span.End()
	var lt uint64
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(epochValidatorIdxAttsBucket).Cursor()
		k, _ := c.Last()
		if k == nil {
			return nil
		}
		lt = bytesutil.FromBytes8(k[:8])
		return nil
	})
	return lt, err
}

// DoubleVotes looks up db for slashable attesting data that were preformed by the same validator.
func (db *Store) DoubleVotes(ctx context.Context, validatorIdx uint64, dataRoot []byte, origAtt *ethpb.IndexedAttestation) ([]*ethpb.AttesterSlashing, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.DoubleVotes")
	defer span.End()
	idxAtts, err := db.IdxAttsForTargetFromID(ctx, origAtt.Data.Target.Epoch, validatorIdx)
	if err != nil {
		return nil, err
	}
	if idxAtts == nil || len(idxAtts) == 0 {
		return nil, nil
	}

	var idxAttsToSlash []*ethpb.IndexedAttestation
	for _, att := range idxAtts {
		if att.Data == nil {
			continue
		}
		root, err := hashutil.HashProto(att.Data)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(root[:], dataRoot) {
			idxAttsToSlash = append(idxAttsToSlash, att)
		}
	}

	var as []*ethpb.AttesterSlashing
	for _, idxAtt := range idxAttsToSlash {
		as = append(as, &ethpb.AttesterSlashing{
			Attestation_1: origAtt,
			Attestation_2: idxAtt,
		})
	}
	return as, nil
}

// HasIndexedAttestation accepts an epoch and validator id and returns true if the indexed attestation exists.
func (db *Store) HasIndexedAttestation(ctx context.Context, targetEpoch uint64, validatorID uint64) (bool, error) {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.HasIndexedAttestation")
	defer span.End()
	key := encodeEpochValidatorID(targetEpoch, validatorID)
	var hasAttestation bool
	// #nosec G104
	err := db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(epochValidatorIdxAttsBucket)
		enc := bucket.Get(key)
		if enc == nil || len(enc) == 0 {
			return nil
		}
		hasAttestation = true
		return nil
	})
	return hasAttestation, err
}

// SaveIndexedAttestation accepts epoch and indexed attestation and writes it to disk.
func (db *Store) SaveIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.SaveIndexedAttestation")
	defer span.End()
	key := encodeEpochSig(idxAttestation.Data.Target.Epoch, idxAttestation.Signature)
	enc, err := proto.Marshal(idxAttestation)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}
	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		//if data is in db skip put and index functions
		val := bucket.Get(key)
		if val != nil {
			return nil
		}
		if err := saveEpochValidatorIdxAttList(ctx, idxAttestation, tx); err != nil {
			return errors.Wrap(err, "failed to save indices from indexed attestation")
		}
		if err := bucket.Put(key, enc); err != nil {
			return errors.Wrap(err, "failed to save indexed attestation into historical bucket")
		}

		return err
	})

	// Prune history to max size every PruneSlasherStoragePeriod epoch.
	if idxAttestation.Data.Source.Epoch%params.BeaconConfig().PruneSlasherStoragePeriod == 0 {
		wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
		if err = db.PruneAttHistory(ctx, idxAttestation.Data.Source.Epoch, wsPeriod); err != nil {
			return err
		}
	}
	return err
}

func saveEpochValidatorIdxAttList(ctx context.Context, idxAttestation *ethpb.IndexedAttestation, tx *bolt.Tx) error {
	bucket := tx.Bucket(epochValidatorIdxAttsBucket)
	idxAttKey := encodeEpochSig(idxAttestation.Data.Target.Epoch, idxAttestation.Signature)
	for _, valIdx := range idxAttestation.AttestingIndices {
		key := encodeEpochValidatorID(idxAttestation.Data.Target.Epoch, valIdx)
		enc := bucket.Get(key)
		if enc == nil {
			if err := bucket.Put(key, idxAttKey); err != nil {
				return errors.Wrap(err, "failed to save indexed attestation into historical bucket")
			}
		}
		keys, err := unmarshalIdxAttKeys(ctx, enc)
		if err != nil {
			return errors.Wrap(err, "failed to marshal")
		}
		for _, k := range keys {
			if bytes.Equal(k, idxAttKey) {
				return nil
			}
		}
		if err := bucket.Put(key, append(enc, idxAttKey...)); err != nil {
			return errors.Wrap(err, "failed to save indexed attestation into historical bucket")
		}
	}
	return nil
}

// DeleteIndexedAttestation deletes a indexed attestation using the slot and its root as keys in their respective buckets.
func (db *Store) DeleteIndexedAttestation(ctx context.Context, idxAttestation *ethpb.IndexedAttestation) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.DeleteIndexedAttestation")
	defer span.End()
	key := encodeEpochSig(idxAttestation.Data.Target.Epoch, idxAttestation.Signature)
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicIndexedAttestationsBucket)
		enc := bucket.Get(key)
		if enc == nil {
			return nil
		}
		if err := removeEpochValidatorIdxAttList(ctx, idxAttestation, tx); err != nil {
			return err
		}
		if err := bucket.Delete(key); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				return errors.Wrapf(rollbackErr, "failed to rollback after %v", err)
			}
			return errors.Wrap(err, "failed to delete indexed attestation from historical bucket")
		}
		return nil
	})
}

func removeEpochValidatorIdxAttList(ctx context.Context, idxAttestation *ethpb.IndexedAttestation, tx *bolt.Tx) error {
	idxAttKey := encodeEpochSig(idxAttestation.Data.Target.Epoch, idxAttestation.Signature)
	bucket := tx.Bucket(epochValidatorIdxAttsBucket)

	for _, valIdx := range idxAttestation.AttestingIndices {
		key := encodeEpochValidatorID(idxAttestation.Data.Target.Epoch, valIdx)
		enc := bucket.Get(key)
		if enc == nil {
			continue
		}
		keys, err := unmarshalIdxAttKeys(ctx, enc)
		if err != nil {
			return errors.Wrap(err, "failed to marshal")
		}
		for i, k := range keys {
			if bytes.Equal(k, idxAttKey) {
				keys = append(keys[:i], keys[i+1:]...)
				if err := bucket.Put(key, bytes.Join(keys, []byte{})); err != nil {
					return errors.Wrap(err, "failed to delete indexed attestation from historical bucket")
				}
			}
		}
	}
	return nil
}

// PruneAttHistory removes all attestations from the DB older than the pruning epoch age.
func (db *Store) PruneAttHistory(ctx context.Context, currentEpoch uint64, pruningEpochAge uint64) error {
	ctx, span := trace.StartSpan(ctx, "SlasherDB.pruneAttHistory")
	defer span.End()
	pruneFromEpoch := int64(currentEpoch) - int64(pruningEpochAge)
	if pruneFromEpoch <= 0 {
		return nil
	}

	return db.update(func(tx *bolt.Tx) error {
		attBucket := tx.Bucket(historicIndexedAttestationsBucket)
		c := tx.Bucket(historicIndexedAttestationsBucket).Cursor()
		max := bytesutil.Bytes8(uint64(pruneFromEpoch))
		for k, _ := c.First(); k != nil && bytes.Compare(k[:8], max) <= 0; k, _ = c.Next() {
			if err := attBucket.Delete(k); err != nil {
				return errors.Wrap(err, "failed to delete indexed attestation from historical bucket")
			}
		}

		idxBucket := tx.Bucket(epochValidatorIdxAttsBucket)
		c = tx.Bucket(epochValidatorIdxAttsBucket).Cursor()
		for k, _ := c.First(); k != nil && bytes.Compare(k[:8], max) <= 0; k, _ = c.Next() {
			if err := idxBucket.Delete(k); err != nil {
				return errors.Wrap(err, "failed to delete indexed attestation from epoch validatorID bucket")
			}
		}
		return nil
	})
}
