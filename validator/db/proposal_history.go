package db

import (
	"bytes"
	"github.com/prysmaticlabs/go-bitfield"
	"math"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// ValidatorProposalHistory defines the structure for recording a validators historical proposals.
// Using a bitlist and an uint64 to mark the "starting epoch" of the bitlist, we can easily store
// which epochs a validator has proposed a block for.
type ValidatorProposalHistory struct {
	ProposalHistory []byte
	EpochAtFirstBit uint64
}

func (history *ValidatorProposalHistory) HasProposedForEpoch(epoch uint64) bool {
	hasProposed := history.ProposalHistory.BitAt(epoch - history.EpochAtFirstBit)
	return hasProposed
}

func (history *ValidatorProposalHistory) SetProposedForEpoch(epoch uint64) bool {
	wsPeriod := params.BeaconConfig().WeakSubjectivityPeriod
	offsetMultiplier := epoch / wsPeriod
	bitOffset := epoch % wsPeriod
	if offsetMultiplier > 0 {
		offsetMultiplier = offsetMultiplier - 1
	}
	newEpochAtFirstBit := wsPeriod*offsetMultiplier + bitOffset
	if offsetMultiplier > 1 && newEpochAtFirstBit > history.EpochAtFirstBit {
		history.ProposalHistory << bitOffset
		history.EpochAtFirstBit = newEpochAtFirstBit
	}
	history.ProposalHistory.SetBitAt(epoch-history.EpochAtFirstBit, true)
}

func unmarshallProposalHistory(enc []byte) (*ValidatorProposalHistory, error) {
	bitlist := &ValidatorProposalHistory{}
	err := proto.Unmarshal(enc, bitlist)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return bitlist, nil
}

// HasProposedAtEpoch accepts an epoch and validator id and returns the corresponding block header array.
// Returns nil if the block header for those values does not exist.
func (db *Store) ProposalHistory(pubKey []byte, epoch uint64) (*ValidatorProposalHistory, error) {
	var proposalHistory *ValidatorProposalHistory
	err := db.view(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicBlockHeadersBucket)
		enc := bucket.Get(pubKey)
		proposalHistory, err := unmarshallProposalHistory(enc)
		if err != nil {
			return err
		}
		return nil
	})
	return proposalHistory, err
}

// MarkProposedForEpoch accepts a block header and writes it to disk.
func (db *Store) SetProposalHistory(pubKey []byte, proposalHistory *ValidatorProposalHistory) error {
	enc, err := proto.Marshal(proposalHistory)
	if err != nil {
		return errors.Wrap(err, "failed to encode block")
	}

	err = db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicBlockHeadersBucket)
		if err := bucket.Put(pubKey, enc); err != nil {
			return errors.Wrap(err, "failed to include the block header in the historic block header bucket")
		}
		return nil
	})
}

// DeleteProposalHistory deletes a validators proposal history using the validators public key.
func (db *Store) DeleteProposalHistory(pubkey []byte) error {
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicBlockHeadersBucket)
		if err := bucket.Delete(key); err != nil {
			return errors.Wrap(err, "failed to delete the block header from historic block header bucket")
		}
		return bucket.Delete(key)
	})
}

// PruneHistory leaves only records younger then history size.
func (db *Store) PruneHistory(currentEpoch uint64, historySize uint64) error {
	pruneTill := int64(currentEpoch) - int64(historySize)
	if pruneTill <= 0 {
		return nil
	}
	return db.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicBlockHeadersBucket)
		c := tx.Bucket(historicBlockHeadersBucket).Cursor()
		for k, _ := c.First(); k != nil && bytesutil.FromBytes8(k[:8]) <= uint64(pruneTill); k, _ = c.Next() {
			if err := bucket.Delete(k); err != nil {
				return errors.Wrap(err, "failed to delete the block header from historic block header bucket")
			}
		}
		return nil
	})
}
