package db

import (
	"bytes"

	"github.com/prysmaticlabs/prysm/shared/hashutil"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

func createAttesterSlashing(enc []byte) (*ethpb.AttesterSlashing, error) {
	protoSlashing := &ethpb.AttesterSlashing{}
	err := proto.Unmarshal(enc, protoSlashing)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoSlashing, nil
}

func toAttesterSlashings(encoded [][]byte) ([]*ethpb.AttesterSlashing, error) {
	attesterSlashings := make([]*ethpb.AttesterSlashing, len(encoded))
	for i, enc := range encoded {
		ps, err := createAttesterSlashing(enc)
		if err != nil {
			return nil, err
		}
		attesterSlashings[i] = ps
	}
	return attesterSlashings, nil
}

// AttesterSlashings accepts a status and returns all slashings with this status.
// returns empty []*ethpb.AttesterSlashing if no slashing has been found with this status.
func (db *Store) AttesterSlashings(status SlashingStatus) ([]*ethpb.AttesterSlashing, error) {
	encoded := make([][]byte, 0)
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(slashingBucket).Cursor()
		prefix := encodeType(SlashingType(Attestation))
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			if v[0] == byte(status) {
				encoded = append(encoded, v[1:])
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toAttesterSlashings(encoded)
}

// DeleteAttesterSlashing deletes a slashing proof from db.
func (db *Store) DeleteAttesterSlashing(attesterSlashing *ethpb.AttesterSlashing) error {
	root, err := hashutil.HashProto(attesterSlashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of attesterSlashing")
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(slashingBucket)
		k := encodeTypeRoot(SlashingType(Attestation), root)
		if err != nil {
			return errors.Wrap(err, "failed to get key for for attester slashing.")
		}
		if err := bucket.Delete(k); err != nil {
			return errors.Wrap(err, "failed to delete the slashing proof from slashing bucket")
		}
		return nil
	})
}

// HasAttesterSlashing returns the slashing key if it is found in db.
func (db *Store) HasAttesterSlashing(slashing *ethpb.AttesterSlashing) (bool, SlashingStatus, error) {
	root, err := hashutil.HashProto(slashing)
	var status SlashingStatus
	var found bool
	key := encodeTypeRoot(SlashingType(Attestation), root)
	if err != nil {
		return found, status, errors.Wrap(err, "failed to get hash root of attesterSlashing")
	}
	err = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(slashingBucket)
		enc := b.Get(key)
		if enc != nil {
			found = true
			status = SlashingStatus(enc[0])
		}
		return nil
	})
	return found, status, err
}

// SaveAttesterSlashing accepts a slashing proof and its status and writes it to disk.
func (db *Store) SaveAttesterSlashing(status SlashingStatus, slashing *ethpb.AttesterSlashing) error {
	enc, err := proto.Marshal(slashing)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}
	root := hashutil.Hash(enc)
	key := encodeTypeRoot(SlashingType(Attestation), root)
	err = db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(slashingBucket)
		e := b.Put(key, append([]byte{byte(status)}, enc...))
		if e != nil {
			return nil
		}
		return err
	})
	if err != nil {
		return err
	}
	return err
}
