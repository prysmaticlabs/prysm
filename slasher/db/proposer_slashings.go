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
	Relevant = iota
	Included
	Reverted //relevant again
)

func (day SlashingStatus) String() string {
	names := [...]string{
		"Relevant",
		"Included",
		"Reverted"}

	if day < Relevant || day > Reverted {
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

// BlockHeader accepts an epoch and validator id and returns the corresponding block header array.
// Returns nil if the block header for those values does not exist.
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

// BlockHeader accepts an epoch and validator id and returns the corresponding block header array.
// Returns nil if the block header for those values does not exist.
func (db *Store) ValidatorProposerSlashings(status SlashingStatus,validatorID uint64) ([]*ethpb.ProposerSlashing, error) {
	var proposerSlashings []*ethpb.ProposerSlashing
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(proposerSlashingBucket).Cursor()
		prefix := encodeStatusValidatorID(status,validatorID)
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

// BlockHeader accepts an epoch and validator id and returns the corresponding block header array.
// Returns nil if the block header for those values does not exist.
func (db *Store) ProposerSlashingsKey(status SlashingStatus,proposerSlashing *ethpb.ProposerSlashing) ([]byte, error) {
	r, err := ssz.HashTreeRoot(proposerSlashing)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get hash root of proposerSlashing")
	}
	var key []byte
	err = db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(proposerSlashingBucket).Cursor()
		prefix := []byte{byte(status)}
		for k, _ := c.Seek(prefix; k != nil && bytes.HasPrefix(k, prefix; k, _ = c.Next() {
			if bytes.HasSuffix(k,r[:]){
				key=k
				return nil
			}
		}
		return nil
	})
	return key, err
}

// SaveProposerSlashing accepts a block header and writes it to disk.
func (db *Store) SaveProposerSlashing(status SlashingStatus, validatorID uint64, proposerSlashing *ethpb.ProposerSlashing) error {
	r, err := ssz.HashTreeRoot(proposerSlashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of proposerSlashing")
	}
	key := encodeStatusValidatorIDRoot(status, validatorID, r)
	enc, err := proto.Marshal(proposerSlashing)
	if err != nil {
		return errors.Wrap(err, "failed to encode slashing")
	}
	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(proposerSlashingBucket)
		if err := bucket.Put(key, enc); err != nil {
			return errors.Wrap(err, "failed to include the proposer slashing in the proposer slashing bucket")
		}
		return err
	})
	return err
}

// DeleteProposerSlashing deletes a block header using the epoch and validator id.
func (db *Store) DeleteProposerSlashing(status SlashingStatus,proposerSlashing *ethpb.ProposerSlashing) error {

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(proposerSlashingBucket)
		k,err:= db.ProposerSlashingsKey(status,proposerSlashing)
		if err!=nil{
			return errors.Wrap(err,"failed to get key for for proposer slashing.")
		}
		if err := bucket.Delete(k); err != nil {
			return errors.Wrap(err, "failed to delete the block header from historic block header bucket")
		}
		return bucket.Delete(k)
	})
}

// DeleteValidatorProposerSlashings deletes a block header using the epoch and validator id.
func (db *Store) DeleteValidatorProposerSlashings(status SlashingStatus,validatorID uint64) ([]*ethpb.ProposerSlashing, error) {
	var proposerSlashings []*ethpb.ProposerSlashing
	err := db.update(func(tx *bolt.Tx) error {
		c := tx.Bucket(proposerSlashingBucket).Cursor()
		b:=tx.Bucket(proposerSlashingBucket)
		prefix := encodeStatusValidatorID(status,validatorID)
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			errors := b.Delete(k)
		}
		return tx.Commit()
	})
	return proposerSlashings, err
}





