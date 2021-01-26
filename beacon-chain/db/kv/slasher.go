package kv

import (
	"context"

	ssz "github.com/ferranbt/fastssz"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

func (s *Store) LatestEpochWrittenForValidator(ctx context.Context, validatorIdx types.ValidatorIdx) (types.Epoch, bool, error) {
	ctx, span := trace.StartSpan(ctx, "AdvancedSlasherDB.LatestEpochWrittenForValidator")
	defer span.End()
	var epoch types.Epoch
	var exists bool
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(epochByValidatorBucket)
		enc := ssz.MarshalUint64(make([]byte, 0), uint64(validatorIdx))
		epochBytes := bkt.Get(enc)
		if epochBytes == nil {
			return nil
		}
		epoch = types.Epoch(ssz.UnmarshallUint64(epochBytes))
		exists = true
		return nil
	})
	return epoch, exists, err
}

func (s *Store) AttestationRecordForValidator(
	ctx context.Context, validatorIdx types.ValidatorIdx, targetEpoch types.Epoch,
) (*types.AttestationRecord, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.AttestationRecordForValidator")
	defer span.End()
	var record *types.AttestationRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationRecordsBucket)
		encIdx := ssz.MarshalUint64(make([]byte, 0), uint64(validatorIdx))
		encEpoch := ssz.MarshalUint64(make([]byte, 0), uint64(targetEpoch))
		key := append(encIdx, encEpoch...)
		value := bkt.Get(key)
		if value == nil {
			return nil
		}
		record = &types.AttestationRecord{
			Source: ssz.UnmarshallUint64(value[0:8]),
			Target: ssz.UnmarshallUint64(value[8:]),
		}
		return nil
	})
	return record, err

}

func (s *Store) LoadChunk(ctx context.Context, kind types.ChunkKind, diskKey uint64) ([]types.Span, bool, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LoadChunk")
	defer span.End()
	var chunk []types.Span
	var exists bool
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(slasherChunksBucket)
		keyBytes := ssz.MarshalUint64(make([]byte, 0), diskKey)
		keyBytes = append(ssz.MarshalUint8(make([]byte, 0), uint8(kind)), keyBytes...)
		chunkBytes := bkt.Get(keyBytes)
		if chunkBytes == nil {
			return nil
		}
		chunk = make([]types.Span, 0)
		for i := 0; i < len(chunkBytes); i += 2 {
			distance := ssz.UnmarshallUint16(chunkBytes[i : i+2])
			chunk = append(chunk, types.Span(distance))
		}
		exists = true
		return nil
	})
	return chunk, exists, err
}

func (s *Store) UpdateLatestEpochWrittenForValidators(
	ctx context.Context, validatorIndices []types.ValidatorIdx, epoch types.Epoch,
) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.UpdateLatestEpochWrittenForValidator")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(epochByValidatorBucket)
		for _, valIdx := range validatorIndices {
			key := ssz.MarshalUint64(make([]byte, 0), uint64(valIdx))
			val := ssz.MarshalUint64(make([]byte, 0), uint64(epoch))
			if err := bkt.Put(key, val); err != nil {
				return err
			}
		}
		return nil
	})

}

func (s *Store) SaveAttestationRecordForValidator(
	ctx context.Context,
	validatorIdx types.ValidatorIdx,
	attestation *ethpb.IndexedAttestation,
) error {
	ctx, span := trace.StartSpan(ctx, "AdvancedSlasherDB.SaveAttestationRecordForValidator")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationRecordsBucket)
		encIdx := ssz.MarshalUint64(make([]byte, 0), uint64(validatorIdx))
		encEpoch := ssz.MarshalUint64(make([]byte, 0), attestation.Data.Target.Epoch)
		key := append(encIdx, encEpoch...)
		value := make([]byte, 16)
		copy(value[0:8], ssz.MarshalUint64(make([]byte, 0), attestation.Data.Source.Epoch))
		copy(value[8:], ssz.MarshalUint64(make([]byte, 0), attestation.Data.Target.Epoch))
		return bkt.Put(key, value)
	})
}

func (s *Store) SaveChunks(
	ctx context.Context, kind types.ChunkKind, chunkKeys []uint64, chunks [][]types.Span,
) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveChunks")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(slasherChunksBucket)
		for i := 0; i < len(chunkKeys); i++ {
			keyBytes := ssz.MarshalUint64(make([]byte, 0), chunkKeys[i])
			val := make([]byte, 0)
			for j := 0; j < len(chunks[i]); j++ {
				val = append(val, ssz.MarshalUint16(make([]byte, 0), uint16(chunks[i][j]))...)
			}
			keyBytes = append(ssz.MarshalUint8(make([]byte, 0), uint8(kind)), keyBytes...)
			if err := bkt.Put(keyBytes, val); err != nil {
				return err
			}
		}
		return nil
	})
}
