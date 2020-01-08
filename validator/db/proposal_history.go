package db

import (
	"context"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"go.opencensus.io/trace"
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
func (db *Store) ProposalHistory(ctx context.Context, publicKey []byte) (*slashpb.ProposalHistory, error) {
	ctx, span := trace.StartSpan(ctx, "Validator.ProposalHistory")
	defer span.End()

	var err error
	var proposalHistory *slashpb.ProposalHistory
	err = db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		enc := bucket.Get(publicKey)
		if enc == nil {
			return nil
		}
		proposalHistory, err = unmarshalProposalHistory(enc)
		return err
	})
	return proposalHistory, err
}

// SaveProposalHistory returns the proposal history for the requested validator public key.
func (db *Store) SaveProposalHistory(ctx context.Context, pubKey []byte, proposalHistory *slashpb.ProposalHistory) error {
	ctx, span := trace.StartSpan(ctx, "Validator.SaveProposalHistory")
	defer span.End()

	enc, err := proto.Marshal(proposalHistory)
	if err != nil {
		return errors.Wrap(err, "failed to encode proposal history")
	}

	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		return bucket.Put(pubKey, enc)
	})
	return err
}

// DeleteProposalHistory deletes the proposal history for the corresponding validator public key.
func (db *Store) DeleteProposalHistory(ctx context.Context, pubkey []byte) error {
	ctx, span := trace.StartSpan(ctx, "Validator.DeleteProposalHistory")
	defer span.End()

	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		if err := bucket.Delete(pubkey); err != nil {
			return errors.Wrap(err, "failed to delete the proposal history")
		}
		return nil
	})
}
