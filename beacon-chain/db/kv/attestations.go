package kv

import (
	"context"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"go.opencensus.io/trace"
)

// Attestation retrieval by attestation data root.
func (k *Store) Attestation(ctx context.Context, attDataRoot [32]byte) (*ethpb.Attestation, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Attestation")
	defer span.End()
	var att *ethpb.Attestation
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		enc := bkt.Get(attDataRoot[:])
		if enc == nil {
			return nil
		}
		att = &ethpb.Attestation{}
		return proto.Unmarshal(enc, att)
	})
	return att, err
}

// Attestations retrieves a list of attestations by filter criteria.
func (k *Store) Attestations(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.Attestation, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Attestations")
	defer span.End()
	atts := make([]*ethpb.Attestation, 0)
	err := k.db.Batch(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)

		// If no filter criteria are specified, return all attestations.
		if f == nil {
			return bkt.ForEach(func(k, v []byte) error {
				att := &ethpb.Attestation{}
				if err := proto.Unmarshal(v, att); err != nil {
					return err
				}
				atts = append(atts, att)
				return nil
			})
		}

		// Creates a list of indices from the passed in filter values, such as:
		// []byte("parent-root-0x2093923"), etc. to be used for looking up
		// block roots that were stored under each of those indices for O(1) lookup.
		indicesByBucket, err := createAttestationIndicesFromFilters(f)
		if err != nil {
			return errors.Wrap(err, "could not determine block lookup indices")
		}
		// Once we have a list of attestation data roots that correspond to each
		// lookup index, we find the intersection across all of them and use
		// that list of roots to lookup the attestations. These attestations will
		// meet the filter criteria.
		keys := sliceutil.IntersectionByteSlices(lookupValuesForIndices(indicesByBucket, tx)...)
		for i := 0; i < len(keys); i++ {
			encoded := bkt.Get(keys[i])
			att := &ethpb.Attestation{}
			if err := proto.Unmarshal(encoded, att); err != nil {
				return err
			}
			atts = append(atts, att)
		}
		return nil
	})
	return atts, err
}

// HasAttestation checks if an attestation by its attestation data root exists in the db.
func (k *Store) HasAttestation(ctx context.Context, attDataRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasAttestation")
	defer span.End()
	exists := false
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		exists = bkt.Get(attDataRoot[:]) != nil
		return nil
	})
	return exists
}

// DeleteAttestation by attestation data root.
// TODO(#3064): Add the ability for batch deletions.
func (k *Store) DeleteAttestation(ctx context.Context, attDataRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteAttestation")
	defer span.End()
	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		enc := bkt.Get(attDataRoot[:])
		if enc == nil {
			return nil
		}
		att := &ethpb.Attestation{}
		if err := proto.Unmarshal(enc, att); err != nil {
			return err
		}
		indicesByBucket := createAttestationIndicesFromData(att.Data, tx)
		if err := deleteValueForIndices(indicesByBucket, attDataRoot[:], tx); err != nil {
			return errors.Wrap(err, "could not delete root for DB indices")
		}
		return bkt.Delete(attDataRoot[:])
	})
}

// SaveAttestation to the db.
func (k *Store) SaveAttestation(ctx context.Context, att *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveAttestation")
	defer span.End()
	attDataRoot, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		return err
	}
	enc, err := proto.Marshal(att)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		indicesByBucket := createAttestationIndicesFromData(att.Data, tx)
		if err := updateValueForIndices(indicesByBucket, attDataRoot[:], tx); err != nil {
			return errors.Wrap(err, "could not update DB indices")
		}
		return bkt.Put(attDataRoot[:], enc)
	})
}

// SaveAttestations via batch updates to the db.
func (k *Store) SaveAttestations(ctx context.Context, atts []*ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveAttestations")
	defer span.End()
	encodedValues := make([][]byte, len(atts))
	keys := make([][]byte, len(atts))
	for i := 0; i < len(atts); i++ {
		enc, err := proto.Marshal(atts[i])
		if err != nil {
			return err
		}
		key, err := ssz.HashTreeRoot(atts[i].Data)
		if err != nil {
			return err
		}
		encodedValues[i] = enc
		keys[i] = key[:]
	}
	return k.db.Batch(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		for i := 0; i < len(atts); i++ {
			indicesByBucket := createAttestationIndicesFromData(atts[i].Data, tx)
			if err := updateValueForIndices(indicesByBucket, keys[i], tx); err != nil {
				return errors.Wrap(err, "could not update DB indices")
			}
			if err := bkt.Put(keys[i], encodedValues[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// createAttestationIndicesFromData takes in attestation data and returns
// a map of bolt DB index buckets corresponding to each particular key for indices for
// data, such as (shard indices bucket -> shard 5).
func createAttestationIndicesFromData(attData *ethpb.AttestationData, tx *bolt.Tx) map[string][]byte {
	indicesByBucket := make(map[string][]byte)
	buckets := make([][]byte, 0)
	indices := make([][]byte, 0)
	if attData.Source != nil {
		buckets = append(buckets, attestationSourceEpochIndicesBucket)
		indices = append(indices, uint64ToBytes(attData.Source.Epoch))
		if attData.Source.Root != nil && len(attData.Source.Root) > 0 {
			buckets = append(buckets, attestationSourceRootIndicesBucket)
			indices = append(indices, attData.Source.Root)
		}
	}
	if attData.Target != nil {
		buckets = append(buckets, attestationTargetEpochIndicesBucket)
		indices = append(indices, uint64ToBytes(attData.Target.Epoch))
		if attData.Target.Root != nil && len(attData.Target.Root) > 0 {
			buckets = append(buckets, attestationTargetRootIndicesBucket)
			indices = append(indices, attData.Target.Root)
		}
	}
	if attData.BeaconBlockRoot != nil && len(attData.BeaconBlockRoot) > 0 {
		buckets = append(buckets, attestationHeadBlockRootBucket)
		indices = append(indices, attData.BeaconBlockRoot)
	}
	for i := 0; i < len(buckets); i++ {
		indicesByBucket[string(buckets[i])] = indices[i]
	}
	return indicesByBucket
}

// createAttestationIndicesFromFilters takes in filter criteria and returns
// a list of of byte keys used to retrieve the values stored
// for the indices from the DB.
//
// For attestations, these are list of hash tree roots of attestation.Data
// objects. If a certain filter criterion does not apply to
// attestations, an appropriate error is returned.
func createAttestationIndicesFromFilters(f *filters.QueryFilter) (map[string][]byte, error) {
	indicesByBucket := make(map[string][]byte)
	for k, v := range f.Filters() {
		switch k {
		case filters.HeadBlockRoot:
			headBlockRoot := v.([]byte)
			indicesByBucket[string(attestationHeadBlockRootBucket)] = headBlockRoot
		case filters.SourceRoot:
			sourceRoot := v.([]byte)
			indicesByBucket[string(attestationSourceRootIndicesBucket)] = sourceRoot
		case filters.SourceEpoch:
			sourceEpoch := v.(uint64)
			indicesByBucket[string(attestationSourceEpochIndicesBucket)] = uint64ToBytes(sourceEpoch)
		case filters.TargetEpoch:
			targetEpoch := v.(uint64)
			indicesByBucket[string(attestationTargetEpochIndicesBucket)] = uint64ToBytes(targetEpoch)
		case filters.TargetRoot:
			targetRoot := v.([]byte)
			indicesByBucket[string(attestationTargetRootIndicesBucket)] = targetRoot
		default:
			return nil, fmt.Errorf("filter criterion %v not supported for attestations", k)
		}
	}
	return indicesByBucket, nil
}
