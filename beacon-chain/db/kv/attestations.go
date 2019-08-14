package kv

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/filters"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
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
		keys := createIndicesFromFilters(f)
		rootSets := [][][]byte{}
		for _, k := range keys {
			roots := bkt.Get(k)
			splitRoots := [][]byte{}
			for i := 0; i < len(roots); i += 32 {
				splitRoots = append(splitRoots, roots[i:i+32])
			}
			rootSets = append(rootSets, splitRoots)
		}
		intersectedRoots := findRootsIntersection(rootSets)
		for i := 0; i < len(intersectedRoots); i++ {
			encoded := bkt.Get(intersectedRoots[i])
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

func findRootsIntersection(rootSets [][][]byte) [][]byte {
	if len(rootSets) == 0 {
		return [][]byte{}
	}
	if len(rootSets) == 1 {
		return rootSets[0]
	}
	intersected := intersection(rootSets[0], rootSets[1])
	for i := 2; i < len(rootSets); i++ {
		intersected = intersection(intersected, rootSets[i])
	}
	return intersected
}

func intersection(s1, s2 [][]byte) (inter [][]byte) {
	hash := make(map[string]bool)
	for _, e := range s1 {
		hash[string(e)] = true
	}
	for _, e := range s2 {
		// If elements present in the hashmap then append intersection list.
		if hash[string(e)] {
			inter = append(inter, e)
		}
	}
	//Remove dups from slice.
	inter = removeDups(inter)
	return
}

//Remove dups from slice.
func removeDups(elements [][]byte) (nodups [][]byte) {
	encountered := make(map[string]bool)
	for _, element := range elements {
		if !encountered[string(element)] {
			nodups = append(nodups, element)
			encountered[string(element)] = true
		}
	}
	return
}
