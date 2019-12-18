package db

import (
	"bytes"

	"github.com/prysmaticlabs/go-ssz"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
)

// SlashingStatus enum like structure.
type SlashingStatus uint8

const (
	// Unknown default status in case it is not set
	Unknown = iota
	// Active slashing proof hasn't been included yet.
	Active
	// Included slashing proof that has been included in a block.
	Included
	// Reverted slashing proof that has been reverted and therefore is relevant again.
	Reverted //relevant again
)

func (status SlashingStatus) String() string {
	names := [...]string{
		"Unknown",
		"Active",
		"Included",
		"Reverted"}

	if status < Active || status > Reverted {
		return "Unknown"
	}
	// return the name of a SlashingStatus
	// constant from the names array
	// above.
	return names[status]
}

// SlashingType enum like type of slashing proof.
type SlashingType uint8

const (
	// Proposal enun value.
	Proposal = iota
	// Attestation enum value.
	Attestation
)

func (status SlashingType) String() string {
	names := [...]string{
		"Proposal",
		"Attestation"}

	if status < Active || status > Reverted {
		return "Unknown"
	}
	// return the name of a SlashingType
	// constant from the names array
	// above.
	return names[status]
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
		c := tx.Bucket(slashingBucket).Cursor()
		prefix := encodeStatusType(status, SlashingType(Proposal))
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

// ProposalSlashingsByStatus returns all the proposal slashing proofs with a certain status.
func (db *Store) ProposalSlashingsByStatus(status SlashingStatus) ([]*ethpb.ProposerSlashing, error) {
	var proposerSlashings []*ethpb.ProposerSlashing
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(slashingBucket).Cursor()
		prefix := encodeStatusType(status, SlashingType(Proposal))
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

// SaveProposerSlashing accepts a proposer slashing and its status header and writes it to disk.
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

// DeleteProposerSlashingWithStatus deletes a proposal slashing proof given its status.
func (db *Store) DeleteProposerSlashingWithStatus(status SlashingStatus, proposerSlashing *ethpb.ProposerSlashing) error {
	root, err := ssz.HashTreeRoot(proposerSlashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of proposerSlashing")
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(slashingBucket)
		k := encodeStatusTypeRoot(status, SlashingType(Proposal), root)
		if err != nil {
			return errors.Wrap(err, "failed to get key for for proposer slashing.")
		}
		if err := bucket.Delete(k); err != nil {
			return errors.Wrap(err, "failed to delete the slashing proof header from slashing bucket")
		}
		return nil
	})
}

// DeleteProposerSlashing deletes a proposer slashing proof.
func (db *Store) DeleteProposerSlashing(slashing *ethpb.ProposerSlashing) error {
	root, err := ssz.HashTreeRoot(slashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of proposerSlashing")
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

// HasProposerSlashing returns the slashing key if it is found in db.
func (db *Store) HasProposerSlashing(slashing *ethpb.ProposerSlashing) (bool, SlashingStatus, error) {
	root, err := ssz.HashTreeRoot(slashing)
	var status SlashingStatus
	var found bool
	if err != nil {
		return found, status, errors.Wrap(err, "failed to get hash root of proposerSlashing")
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

// updateProposerSlashingStatus deletes a proposer slashing and saves it with a new status.
// if old proposer slashing is not found a new entry is being saved as a new entry.
func (db *Store) updateProposerSlashingStatus(slashing *ethpb.ProposerSlashing, status SlashingStatus) error {
	root, err := ssz.HashTreeRoot(slashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of proposerSlashing")
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
		err = b.Put(encodeStatusTypeRoot(status, SlashingType(Proposal), root), enc)
		return err
	})
	if err != nil {
		return err
	}
	return err
}
