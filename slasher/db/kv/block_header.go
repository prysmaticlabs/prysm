package kv

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

func unmarshalBlockHeader(ctx context.Context, enc []byte) (*ethpb.SignedBeaconBlockHeader, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.unmarshalBlockHeader")
	defer span.End()
	protoBlockHeader := &ethpb.SignedBeaconBlockHeader{}
	err := proto.Unmarshal(enc, protoBlockHeader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal encoding")
	}
	return protoBlockHeader, nil
}

// BlockHeaders accepts an slot and validator id and returns the corresponding block header array.
// Returns nil if the block header for those values does not exist.
func (s *Store) BlockHeaders(ctx context.Context, slot types.Slot, validatorIndex types.ValidatorIndex) ([]*ethpb.SignedBeaconBlockHeader, error) {
	ctx, span := trace.StartSpan(ctx, "slasherDB.BlockHeaders")
	defer span.End()
	var blockHeaders []*ethpb.SignedBeaconBlockHeader
	err := s.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicBlockHeadersBucket).Cursor()
		prefix := encodeSlotValidatorIndex(slot, validatorIndex)
		for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {
			bh, err := unmarshalBlockHeader(ctx, v)
			if err != nil {
				return err
			}
			blockHeaders = append(blockHeaders, bh)
		}
		return nil
	})
	return blockHeaders, err
}

// HasBlockHeader accepts a slot and validator id and returns true if the block header exists.
func (s *Store) HasBlockHeader(ctx context.Context, slot types.Slot, validatorIndex types.ValidatorIndex) bool {
	ctx, span := trace.StartSpan(ctx, "slasherDB.HasBlockHeader")
	defer span.End()
	prefix := encodeSlotValidatorIndex(slot, validatorIndex)
	var hasBlockHeader bool
	if err := s.view(func(tx *bolt.Tx) error {
		c := tx.Bucket(historicBlockHeadersBucket).Cursor()
		for k, _ := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
			hasBlockHeader = true
			return nil
		}
		hasBlockHeader = false
		return nil
	}); err != nil {
		log.WithError(err).Error("Failed to lookup block header from DB")
	}

	return hasBlockHeader
}

// SaveBlockHeader accepts a block header and writes it to disk.
func (s *Store) SaveBlockHeader(ctx context.Context, blockHeader *ethpb.SignedBeaconBlockHeader) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.SaveBlockHeader")
	defer span.End()
	epoch := helpers.SlotToEpoch(blockHeader.Header.Slot)
	key := encodeSlotValidatorIndexSig(blockHeader.Header.Slot, blockHeader.Header.ProposerIndex, blockHeader.Signature)
	enc, err := proto.Marshal(blockHeader)
	if err != nil {
		return errors.Wrap(err, "failed to encode block")
	}

	err = s.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicBlockHeadersBucket)
		if err := bucket.Put(key, enc); err != nil {
			return errors.Wrap(err, "failed to include block header in the historical bucket")
		}

		return err
	})
	if err != nil {
		return err
	}

	// Prune block header history every 10th epoch.
	if epoch%params.BeaconConfig().PruneSlasherStoragePeriod == 0 {
		return s.PruneBlockHistory(ctx, epoch, params.BeaconConfig().WeakSubjectivityPeriod)
	}
	return nil
}

// DeleteBlockHeader deletes a block header using the slot and validator id.
func (s *Store) DeleteBlockHeader(ctx context.Context, blockHeader *ethpb.SignedBeaconBlockHeader) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.DeleteBlockHeader")
	defer span.End()
	key := encodeSlotValidatorIndexSig(blockHeader.Header.Slot, blockHeader.Header.ProposerIndex, blockHeader.Signature)
	return s.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicBlockHeadersBucket)
		if err := bucket.Delete(key); err != nil {
			return errors.Wrap(err, "failed to delete the block header from historical bucket")
		}
		return bucket.Delete(key)
	})
}

// PruneBlockHistory leaves only records younger then history size.
func (s *Store) PruneBlockHistory(ctx context.Context, currentEpoch, pruningEpochAge types.Epoch) error {
	ctx, span := trace.StartSpan(ctx, "slasherDB.pruneBlockHistory")
	defer span.End()
	pruneTill := int64(currentEpoch) - int64(pruningEpochAge)
	if pruneTill <= 0 {
		return nil
	}
	pruneTillSlot := uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(pruneTill)))
	return s.update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(historicBlockHeadersBucket)
		c := tx.Bucket(historicBlockHeadersBucket).Cursor()
		for k, _ := c.First(); k != nil && bytesutil.FromBytes8(k[:8]) <= pruneTillSlot; k, _ = c.Next() {
			if err := bucket.Delete(k); err != nil {
				return errors.Wrap(err, "failed to delete the block header from historical bucket")
			}
		}
		return nil
	})
}
