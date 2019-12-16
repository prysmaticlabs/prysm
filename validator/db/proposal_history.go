package db

import (
	"math/big"

	"github.com/boltdb/bolt"
	// "github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ValidatorProposalHistory defines the structure for recording a validators historical proposals.
// Using a bitlist and an uint64 to mark the "starting epoch" of the bitlist, we can easily store
// which epochs a validator has proposed a block for.
type ValidatorProposalHistory struct {
	ProposalHistory *big.Int
	EpochAtFirstBit uint64
}

// HasProposedForEpoch returns whether a validators proposal history has been marked for the entered epoch.
// If the request is more in the future than what the history contains, it will return false.
// If the request is from the past, and likely previously pruned it will return true to avoid slashing.
func (history *ValidatorProposalHistory) HasProposedForEpoch(epoch uint64) bool {
	// Previously pruned, but to be safe we should return true.
	if epoch < history.EpochAtFirstBit {
		return true
	}
	// Out of bounds, must be false.
	if epoch > params.BeaconConfig().WeakSubjectivityPeriod+history.EpochAtFirstBit {
		return false
	}
	return history.ProposalHistory.Bit(int(epoch)-int(history.EpochAtFirstBit)) == 1
}

// SetProposedForEpoch
func (history *ValidatorProposalHistory) SetProposedForEpoch(epoch uint64) {
	if epoch < history.EpochAtFirstBit {
		return
	}
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	offsetMultiplier := epoch / wsPeriod
	bitOffset := epoch % wsPeriod
	if offsetMultiplier > 0 {
		offsetMultiplier = offsetMultiplier - 1
	}
	newEpochAtFirstBit := wsPeriod*offsetMultiplier + bitOffset
	//This condition needs to be changed, won't work before 54k.
	if epoch > wsPeriod && newEpochAtFirstBit > history.EpochAtFirstBit {
		history.ProposalHistory = history.ProposalHistory.Rsh(history.ProposalHistory, uint(bitOffset))
		history.EpochAtFirstBit = newEpochAtFirstBit
	}
	history.ProposalHistory = history.ProposalHistory.SetBit(history.ProposalHistory, int(epoch-history.EpochAtFirstBit), 1)
}

func unmarshallProposalHistory(enc []byte) (*ValidatorProposalHistory, error) {
	history := &ValidatorProposalHistory{}
	// err := proto.Unmarshal(enc, history)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "failed to unmarshal encoding")
	// }
	return history, nil
}

// HasProposedAtEpoch accepts an epoch and validator id and returns the corresponding block header array.
// Returns nil if the block header for those values does not exist.
func (db *Store) ProposalHistory(pubKey []byte, epoch uint64) (*ValidatorProposalHistory, error) {
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

// MarkProposedForEpoch accepts a block header and writes it to disk.
// func (db *Store) SetProposalHistory(pubKey []byte, proposalHistory *ValidatorProposalHistory) error {
// 	enc, err := proto.Marshal(proposalHistory)
// 	if err != nil {
// 		return errors.Wrap(err, "failed to encode block")
// 	}

// 	err = db.update(func(tx *bolt.Tx) error {
// 		bucket := tx.Bucket(historicProposalsBucket)
// 		if err := bucket.Put(pubKey, enc); err != nil {
// 			return errors.Wrap(err, "failed to include the block header in the historic block header bucket")
// 		}
// 		return nil
// 	})
// }

// DeleteProposalHistory deletes a validators proposal history using the validators public key.
func (db *Store) DeleteProposalHistory(pubkey []byte) error {
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicProposalsBucket)
		if err := bucket.Delete(pubkey); err != nil {
			return errors.Wrap(err, "failed to delete the block header from historic block header bucket")
		}
		return bucket.Delete(pubkey)
	})
}
