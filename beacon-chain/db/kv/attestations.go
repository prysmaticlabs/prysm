package kv

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/sliceutil"
)

// Attestation retrieval by root.
func (k *Store) Attestation(ctx context.Context, attRoot [32]byte) (*ethpb.Attestation, error) {
	att := &ethpb.Attestation{}
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		enc := bkt.Get(attRoot[:])
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

		// Once we have a list of attestation roots that correspond to each
		// lookup index, we find the intersection across all of them and use
		// that list of roots to lookup the attestations. These attestations will
		// meet the filter criteria.
		keys := sliceutil.TotalIntersectionByteSlices(valuesForIndices(indices, bkt))
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

// HasAttestation checks if an attestation by root exists in the db.
func (k *Store) HasAttestation(ctx context.Context, attRoot [32]byte) bool {
	exists := false
	// #nosec G104. Always returns nil.
	k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		exists = bkt.Get(attRoot[:]) != nil
		return nil
	})
	return exists
}

// DeleteAttestation by root.
func (k *Store) DeleteAttestation(ctx context.Context, attRoot [32]byte) error {
	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		// TODO(#3018): Also delete the keys from the indices list. Add delete attestations batch.
		return bkt.Delete(attRoot[:])
	})
}

// SaveAttestation to the db.
func (k *Store) SaveAttestation(ctx context.Context, att *ethpb.Attestation) error {
	root, err := ssz.SigningRoot(att)
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
		if bkt.Get(root[:]) != nil {
			return nil
		}

		shardKey := append(shardIdx, uint64ToBytes(att.Data.Crosslink.Shard)...)
		shardRoots := bkt.Get(shardKey)
		if shardRoots == nil {
			if err := bkt.Put(shardKey, root[:]); err != nil {
				return err
			}
		} else {
			if err := bkt.Put(shardKey, append(shardRoots, root[:]...)); err != nil {
				return err
			}
		}

		parentRootKey := append(parentRootIdx, att.Data.Crosslink.ParentRoot...)
		parentRoots := bkt.Get(parentRootKey)
		if parentRoots == nil {
			if err := bkt.Put(parentRootKey, root[:]); err != nil {
				return err
			}
		} else {
			if err := bkt.Put(parentRootKey, append(parentRoots, root[:]...)); err != nil {
				return err
			}
		}
		return bkt.Put(root[:], enc)
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
		key, err := ssz.SigningRoot(atts[i])
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

			shardKey := append(shardIdx, uint64ToBytes(atts[i].Data.Crosslink.Shard)...)
			shardRoots := bkt.Get(shardKey)
			if shardRoots == nil {
				if err := bkt.Put(shardKey, keys[i]); err != nil {
					return err
				}
			} else {
				if err := bkt.Put(shardKey, append(shardRoots, keys[i]...)); err != nil {
					return err
				}
			}

			parentRootKey := append(parentRootIdx, atts[i].Data.Crosslink.ParentRoot...)
			parentRoots := bkt.Get(parentRootKey)
			if parentRoots == nil {
				if err := bkt.Put(parentRootKey, keys[i]); err != nil {
					return err
				}
			} else {
				if err := bkt.Put(parentRootKey, append(parentRoots, keys[i]...)); err != nil {
					return err
				}
			}
			if err := bkt.Put(keys[i], encodedValues[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func createIndicesFromFilters(f *filters.QueryFilter) [][]byte {
	keys := make([][]byte, 0)
	for k, v := range f.Filters() {
		switch k {
		case filters.Shard:
			idx := append(shardIdx, uint64ToBytes(v.(uint64))...)
			keys = append(keys, idx)
		case filters.ParentRoot:
			parentRoot := v.([]byte)
			idx := append(parentRootIdx, parentRoot...)
			keys = append(keys, idx)
		}
	}
	return keys
}

func valuesForIndices(indices [][]byte, bkt *bolt.Bucket) [][][]byte {
	values := make([][][]byte, 0)
	for _, k := range indices {
		roots := bkt.Get(k)
		splitRoots := make([][]byte, 0)
		for i := 0; i < len(roots); i += 32 {
			splitRoots = append(splitRoots, roots[i:i+32])
		}
		values = append(values, splitRoots)
	}
	return values
}
