package db

import (
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
)

func unmarshalProposalHistory(enc []byte) (*slashpb.ProposalHistory, error) {
	history := &slashpb.ProposalHistory{}
	err := proto.Unmarshal(enc, history)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return history, nil
}

// ProposalHistory accepts a validator public key and returns the corresponding proposal history.
// Returns nil if there is no proposal history for the validator.
func (db *Store) ProposalHistory(pubKey []byte) (*slashpb.ProposalHistory, error) {
	var err error
	var proposalHistory *slashpb.ProposalHistory
	err = db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		enc := bucket.Get(pubKey)
		proposalHistory, err = unmarshalProposalHistory(enc)
		if err != nil {
			return err
		}
		return nil
	})
	return proposalHistory, err
}

// SaveProposalHistory returns the proposal history for the requested validator public key.
func (db *Store) SaveProposalHistory(pubKey []byte, proposalHistory *slashpb.ProposalHistory) error {
	enc, err := proto.Marshal(proposalHistory)
	if err != nil {
		return errors.Wrap(err, "failed to encode proposal history")
	}

	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		if err := bucket.Put(pubKey, enc); err != nil {
			return errors.Wrap(err, "failed to save the proposal history")
		}
		return nil
	})
	return err
}

// DeleteProposalHistory deletes the proposal history for the corresponding validator public key.
func (db *Store) DeleteProposalHistory(pubkey []byte) error {
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		if err := bucket.Delete(pubkey); err != nil {
			return errors.Wrap(err, "failed to delete the proposal history")
		}
		return nil
	})
}
