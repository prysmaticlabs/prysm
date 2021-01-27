package kv

import (
	"context"
	"errors"
	"fmt"

	ssz "github.com/ferranbt/fastssz"
	"github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// LatestEpochAttestedForValidator given a validator index returns the latest
// epoch we have recorded the validator attested for.
func (s *Store) LatestEpochAttestedForValidator(
	ctx context.Context, validatorIdx types.ValidatorIndex,
) (types.Epoch, bool, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LatestEpochAttestedForValidator")
	defer span.End()
	var epoch types.Epoch
	var exists bool
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestedEpochsByValidator)
		enc, err := validatorIdx.MarshalSSZ()
		if err != nil {
			return err
		}
		epochBytes := bkt.Get(enc)
		if epochBytes == nil {
			return nil
		}
		exists = true
		return epoch.UnmarshalSSZ(epochBytes)
	})
	return epoch, exists, err
}

// SaveLatestEpochAttestedForValidators updates the latest epoch a slice
// of validator indices has attested to.
func (s *Store) SaveLatestEpochAttestedForValidators(
	ctx context.Context, validatorIndices []types.ValidatorIndex, epoch types.Epoch,
) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveLatestEpochAttestedForValidator")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestedEpochsByValidator)
		for _, valIdx := range validatorIndices {
			key, err := valIdx.MarshalSSZ()
			if err != nil {
				return err
			}
			val, err := epoch.MarshalSSZ()
			if err != nil {
				return err
			}
			if err := bkt.Put(key, val); err != nil {
				return err
			}
		}
		return nil
	})
}

// AttestationRecordForValidator given a validator index and a target epoch,
// retrieves an existing attestation record we have stored in the database.
func (s *Store) AttestationRecordForValidator(
	ctx context.Context, validatorIdx types.ValidatorIndex, targetEpoch types.Epoch,
) (*slashertypes.AttestationRecord, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.AttestationRecordForValidator")
	defer span.End()
	var record *slashertypes.AttestationRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationRecordsBucket)
		encIdx, err := validatorIdx.MarshalSSZ()
		if err != nil {
			return err
		}
		encEpoch, err := targetEpoch.MarshalSSZ()
		if err != nil {
			return err
		}
		key := append(encIdx, encEpoch...)
		value := bkt.Get(key)
		if value == nil {
			return nil
		}
		decoded, err := decodeAttestationRecord(value)
		if err != nil {
			return err
		}
		record = decoded
		return nil
	})
	return record, err
}

// SaveAttestationRecordForValidator saves an attestation record for a validator index.
func (s *Store) SaveAttestationRecordForValidator(
	ctx context.Context,
	validatorIdx types.ValidatorIndex,
	signingRoot [32]byte,
	attestation *ethpb.IndexedAttestation,
) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveAttestationRecordForValidator")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationRecordsBucket)
		encIdx, err := validatorIdx.MarshalSSZ()
		if err != nil {
			return err
		}
		encEpoch := ssz.MarshalUint64(make([]byte, 0), attestation.Data.Target.Epoch)
		key := append(encIdx, encEpoch...)
		value, err := encodeIndexedAttestationRecord(attestation, signingRoot)
		if err != nil {
			return err
		}
		return bkt.Put(key, value)
	})
}

// LoadSlasherChunk given a chunk kind and a disk key, retrieves a chunk for a validator
// min or max span used by slasher from our database.
func (s *Store) LoadSlasherChunk(
	ctx context.Context, kind slashertypes.ChunkKind, diskKey uint64,
) ([]uint16, bool, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LoadSlasherChunk")
	defer span.End()
	var chunk []uint16
	var exists bool
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(slasherChunksBucket)
		keyBytes := ssz.MarshalUint64(make([]byte, 0), diskKey)
		keyBytes = append(ssz.MarshalUint8(make([]byte, 0), uint8(kind)), keyBytes...)
		chunkBytes := bkt.Get(keyBytes)
		if chunkBytes == nil {
			return nil
		}
		chunk = make([]uint16, 0)
		for i := 0; i < len(chunkBytes); i += 2 {
			distance := ssz.UnmarshallUint16(chunkBytes[i : i+2])
			chunk = append(chunk, distance)
		}
		exists = true
		return nil
	})
	return chunk, exists, err
}

// SaveSlasherChunk given a chunk kind, list of disk keys, and list of chunks,
// saves the chunks to our database for use by slasher in slashing detection.
func (s *Store) SaveSlasherChunks(
	ctx context.Context, kind slashertypes.ChunkKind, chunkKeys []uint64, chunks [][]uint16,
) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveSlasherChunks")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(slasherChunksBucket)
		for i := 0; i < len(chunkKeys); i++ {
			keyBytes := ssz.MarshalUint64(make([]byte, 0), chunkKeys[i])
			val := make([]byte, 0)
			for j := 0; j < len(chunks[i]); j++ {
				val = append(val, ssz.MarshalUint16(make([]byte, 0), chunks[i][j])...)
			}
			keyBytes = append(ssz.MarshalUint8(make([]byte, 0), uint8(kind)), keyBytes...)
			if err := bkt.Put(keyBytes, val); err != nil {
				return err
			}
		}
		return nil
	})
}

// Encode an indexed attestation's required fields for slashing protection into bytes.
func encodeIndexedAttestationRecord(att *ethpb.IndexedAttestation, signingRoot [32]byte) ([]byte, error) {
	if att == nil || att.Data == nil || att.Data.Target == nil || att.Data.Source == nil {
		return nil, errors.New("encoding nil attestation")
	}
	value := make([]byte, 48)
	copy(value[0:8], ssz.MarshalUint64(make([]byte, 0), att.Data.Source.Epoch))
	copy(value[8:16], ssz.MarshalUint64(make([]byte, 0), att.Data.Target.Epoch))
	copy(value[16:], signingRoot[:])
	return value, nil
}

// Decode attestation record from bytes.
func decodeAttestationRecord(encoded []byte) (*slashertypes.AttestationRecord, error) {
	if len(encoded) != 48 {
		return nil, fmt.Errorf("wrong length for encoded attestation record, want 48, got %d", len(encoded))
	}
	var sr [32]byte
	copy(sr[:], encoded[16:])
	return &slashertypes.AttestationRecord{
		Source:      ssz.UnmarshallUint64(encoded[0:8]),
		Target:      ssz.UnmarshallUint64(encoded[8:16]),
		SigningRoot: sr,
	}, nil
}
