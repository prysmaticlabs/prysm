package db

import (
	"bytes"

	"github.com/prysmaticlabs/prysm/shared/hashutil"

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
	// Proposal enum value.
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

func toProposerSlashings(encoded [][]byte) ([]*ethpb.ProposerSlashing, error) {
	proposerSlashings := make([]*ethpb.ProposerSlashing, len(encoded))
	for i, enc := range encoded {
		ps, err := createProposerSlashing(enc)
		if err != nil {
			return nil, err
		}
		proposerSlashings[i] = ps
	}
	return proposerSlashings, nil
}

// ProposalSlashingsByStatus returns all the proposal slashing proofs with a certain status.
func (db *Store) ProposalSlashingsByStatus(status SlashingStatus) ([]*ethpb.ProposerSlashing, error) {
	encoded := make([][]byte, 0)
	err := db.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(slashingBucket).Cursor()
		prefix := encodeType(SlashingType(Proposal))
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
	return toProposerSlashings(encoded)
}

// DeleteProposerSlashing deletes a proposer slashing proof.
func (db *Store) DeleteProposerSlashing(slashing *ethpb.ProposerSlashing) error {
	root, err := hashutil.HashProto(slashing)
	if err != nil {
		return errors.Wrap(err, "failed to get hash root of proposerSlashing")
	}
	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(slashingBucket)
		k := encodeTypeRoot(SlashingType(Proposal), root)
		if err != nil {
			return errors.Wrap(err, "failed to get key for for attester slashing.")
		}
		if err := bucket.Delete(k); err != nil {
			return errors.Wrap(err, "failed to delete the slashing proof from slashing bucket")
		}
		return nil
	})
	return err
}

// HasProposerSlashing returns the slashing key if it is found in db.
func (db *Store) HasProposerSlashing(slashing *ethpb.ProposerSlashing) (bool, SlashingStatus, error) {
	root, err := hashutil.HashProto(slashing)
	key := encodeTypeRoot(SlashingType(Proposal), root)
	var status SlashingStatus
	var found bool
	if err != nil {
		return found, status, errors.Wrap(err, "failed to get hash root of proposerSlashing")
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

// SaveProposerSlashing accepts a proposer slashing and its status header and writes it to disk.
func (db *Store) SaveProposerSlashing(status SlashingStatus, slashing *ethpb.ProposerSlashing) error {
	enc, err := proto.Marshal(slashing)
	if err != nil {
		return errors.Wrap(err, "failed to marshal")
	}
	root := hashutil.Hash(enc)
	key := encodeTypeRoot(SlashingType(Proposal), root)
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
