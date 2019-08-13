package kv

import (
	"bytes"
	"context"
	"reflect"

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
	hasFilterSpecified := !reflect.DeepEqual(f, &filters.QueryFilter{}) && f != nil
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		c := bkt.Cursor()
		// TODO: Use indices here.
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if v != nil && (!hasFilterSpecified || ensureAttestationFilterCriteria(k, f)) {
				att := &ethpb.Attestation{}
				if err := proto.Unmarshal(v, att); err != nil {
					return err
				}
				atts = append(atts, att)
			}
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

		shardKey := append(attestationShardIdx, uint64ToBytes(att.Data.Crosslink.Shard)...)
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

			shardKey := append(attestationShardIdx, uint64ToBytes(atts[i].Data.Crosslink.Shard)...)
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
			if err := bkt.Put(keys[i], encodedValues[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// ensureAttestationFilterCriteria uses a set of specified filters
// to ensure the byte key used for db lookups contains the correct values
// requested by the filter. For example, if a key looks like:
// root-0x23923-parent-root-0x49349-start-epoch-3-end-epoch-4-shard-5
// and our filter criteria wants the key to contain shard 5 and
// start epoch 5, the key will NOT meet all the filter criteria and this
// function will return false.
func ensureAttestationFilterCriteria(key []byte, f *filters.QueryFilter) bool {
	numCriteriaMet := 0
	for k, v := range f.Filters() {
		switch k {
		case filters.Root:
			root := v.([]byte)
			if bytes.Contains(key, append([]byte("root"), root[:]...)) {
				numCriteriaMet++
			}
		case filters.ParentRoot:
			root := v.([]byte)
			if bytes.Contains(key, append([]byte("parent-root"), root[:]...)) {
				numCriteriaMet++
			}
		case filters.StartEpoch:
			if bytes.Contains(key, append([]byte("start-epoch"), uint64ToBytes(v.(uint64))...)) {
				numCriteriaMet++
			}
		case filters.EndEpoch:
			if bytes.Contains(key, append([]byte("end-epoch"), uint64ToBytes(v.(uint64))...)) {
				numCriteriaMet++
			}
		case filters.Shard:
			if bytes.Contains(key, append([]byte("shard"), uint64ToBytes(v.(uint64))...)) {
				numCriteriaMet++
			}
		}
	}
	return numCriteriaMet == len(f.Filters())
}
