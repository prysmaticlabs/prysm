package db

import (
	"bytes"

	"github.com/prysmaticlabs/go-ssz"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

type SlashingStatus uint8

const (
	Unknown = iota
	Active
	Included
	Reverted //relevant again
)

func (day SlashingStatus) String() string {
	names := [...]string{
		"Active",
		"Included",
		"Reverted"}

	if day < Active || day > Reverted {
		return "Unknown"
	}
	// return the name of a Weekday
	// constant from the names array
	// above.
	return names[day]
}

func createProposerSlashing(enc []byte) (*ethpb.ProposerSlashing, error) {
	protoSlashing := &ethpb.ProposerSlashing{}

	err := proto.Unmarshal(enc, protoSlashing)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoSlashing, nil
}

// ProposerSlashings accepts a status and returns all slashings with this status.
// returns empty proposer slashing slice if no slashing has been found with this status.
func (db *Store) ProposerSlashings(status SlashingStatus) ([]*ethpb.ProposerSlashing, error) {
	var proposerSlashings []*ethpb.ProposerSlashing
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(proposerSlashingBucket).Cursor()
		prefix := []byte{byte(status)}
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			ps, err := createProposerSlashing(v)
			if err != nil {
				return err
			}
			proposerSlashings = append(proposerSlashings, ps)
		}
		return nil
	})
	return proposerSlashings, err
}

func (db *Store) SlashingsByStatus(status SlashingStatus) ([]*ethpb.ProposerSlashing, error) {
	var proposerSlashings []*ethpb.ProposerSlashing
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(proposerSlashingBucket).Cursor()
		prefix := []byte{byte(status)}
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			ps, err := createProposerSlashing(v)
			if err != nil {
				return err
			}
			proposerSlashings = append(proposerSlashings, ps)
		}
		return nil
	})
	return proposerSlashings, err
}

// SaveProposerSlashing accepts a block header and writes it to disk.
func (db *Store) SaveProposerSlashing(status SlashingStatus, proposerSlashing *ethpb.ProposerSlashing) error {
	found, st, err := db.HasProposerSlashing(proposerSlashing)
	if err != nil {
		return errors.Wrap(err, "failed to check if proposer slashing is already in db")
	}
	if found && st == status {
		return nil
	}
	return db.updateProposerSlashingStatus(proposerSlashing, status)

}

// DeleteProposerSlashing deletes a block header using the epoch and validator id.
func (db *Store) DeleteProposerSlashingWithStatus(status SlashingStatus, proposerSlashing *ethpb.ProposerSlashing) error {
	root, err := ssz.HashTreeRoot(proposerSlashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of proposerSlashing")
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(proposerSlashingBucket)
		k := encodeStatusRoot(status, root)
		if err != nil {
			return errors.Wrap(err, "failed to get key for for proposer slashing.")
		}
		if err := bucket.Delete(k); err != nil {
			return errors.Wrap(err, "failed to delete the block header from historic block header bucket")
		}
		return nil
	})
}

// DeleteValidatorProposerSlashings deletes a block header using the epoch and validator id.
func (db *Store) DeleteProposerSlashing(slashing *ethpb.ProposerSlashing) error {
	root, err := ssz.HashTreeRoot(slashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of proposerSlashing")
	}
	err = db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(proposerSlashingBucket)
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

// HasProposerSlashing returns the slashing key if it is found in db.
func (db *Store) HasProposerSlashing(slashing *ethpb.ProposerSlashing) (bool, SlashingStatus, error) {
	root, err := ssz.HashTreeRoot(slashing)
	var status SlashingStatus
	var found bool
	if err != nil {
		return found, status, errors.Wrap(err, "failed to get hash root of proposerSlashing")
	}
	err = db.view(func(tx *bolt.Tx) error {
		b := tx.Bucket(proposerSlashingBucket)
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

// updateProposerSlashingStatus deletes a proposer slashing and saves it with a new status.
// if old proposer slashing is not found a new entry is being saved as a new entry.
func (db *Store) updateProposerSlashingStatus(slashing *ethpb.ProposerSlashing, status SlashingStatus) error {
	root, err := ssz.HashTreeRoot(slashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of proposerSlashing")
	}
	err = db.update(func(tx *bolt.Tx) error {
		b := tx.Bucket(proposerSlashingBucket)
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
		err = b.Put(encodeStatusRoot(status, root), enc)
		return err
	})
	if err != nil {
		return err
	}
	return err
}
