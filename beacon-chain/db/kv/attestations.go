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
	hasFilterSpecified := !reflect.DeepEqual(f, &filters.QueryFilter{})
	err := k.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationsBucket)
		c := bkt.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			meetsFilterCritiera := attestationFilterCriteria(k, f)
			if v != nil && (!hasFilterSpecified || meetsFilterCritiera) {
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
	return k.db.Batch(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(attestationsBucket)
		for _, att := range atts {
			key, err := generateAttestationKey(att)
			if err != nil {
				return err
			}
			enc, err := proto.Marshal(att)
			if err != nil {
				return err
			}
			if err := bucket.Put(key, enc); err != nil {
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

func attestationFilterCriteria(key []byte, f *filters.QueryFilter) bool {
	ok := false
	parentKey := append([]byte("parent-root"), f.ParentRoot...)
	rootKey := append([]byte("root"), f.Root...)
	shardKey := append([]byte("shard"), uint64ToBytes(f.Shard)...)
	startSlot := append([]byte("start-slot"), uint64ToBytes(f.StartSlot)...)
	endSlot := append([]byte("end-slot"), uint64ToBytes(f.EndSlot)...)
	startEpoch := append([]byte("start-epoch"), uint64ToBytes(f.StartEpoch)...)
	endEpoch := append([]byte("end-epoch"), uint64ToBytes(f.EndEpoch)...)
	if len(f.ParentRoot) > 0 && bytes.Contains(key, parentKey) {
		ok = true
	}
	if len(f.Root) > 0 && bytes.Contains(key, rootKey) {
		ok = true
	}
	// TODO: What if we want to query for shard 0?
	if bytes.Contains(key, shardKey) {
		ok = true
	}
	if bytes.Contains(key, startSlot) {
		ok = true
	}
	if bytes.Contains(key, endSlot) {
		ok = true
	}
	if bytes.Contains(key, startEpoch) {
		ok = true
	}
	if bytes.Contains(key, endEpoch) {
		ok = true
	}
	return ok
}
