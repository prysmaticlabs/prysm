package kv

import (
	"context"
	"errors"
	"fmt"

	ssz "github.com/ferranbt/fastssz"
	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// LatestEpochAttestedForValidator given a validator index returns the latest
// epoch we have recorded the validator attested for.
func (s *Store) LatestEpochAttestedForValidators(
	ctx context.Context, validatorIndices []types.ValidatorIndex,
) ([]types.Epoch, []bool, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LatestEpochAttestedForValidators")
	defer span.End()
	epochs := make([]types.Epoch, 0)
	epochsExist := make([]bool, 0)
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestedEpochsByValidator)
		for _, valIdx := range validatorIndices {
			enc, err := valIdx.MarshalSSZ()
			if err != nil {
				return err
			}
			epochBytes := bkt.Get(enc)
			if epochBytes == nil {
				epochsExist = append(epochsExist, false)
				epochs = append(epochs, 0)
				continue
			}
			var epoch types.Epoch
			if err := epoch.UnmarshalSSZ(epochBytes); err != nil {
				return err
			}
			epochsExist = append(epochsExist, true)
			epochs = append(epochs, epoch)
		}
		return nil
	})
	return epochs, epochsExist, err
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
) (*slashertypes.CompactAttestation, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.AttestationRecordForValidator")
	defer span.End()
	var record *slashertypes.CompactAttestation
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

// SaveAttestationRecordsForValidators saves an attestation records for the
// specified validator indices.
func (s *Store) SaveAttestationRecordsForValidators(
	ctx context.Context,
	validatorIndices []types.ValidatorIndex,
	attestations []*slashertypes.CompactAttestation,
) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveAttestationRecordsForValidators")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationRecordsBucket)
		for _, valIdx := range validatorIndices {
			encIdx, err := valIdx.MarshalSSZ()
			if err != nil {
				return err
			}
			for _, att := range attestations {
				encEpoch := ssz.MarshalUint64(make([]byte, 0), att.Target)
				key := append(encIdx, encEpoch...)
				value, err := encodeAttestationRecord(att)
				if err != nil {
					return err
				}
				if err := bkt.Put(key, value); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// LoadSlasherChunks given a chunk kind and a disk keys, retrieves chunks for a validator
// min or max span used by slasher from our database.
func (s *Store) LoadSlasherChunks(
	ctx context.Context, kind slashertypes.ChunkKind, diskKeys []uint64,
) ([][]uint16, []bool, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LoadSlasherChunk")
	defer span.End()
	chunks := make([][]uint16, 0)
	var exists []bool
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(slasherChunksBucket)
		for _, diskKey := range diskKeys {
			keyBytes := ssz.MarshalUint64(make([]byte, 0), diskKey)
			keyBytes = append(ssz.MarshalUint8(make([]byte, 0), uint8(kind)), keyBytes...)
			chunkBytes := bkt.Get(keyBytes)
			if chunkBytes == nil {
				chunks = append(chunks, []uint16{})
				exists = append(exists, false)
				return nil
			}
			chunk := make([]uint16, 0)
			for i := 0; i < len(chunkBytes); i += 2 {
				distance := ssz.UnmarshallUint16(chunkBytes[i : i+2])
				chunk = append(chunk, distance)
			}
			chunks = append(chunks, chunk)
			exists = append(exists, true)
		}
		return nil
	})
	return chunks, exists, err
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

// Encode an attestation record's required fields for slashing protection into bytes.
func encodeAttestationRecord(att *slashertypes.CompactAttestation) ([]byte, error) {
	if att == nil {
		return nil, errors.New("encoding nil attestation")
	}
	value := make([]byte, 48)
	copy(value[0:8], ssz.MarshalUint64(make([]byte, 0), att.Source))
	copy(value[8:16], ssz.MarshalUint64(make([]byte, 0), att.Target))
	copy(value[16:], att.SigningRoot[:])
	return value, nil
}

// Decode attestation record from bytes.
func decodeAttestationRecord(encoded []byte) (*slashertypes.CompactAttestation, error) {
	if len(encoded) != 48 {
		return nil, fmt.Errorf("wrong length for encoded attestation record, want 48, got %d", len(encoded))
	}
	var sr [32]byte
	copy(sr[:], encoded[16:])
	return &slashertypes.CompactAttestation{
		Source:      ssz.UnmarshallUint64(encoded[0:8]),
		Target:      ssz.UnmarshallUint64(encoded[8:16]),
		SigningRoot: sr,
	}, nil
}
