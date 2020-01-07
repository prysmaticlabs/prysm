package db

import (
	"bytes"

	"github.com/prysmaticlabs/go-ssz"

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
		prefix := encodeStatusType(status, SlashingType(Attestation))
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			encoded = append(encoded, v)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return toAttesterSlashings(encoded)
}

// DeleteAttesterSlashingWithStatus deletes a slashing proof using the slashing status and slashing proof.
func (db *Store) DeleteAttesterSlashingWithStatus(status SlashingStatus, attesterSlashing *ethpb.AttesterSlashing) error {
	root, err := ssz.HashTreeRoot(attesterSlashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of attesterSlashing")
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(slashingBucket)
		k := encodeStatusTypeRoot(status, SlashingType(Attestation), root)
		if err != nil {
			return errors.Wrap(err, "failed to get key for for attester slashing.")
		}
		if err := bucket.Delete(k); err != nil {
			return errors.Wrap(err, "failed to delete the slashing proof from slashing bucket")
		}
		return nil
	})
}

// DeleteAttesterSlashing deletes attester slashing proof.
func (db *Store) DeleteAttesterSlashing(slashing *ethpb.AttesterSlashing) error {
	root, err := ssz.HashTreeRoot(slashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of attester slashing")
	}
	err = db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(slashingBucket)
		err = b.ForEach(func(k, v []byte) error {
			if bytes.HasSuffix(k, root[:]) {
				err = b.Delete(k)
				if err != nil {
					return err
				}
			}
			return nil
		})
		return err
	})
	return err
}

// HasAttesterSlashing returns the slashing key if it is found in db.
func (db *Store) HasAttesterSlashing(slashing *ethpb.AttesterSlashing) (bool, SlashingStatus, error) {
	root, err := ssz.HashTreeRoot(slashing)
	var status SlashingStatus
	var found bool
	if err != nil {
		return found, status, errors.Wrap(err, "failed to get hash root of attesterSlashing")
	}
	err = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(slashingBucket)
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if bytes.HasSuffix(k, root[:]) {
				found = true
				status = SlashingStatus(k[0])
				return nil
			}
		}
		return nil
	})
	return found, status, err
}

// SaveAttesterSlashing accepts a slashing proof and its status and writes it to disk.
func (db *Store) SaveAttesterSlashing(status SlashingStatus, slashing *ethpb.AttesterSlashing) error {
	root, err := ssz.HashTreeRoot(slashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of attesterSlashing")
	}
	enc, err := proto.Marshal(slashing)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}
	key := encodeStatusTypeRoot(status, SlashingType(Attestation), root)
	err = db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(slashingBucket)
		e := b.Get(key)
		if e != nil {
			return nil
		}
		var keysToDelete [][]byte
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if bytes.HasSuffix(k, root[:]) {
				keysToDelete = append(keysToDelete, k)
			}
		}
		for _, k := range keysToDelete {
			err = b.Delete(k)
			if err != nil {
				return err
			}

		}
		err = b.Put(key, enc)
		return err
	})
	if err != nil {
		return err
	}
	return err
}
