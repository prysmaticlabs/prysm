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

// AttesterSlashings accepts a status and returns all slashings with this status.
// returns empty []*ethpb.AttesterSlashing if no slashing has been found with this status.
func (db *Store) AttesterSlashings(status SlashingStatus) ([]*ethpb.AttesterSlashing, error) {
	var attesterSlashings []*ethpb.AttesterSlashing
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(slashingBucket).Cursor()
		prefix := encodeStatusType(status, SlashingType(Attestation))
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			ps, err := createAttesterSlashing(v)
			if err != nil {
				return err
			}
			attesterSlashings = append(attesterSlashings, ps)
		}
		return nil
	})
	return attesterSlashings, err
}

func (db *Store) AttestingSlashingsByStatus(status SlashingStatus) ([]*ethpb.AttesterSlashing, error) {
	var attesterSlashings []*ethpb.AttesterSlashing
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(slashingBucket).Cursor()
		prefix := encodeStatusType(status, SlashingType(Attestation))
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			ps, err := createAttesterSlashing(v)
			if err != nil {
				return err
			}
			attesterSlashings = append(attesterSlashings, ps)
		}
		return nil
	})
	return attesterSlashings, err
}

// SaveAttesterSlashing accepts a block header and writes it to disk.
func (db *Store) SaveAttesterSlashing(status SlashingStatus, attesterSlashing *ethpb.AttesterSlashing) error {
	found, st, err := db.HasAttesterSlashing(attesterSlashing)
	if err != nil {
		return errors.Wrap(err, "failed to check if attester slashing is already in db")
	}
	if found && st == status {
		return nil
	}
	return db.updateAttesterSlashingStatus(attesterSlashing, status)

}

// DeleteAttesterSlashing deletes a block header using the epoch and validator id.
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
			return errors.Wrap(err, "failed to delete the block header from historic block header bucket")
		}
		return nil
	})
}

// DeleteValidatorAttesterSlashings deletes a block header using the epoch and validator id.
func (db *Store) DeleteAttesterSlashing(slashing *ethpb.AttesterSlashing) error {
	root, err := ssz.HashTreeRoot(slashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of attesterSlashing")
	}
	err = db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(slashingBucket)
		b.ForEach(func(k, v []byte) error {
			if bytes.HasSuffix(k, root[:]) {
				b.Delete(k)
			}
			return nil
		})
		return nil
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

// updateAttesterSlashingStatus deletes a attester slashing and saves it with a new status.
// if old attester slashing is not found a new entry is being saved as a new entry.
func (db *Store) updateAttesterSlashingStatus(slashing *ethpb.AttesterSlashing, status SlashingStatus) error {
	root, err := ssz.HashTreeRoot(slashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of attesterSlashing")
	}
	err = db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(slashingBucket)
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
		enc, err := proto.Marshal(slashing)
		if err != nil {
			return errors.Wrap(err, "failed to marshal")
		}
		err = b.Put(encodeStatusTypeRoot(status, SlashingType(Attestation), root), enc)
		return err
	})
	if err != nil {
		return err
	}
	return err
}
