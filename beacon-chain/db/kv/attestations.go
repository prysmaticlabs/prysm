package kv

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	"github.com/prysmaticlabs/prysm/beacon-chain/state/stateutil"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// AttestationsByDataRoot returns any (aggregated) attestations matching this data root.
func (kv *Store) AttestationsByDataRoot(ctx context.Context, attDataRoot [32]byte) ([]*ethpb.Attestation, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Attestation")
	defer span.End()
	var atts []*ethpb.Attestation
	err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		enc := bkt.Get(attDataRoot[:])
		if enc == nil {
			return nil
		}
		ac := &dbpb.AttestationContainer{}
		if err := decode(ctx, enc, ac); err != nil {
			return err
		}
		atts = ac.ToAttestations()
		return nil
	})
	if err != nil {
		traceutil.AnnotateError(span, err)
	}
	return atts, err
}

// Attestations retrieves a list of attestations by filter criteria.
func (kv *Store) Attestations(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.Attestation, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.Attestations")
	defer span.End()
	atts := make([]*ethpb.Attestation, 0)
	err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)

		// If no filter criteria are specified, return an error.
		if f == nil {
			return errors.New("must specify a filter criteria for retrieving attestations")
		}

		// Creates a list of indices from the passed in filter values, such as:
		// []byte("parent-root-0x2093923"), etc. to be used for looking up
		// block roots that were stored under each of those indices for O(1) lookup.
		indicesByBucket, err := createAttestationIndicesFromFilters(ctx, f)
		if err != nil {
			return errors.Wrap(err, "could not determine lookup indices")
		}
		// Once we have a list of attestation data roots that correspond to each
		// lookup index, we find the intersection across all of them and use
		// that list of roots to lookup the attestations. These attestations will
		// meet the filter criteria.
		keys := sliceutil.IntersectionByteSlices(lookupValuesForIndices(ctx, indicesByBucket, tx)...)
		for i := 0; i < len(keys); i++ {
			encoded := bkt.Get(keys[i])
			ac := &dbpb.AttestationContainer{}
			if err := decode(ctx, encoded, ac); err != nil {
				return err
			}
			atts = append(atts, ac.ToAttestations()...)
		}
		return nil
	})
	return atts, err
}

// HasAttestation checks if an attestation by its attestation data root exists in the db.
func (kv *Store) HasAttestation(ctx context.Context, attDataRoot [32]byte) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.HasAttestation")
	defer span.End()
	exists := false
	if err := kv.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		exists = bkt.Get(attDataRoot[:]) != nil
		return nil
	}); err != nil { // This view never returns an error, but we'll handle anyway for sanity.
		panic(err)
	}
	return exists
}

// DeleteAttestation by attestation data root.
func (kv *Store) DeleteAttestation(ctx context.Context, attDataRoot [32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteAttestation")
	defer span.End()
	return kv.DeleteAttestations(ctx, [][32]byte{attDataRoot})
}

// DeleteAttestations by attestation data roots.
func (kv *Store) DeleteAttestations(ctx context.Context, attDataRoots [][32]byte) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DeleteAttestations")
	defer span.End()

	return kv.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		for _, attDataRoot := range attDataRoots {
			enc := bkt.Get(attDataRoot[:])
			if enc == nil {
				continue
			}
			ac := &dbpb.AttestationContainer{}
			if err := decode(ctx, enc, ac); err != nil {
				return err
			}
			indicesByBucket := createAttestationIndicesFromData(ctx, ac.Data)
			if err := deleteValueForIndices(ctx, indicesByBucket, attDataRoot[:], tx); err != nil {
				return errors.Wrap(err, "could not delete root for DB indices")
			}
			if err := bkt.Delete(attDataRoot[:]); err != nil {
				return err
			}
		}
		return nil
	})
}

// SaveAttestation to the db.
func (kv *Store) SaveAttestation(ctx context.Context, att *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveAttestation")
	defer span.End()

	return kv.SaveAttestations(ctx, []*ethpb.Attestation{att})
}

