package kv

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

// Attestation retrieval by attestation data root.
func (k *Store) Attestation(ctx context.Context, attDataRoot [32]byte) (*ethpb.Attestation, error) {
	att := &ethpb.Attestation{}
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		enc := bkt.Get(attDataRoot[:])
		if enc == nil {
			return nil
		}
		return proto.Unmarshal(enc, att)
	})
	return att, err
}

// Attestations retrieves a list of attestations by filter criteria.
func (k *Store) Attestations(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.Attestation, error) {
	atts := make([]*ethpb.Attestation, 0)
	err := k.db.Batch(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		// Creates a list of indices from the passed in filter values, such as:
		// []byte("shard-5"), []byte("parent-root-0x2093923"), etc. to be used for looking up
		// attestation roots that were stored under each of those indices for O(1) lookup.
		indices := createIndicesFromFilters(f)
		indicesBkt := tx.Bucket(attestationIndicesBucket)
		// Once we have a list of attestation roots that correspond to each
		// lookup index, we find the intersection across all of them and use
		// that list of roots to lookup the attestations. These attestations will
		// meet the filter criteria.
		keys := sliceutil.IntersectionByteSlices(lookupValuesForIndices(indices, indicesBkt)...)
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
// TODO(#3018): Add the ability for batch deletions.
func (k *Store) DeleteAttestation(ctx context.Context, attDataRoot [32]byte) error {
	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		// TODO(#3018): Also delete the keys from the indices list. Add delete attestations batch.
		return bkt.Delete(attDataRoot[:])
	})
}

// SaveAttestation to the db.
func (k *Store) SaveAttestation(ctx context.Context, att *ethpb.Attestation) error {
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
		// Do not save if already saved.
		if bkt.Get(attDataRoot[:]) != nil {
			return nil
		}
		indices := [][]byte{
			append(shardIdx, uint64ToBytes(att.Data.Crosslink.Shard)...),
			append(parentRootIdx, att.Data.Crosslink.ParentRoot...),
		}
		indicesBkt := tx.Bucket(attestationIndicesBucket)
		if err := updateIndices(indices, attDataRoot[:], indicesBkt); err != nil {
			return errors.Wrap(err, "could not update DB indices")
		}
		return bkt.Put(attDataRoot[:], enc)
	})
}

// SaveAttestations via batch updates to the db.
func (k *Store) SaveAttestations(ctx context.Context, atts []*ethpb.Attestation) error {
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
			// Do not save if already saved.
			if bkt.Get(keys[i]) != nil {
				return nil
			}
			indices := [][]byte{
				append(shardIdx, uint64ToBytes(atts[i].Data.Crosslink.Shard)...),
				append(parentRootIdx, atts[i].Data.Crosslink.ParentRoot...),
			}
			indicesBkt := tx.Bucket(attestationIndicesBucket)
			if err := updateIndices(indices, keys[i], indicesBkt); err != nil {
				return errors.Wrap(err, "could not update DB indices")
			}
			if err := bkt.Put(keys[i], encodedValues[i]); err != nil {
				return err
			}
		}
		return nil
	})
}
