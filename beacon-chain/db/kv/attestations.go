package kv

import (
	"bytes"
	"context"
	"encoding/binary"

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
// TODO(#3164): Implement.
func (k *Store) Attestations(ctx context.Context, f *filters.QueryFilter) ([]*ethpb.Attestation, error) {
	return nil, nil
}

// HasAttestation checks if an attestation by root exists in the db.
func (k *Store) HasAttestation(ctx context.Context, attRoot [32]byte) bool {
	exists := false
	// #nosec G104, similar to HasBlock, HasAttestation... etc
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
		bucket := tx.Bucket(attestationsBucket)
		return bucket.Delete(attRoot[:])
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
			attRoot, err := ssz.HashTreeRoot(att)
			if err != nil {
				return err
			}
			enc, err := proto.Marshal(att)
			if err != nil {
				return err
			}
			if err := bucket.Put(attRoot[:], enc); err != nil {
				return err
			}
		}
		return nil
	})
}

func generateAttestationKey(att *ethpb.Attestation) ([]byte, error) {
	buf := make([]byte, 0)
	buf = append(buf, []byte("shard")...)
	shardEncoded := make([]byte, 8)
	binary.LittleEndian.PutUint64(shardEncoded, att.Data.Crosslink.Shard)
	buf = append(buf, shardEncoded...)

	buf = append(buf, []byte("parent-root")...)
	buf = append(buf, att.Data.Crosslink.ParentRoot...)
	buf = append(buf, []byte("start-epoch")...)
	startEpochEncoded := make([]byte, 8)
	binary.LittleEndian.PutUint64(startEpochEncoded, att.Data.Crosslink.StartEpoch)
	buf = append(buf, startEpochEncoded...)
	buf = append(buf, []byte("end-epoch")...)
	endEpochEncoded := make([]byte, 8)
	binary.LittleEndian.PutUint64(endEpochEncoded, att.Data.Crosslink.EndEpoch)
	buf = append(buf, endEpochEncoded...)
	buf = append(buf, []byte("root")...)
	attRoot, err := ssz.HashTreeRoot(att)
	if err != nil {
		return nil, err
	}
	buf = append(buf, attRoot[:]...)
	return buf, nil
}
