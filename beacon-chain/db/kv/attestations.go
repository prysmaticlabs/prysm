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
		c := bkt.Cursor()
		for k, v := c.Seek(attRoot[:]); k != nil && bytes.Contains(k, attRoot[:]); k, v = c.Next() {
			if v != nil {
				return proto.Unmarshal(v, att)
			}
		}
		return nil
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
		c := bkt.Cursor()
		for k, v := c.Seek(attRoot[:]); k != nil && bytes.Contains(k, attRoot[:]); k, v = c.Next() {
			if v != nil {
				exists = true
				return nil
			}
		}
		return nil
	})
	return exists
}

// DeleteAttestation by root.
func (k *Store) DeleteAttestation(ctx context.Context, attRoot [32]byte) error {
	return k.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		c := bkt.Cursor()
		for k, v := c.Seek(attRoot[:]); k != nil && bytes.Contains(k, attRoot[:]); k, v = c.Next() {
			if v != nil {
				return bkt.Delete(k)
			}
		}
		return nil
	})
}

// SaveAttestation to the db.
func (k *Store) SaveAttestation(ctx context.Context, att *ethpb.Attestation) error {
	key, err := generateAttestationKey(att)
	if err != nil {
		return err
	}
	enc, err := proto.Marshal(att)
	if err != nil {
		return err
	}
	return k.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(attestationsBucket)
		return bucket.Put(key, enc)
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
		key, err := generateAttestationKey(atts[i])
		if err != nil {
			return err
		}
		encodedValues[i] = enc
		keys[i] = key
	}
	return k.db.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(attestationsBucket)
		for i := 0; i < len(atts); i++ {
			if err := bucket.Put(keys[i], encodedValues[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

func generateAttestationKey(att *ethpb.Attestation) ([]byte, error) {
	buf := make([]byte, 0)
	buf = append(buf, []byte("shard")...)
	buf = append(buf, uint64ToBytes(att.Data.Crosslink.Shard)...)

	buf = append(buf, []byte("parent-root")...)
	buf = append(buf, att.Data.Crosslink.ParentRoot...)

	buf = append(buf, []byte("start-epoch")...)
	buf = append(buf, uint64ToBytes(att.Data.Crosslink.StartEpoch)...)

	buf = append(buf, []byte("end-epoch")...)
	buf = append(buf, uint64ToBytes(att.Data.Crosslink.EndEpoch)...)

	buf = append(buf, []byte("root")...)
	attRoot, err := ssz.HashTreeRoot(att)
	if err != nil {
		return nil, err
	}
	buf = append(buf, attRoot[:]...)
	return buf, nil
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