// SaveAttestations via batch updates to the db.
func (kv *Store) SaveAttestations(ctx context.Context, atts []*ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveAttestations")
	defer span.End()

	err := kv.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		for _, att := range atts {
			// Aggregation bits are required to store attestations within the attestation container. Missing
			// this field may cause silent failures or unexpected results.
			if att.AggregationBits == nil {
				traceutil.AnnotateError(span, errors.New("attestation has nil aggregation bitlist"))
				continue
			}

			attDataRoot, err := stateutil.AttestationDataRoot(att.Data)
			if err != nil {
				return err
			}

			ac := &dbpb.AttestationContainer{
				Data: att.Data,
			}
			existingEnc := bkt.Get(attDataRoot[:])
			if existingEnc != nil {
				if err := decode(ctx, existingEnc, ac); err != nil {
					return err
				}
			}

			ac.InsertAttestation(att)

			enc, err := encode(ctx, ac)
			if err != nil {
				return err
			}

			indicesByBucket := createAttestationIndicesFromData(ctx, att.Data)
			if err := updateValueForIndices(ctx, indicesByBucket, attDataRoot[:], tx); err != nil {
				return errors.Wrap(err, "could not update DB indices")
			}

			if err := bkt.Put(attDataRoot[:], enc); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		traceutil.AnnotateError(span, err)
	}

	return err
}

// createAttestationIndicesFromData takes in attestation data and returns
// a map of bolt DB index buckets corresponding to each particular key for indices for
// data, such as (shard indices bucket -> shard 5).
func createAttestationIndicesFromData(ctx context.Context, attData *ethpb.AttestationData) map[string][]byte {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.createAttestationIndicesFromData")
	defer span.End()

	indicesByBucket := make(map[string][]byte)
	buckets := make([][]byte, 0)
	indices := make([][]byte, 0)
	if attData.Source != nil {
		buckets = append(buckets, attestationSourceEpochIndicesBucket)
		indices = append(indices, bytesutil.Uint64ToBytes(attData.Source.Epoch))
		if attData.Source.Root != nil && len(attData.Source.Root) > 0 {
			buckets = append(buckets, attestationSourceRootIndicesBucket)
			indices = append(indices, attData.Source.Root)
		}
	}
	if attData.Target != nil {
		buckets = append(buckets, attestationTargetEpochIndicesBucket)
		indices = append(indices, bytesutil.Uint64ToBytes(attData.Target.Epoch))
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
func createAttestationIndicesFromFilters(ctx context.Context, f *filters.QueryFilter) (map[string][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.createAttestationIndicesFromFilters")
	defer span.End()

	indicesByBucket := make(map[string][]byte)
	for k, v := range f.Filters() {
		switch k {
		case filters.HeadBlockRoot:
			headBlockRoot, ok := v.([]byte)
			if !ok {
				return nil, errors.New("headBlockRoot is not type []byte")
			}
			indicesByBucket[string(attestationHeadBlockRootBucket)] = headBlockRoot
		case filters.SourceRoot:
			sourceRoot, ok := v.([]byte)
			if !ok {
				return nil, errors.New("sourceRoot is not type []byte")
			}
			indicesByBucket[string(attestationSourceRootIndicesBucket)] = sourceRoot
		case filters.SourceEpoch:
			sourceEpoch, ok := v.(uint64)
			if !ok {
				return nil, errors.New("sourceEpoch is not type uint64")
			}
			indicesByBucket[string(attestationSourceEpochIndicesBucket)] = bytesutil.Uint64ToBytes(sourceEpoch)
		case filters.TargetEpoch:
			targetEpoch, ok := v.(uint64)
			if !ok {
				return nil, errors.New("targetEpoch is not type uint64")
			}
			indicesByBucket[string(attestationTargetEpochIndicesBucket)] = bytesutil.Uint64ToBytes(targetEpoch)
		case filters.TargetRoot:
			targetRoot, ok := v.([]byte)
			if !ok {
				return nil, errors.New("targetRoot is not type []byte")
			}
			indicesByBucket[string(attestationTargetRootIndicesBucket)] = targetRoot
		default:
			return nil, fmt.Errorf("filter criterion %v not supported for attestations", k)
		}
	}
	return indicesByBucket, nil
}
