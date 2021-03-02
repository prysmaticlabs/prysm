package kv

import (
	"bytes"
	"context"
	"fmt"

	ssz "github.com/ferranbt/fastssz"
	types "github.com/prysmaticlabs/eth2-types"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	slashpb "github.com/prysmaticlabs/prysm/proto/slashing"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// LatestEpochAttestedForValidator given a validator index returns the latest
// epoch we have recorded the validator attested for.
func (s *Store) LatestEpochAttestedForValidators(
	ctx context.Context, validatorIndices []types.ValidatorIndex,
) ([]*slashertypes.AttestedEpochForValidator, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LatestEpochAttestedForValidators")
	defer span.End()
	attestedEpochs := make([]*slashertypes.AttestedEpochForValidator, 0)
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestedEpochsByValidator)
		for _, valIdx := range validatorIndices {
			enc, err := valIdx.MarshalSSZ()
			if err != nil {
				return err
			}
			epochBytes := bkt.Get(enc)
			if epochBytes != nil {
				var epoch types.Epoch
				if err := epoch.UnmarshalSSZ(epochBytes); err != nil {
					return err
				}
				attestedEpochs = append(attestedEpochs, &slashertypes.AttestedEpochForValidator{
					ValidatorIndex: valIdx,
					Epoch:          epoch,
				})
			}
		}
		return nil
	})
	return attestedEpochs, err
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

// CheckDoubleAttesterVotes retries any slashable double votes that exist
// for a series of input attestations.
func (s *Store) CheckAttesterDoubleVotes(
	ctx context.Context, attestations []*slashpb.IndexedAttestationWrapper,
) ([]*slashertypes.AttesterDoubleVote, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.CheckAttesterDoubleVotes")
	defer span.End()
	doubleVotes := make([]*slashertypes.AttesterDoubleVote, 0)
	err := s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationRecordsBucket)
		for _, att := range attestations {
			encEpoch, err := att.IndexedAttestation.Data.Target.Epoch.MarshalSSZ()
			if err != nil {
				return err
			}
			for _, valIdx := range att.IndexedAttestation.AttestingIndices {
				encIdx := ssz.MarshalUint64(make([]byte, 0), valIdx)
				key := append(encIdx, encEpoch...)
				existingEncodedRecord := bkt.Get(key)
				if existingEncodedRecord != nil {
					existingAtt, err := decodeAttestationRecord(existingEncodedRecord)
					if err != nil {
						return err
					}
					if !bytes.Equal(existingAtt.SigningRoot, att.SigningRoot) {
						doubleVotes = append(doubleVotes, &slashertypes.AttesterDoubleVote{
							ValidatorIndex:  types.ValidatorIndex(valIdx),
							Target:          att.IndexedAttestation.Data.Target.Epoch,
							SigningRoot:     bytesutil.ToBytes32(att.SigningRoot),
							PrevSigningRoot: bytesutil.ToBytes32(existingAtt.SigningRoot),
						})
					}
				}
			}
		}
		return nil
	})
	return doubleVotes, err
}

// AttestationRecordForValidator given a validator index and a target epoch,
// retrieves an existing attestation record we have stored in the database.
func (s *Store) AttestationRecordForValidator(
	ctx context.Context, validatorIdx types.ValidatorIndex, targetEpoch types.Epoch,
) (*slashpb.IndexedAttestationWrapper, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.AttestationRecordForValidator")
	defer span.End()
	var record *slashpb.IndexedAttestationWrapper
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
	attestations []*slashpb.IndexedAttestationWrapper,
) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveAttestationRecordsForValidators")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestationRecordsBucket)
		for _, att := range attestations {
			encEpoch, err := att.IndexedAttestation.Data.Target.Epoch.MarshalSSZ()
			if err != nil {
				return err
			}
			value, err := att.Marshal()
			if err != nil {
				return err
			}
			for _, valIdx := range att.IndexedAttestation.AttestingIndices {
				encIdx := ssz.MarshalUint64(make([]byte, 0), valIdx)
				key := append(encIdx, encEpoch...)
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
				continue
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

// CheckDoubleBlockProposals takes in a list of proposals and for each,
// checks if there already exists a proposal at the same slot+validatorIndex combination. If so,
// We check if the existing signing root is not-empty and is different than the incoming
// proposal signing root. If so, we return a double block proposal object.
func (s *Store) CheckDoubleBlockProposals(
	ctx context.Context, proposals []*slashpb.SignedBlkHeaderWrapper,
) ([]*slashertypes.DoubleBlockProposal, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.CheckDoubleBlockProposals")
	defer span.End()
	doubleProposals := make([]*slashertypes.DoubleBlockProposal, 0, len(proposals))
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(proposalRecordsBucket)
		for _, proposal := range proposals {
			key, err := keyForValidatorProposal(proposal)
			if err != nil {
				return err
			}
			existingSigningRoot := bkt.Get(key)
			if existingSigningRoot != nil && !bytes.Equal(existingSigningRoot, proposal.SigningRoot) {
				doubleProposals = append(doubleProposals, &slashertypes.DoubleBlockProposal{
					Slot:                proposal.SignedBlockHeader.Header.Slot,
					ProposerIndex:       proposal.SignedBlockHeader.Header.ProposerIndex,
					IncomingSigningRoot: bytesutil.ToBytes32(proposal.SigningRoot),
					ExistingSigningRoot: bytesutil.ToBytes32(existingSigningRoot),
				})
			}
		}
		return nil
	})
	return doubleProposals, err
}

// SaveBlockProposals takes in a list of block proposals and saves them to our
// proposal records bucket in the database.
func (s *Store) SaveBlockProposals(
	ctx context.Context, proposals []*slashpb.SignedBlkHeaderWrapper,
) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveBlockProposals")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(proposalRecordsBucket)
		for _, proposal := range proposals {
			key, err := keyForValidatorProposal(proposal)
			if err != nil {
				return err
			}
			proposalEnc, err := proposal.Marshal()
			if err != nil {
				return err
			}
			if err := bkt.Put(key, proposalEnc); err != nil {
				return err
			}
		}
		return nil
	})
}

// Disk key for a validator proposal, including a slot+validatorIndex as a byte slice.
func keyForValidatorProposal(proposal *slashpb.SignedBlkHeaderWrapper) ([]byte, error) {
	encSlot, err := proposal.SignedBlockHeader.Header.Slot.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	encValidatorIdx, err := proposal.SignedBlockHeader.Header.ProposerIndex.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(encSlot, encValidatorIdx...), nil
}

// Decode attestation record from bytes.
func decodeAttestationRecord(encoded []byte) (*slashpb.IndexedAttestationWrapper, error) {
	if len(encoded) != 48 {
		return nil, fmt.Errorf("wrong length for encoded attestation record, want 48, got %d", len(encoded))
	}
	indexedAttWrapper := &slashpb.IndexedAttestationWrapper{}
	if err := indexedAttWrapper.Unmarshal(encoded); err != nil {
		return nil, err
	}
	return indexedAttWrapper, nil
}
