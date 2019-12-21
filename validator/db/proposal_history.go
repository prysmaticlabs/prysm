package db

import (
	"github.com/boltdb/bolt"
	// "github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/beacon/rpc/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// HasProposedForEpoch returns whether a validators proposal history has been marked for the entered epoch.
// If the request is more in the future than what the history contains, it will return false.
// If the request is from the past, and likely previously pruned it will return true to avoid slashing.
func HasProposedForEpoch(history *ethpb.ValidatorProposalHistory, epoch uint64) bool {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	// Previously pruned, but to be safe we should return true.
	if int(epoch) <= int(history.LastEpochWritten)-int(wsPeriod) {
		return true
	}
	// Accessing future proposals that haven't been marked yet. Needs to return false.
	if epoch > history.LastEpochWritten {
		return false
	}
	return history.ProposalHistory.BitAt(epoch % wsPeriod)
}

// SetProposedForEpoch updates the proposal history to mark the indicated epoch in the bitlist
// and updates the last epoch written if needed.
func SetProposedForEpoch(history *ethpb.ValidatorProposalHistory, epoch uint64) {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod

	if epoch > history.LastEpochWritten {
		// If epoch to mark is ahead of last written epoch, override the old votes and mark the requested epoch.
		for i := history.LastEpochWritten + 1; i <= epoch; i++ {
			if !history.ProposalHistory.BitAt(i % wsPeriod) {
				history.ProposalHistory.SetBitAt(i%wsPeriod, false)
			}
		}
		history.LastEpochWritten = epoch
	}
	history.ProposalHistory.SetBitAt(epoch%wsPeriod, true)
}

func unmarshallProposalHistory(enc []byte) (*ethpb.ValidatorProposalHistory, error) {
	history := &ethpb.ValidatorProposalHistory{}
	err := proto.Unmarshal(enc, history)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return history, nil
}

// ProposalHistory accepts a validator public key and returns the corresponding proposal history.
// Returns nil if there is no proposal history for the validator.
func (db *Store) ProposalHistory(pubKey []byte) (*ethpb.ValidatorProposalHistory, error) {
	var err error
	var proposalHistory *ValidatorProposalHistory
	err = db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		enc := bucket.Get(pubKey)
		proposalHistory, err = unmarshallProposalHistory(enc)
		if err != nil {
			return err
		}
		return nil
	})
	return proposalHistory, err
}

// SaveProposalHistory returns the proposal history for the requested validator public key.
func (db *Store) SaveProposalHistory(pubKey []byte, proposalHistory *ethpb.ValidatorProposalHistory) error {
	enc, err := proto.Marshal(proposalHistory)
	if err != nil {
		return errors.Wrap(err, "failed to encode block")
	}

	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		if err := bucket.Put(pubKey, enc); err != nil {
			return errors.Wrap(err, "failed to include the block header in the historic block header bucket")
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
			return errors.Wrap(err, "failed to delete the block header from historic block header bucket")
		}
		return bucket.Delete(pubkey)
	})
}
