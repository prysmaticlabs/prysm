package kv

import (
	"bytes"
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"

	ssz "github.com/ferranbt/fastssz"
	"github.com/golang/snappy"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	slashertypes "github.com/prysmaticlabs/prysm/beacon-chain/slasher/types"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	bolt "go.etcd.io/bbolt"
	"go.opencensus.io/trace"
)

// LastEpochWrittenForValidator given a validator index returns the latest
// epoch we have recorded the validator attested for.
func (s *Store) LastEpochWrittenForValidators(
	ctx context.Context, validatorIndices []types.ValidatorIndex,
) ([]*slashertypes.AttestedEpochForValidator, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LastEpochWrittenForValidators")
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

// SaveLastEpochWrittenForValidators updates the latest epoch a slice
// of validator indices has attested to.
func (s *Store) SaveLastEpochWrittenForValidators(
	ctx context.Context, validatorIndices []types.ValidatorIndex, epoch types.Epoch,
) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveLastEpochWrittenForValidators")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(attestedEpochsByValidator)
		val, err := epoch.MarshalSSZ()
		if err != nil {
			return err
		}
		for _, valIdx := range validatorIndices {
			key, err := valIdx.MarshalSSZ()
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
	ctx context.Context, attestations []*slashertypes.IndexedAttestationWrapper,
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
				key := append(encEpoch, encIdx...)
				encExistingAttRecord := bkt.Get(key)
				if len(encExistingAttRecord) < 32 {
					continue
				}
				existingSigningRoot := bytesutil.ToBytes32(encExistingAttRecord[:32])
				if existingSigningRoot != att.SigningRoot {
					existingAttRecord, err := decodeAttestationRecord(encExistingAttRecord)
					if err != nil {
						return err
					}
					doubleVotes = append(doubleVotes, &slashertypes.AttesterDoubleVote{
						ValidatorIndex:         types.ValidatorIndex(valIdx),
						Target:                 att.IndexedAttestation.Data.Target.Epoch,
						PrevAttestationWrapper: existingAttRecord,
						AttestationWrapper:     att,
					})
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
) (*slashertypes.IndexedAttestationWrapper, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.AttestationRecordForValidator")
	defer span.End()
	var record *slashertypes.IndexedAttestationWrapper
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
		key := append(encEpoch, encIdx...)
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
	attestations []*slashertypes.IndexedAttestationWrapper,
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
			value, err := encodeAttestationRecord(att)
			if err != nil {
				return err
			}
			for _, valIdx := range att.IndexedAttestation.AttestingIndices {
				encIdx := ssz.MarshalUint64(make([]byte, 0), valIdx)
				key := append(encEpoch, encIdx...)
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
	ctx context.Context, kind slashertypes.ChunkKind, diskKeys [][]byte,
) ([][]uint16, []bool, error) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.LoadSlasherChunk")
	defer span.End()
	chunks := make([][]uint16, 0)
	var exists []bool
	err := s.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(slasherChunksBucket)
		for _, diskKey := range diskKeys {
			key := append(ssz.MarshalUint8(make([]byte, 0), uint8(kind)), diskKey...)
			chunkBytes := bkt.Get(key)
			if chunkBytes == nil {
				chunks = append(chunks, []uint16{})
				exists = append(exists, false)
				continue
			}
			chunk, err := decodeSlasherChunk(chunkBytes)
			if err != nil {
				return err
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
	ctx context.Context, kind slashertypes.ChunkKind, chunkKeys [][]byte, chunks [][]uint16,
) error {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.SaveSlasherChunks")
	defer span.End()
	return s.db.Update(func(tx *bolt.Tx) error {
		bkt := tx.Bucket(slasherChunksBucket)
		for i := 0; i < len(chunkKeys); i++ {
			key := append(ssz.MarshalUint8(make([]byte, 0), uint8(kind)), chunkKeys[i]...)
			enc := encodeSlasherChunk(chunks[i])
			if err := bkt.Put(key, enc); err != nil {
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
	ctx context.Context, proposals []*slashertypes.SignedBlockHeaderWrapper,
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
			encExistingProposalWrapper := bkt.Get(key)
			if len(encExistingProposalWrapper) < 32 {
				continue
			}
			existingSigningRoot := bytesutil.ToBytes32(encExistingProposalWrapper[:32])
			if existingSigningRoot != proposal.SigningRoot {
				existingProposalWrapper, err := decodeProposalRecord(encExistingProposalWrapper)
				if err != nil {
					return err
				}
				doubleProposals = append(doubleProposals, &slashertypes.DoubleBlockProposal{
					PrevBeaconBlockWrapper: existingProposalWrapper,
					BeaconBlockWrapper:     proposal,
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
	ctx context.Context, proposals []*slashertypes.SignedBlockHeaderWrapper,
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
			proposalEnc, err := encodeProposalRecord(proposal)
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

func (s *Store) PruneProposals(ctx context.Context, epoch types.Epoch, historySize uint64) error {
	// + 1 here so we can prune everything less than this, but not equal.
	endPruneSlot, err := helpers.StartSlot(epoch)
	if err != nil {
		return err
	}
	endEnc, err := endPruneSlot.MarshalSSZ()
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		proposalBkt := tx.Bucket(proposalRecordsBucket)
		c := proposalBkt.Cursor()
		for k, _ := c.Seek(endEnc); k != nil && prefixLessThan(k, endEnc); k, _ = c.Prev() {
			if err := proposalBkt.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) PruneAttestations(ctx context.Context, currentEpoch types.Epoch, historySize uint64) error {
	// + 1 here so we can prune everything less than this, but not equal.
	endPruneEpoch := currentEpoch
	epochEnc, err := endPruneEpoch.MarshalSSZ()
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		attBkt := tx.Bucket(attestationRecordsBucket)
		c := attBkt.Cursor()
		for k, _ := c.Seek(epochEnc); k != nil && prefixLessThan(k, epochEnc); k, _ = c.Prev() {
			if err := attBkt.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}

func prefixLessThan(key []byte, lessThan []byte) bool {
	encSlot := key[:8]
	return bytes.Compare(encSlot, lessThan) < 0
}

// Disk key for a validator proposal, including a slot+validatorIndex as a byte slice.
func keyForValidatorProposal(proposal *slashertypes.SignedBlockHeaderWrapper) ([]byte, error) {
	encSlot, err := proposal.SignedBeaconBlockHeader.Header.Slot.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	encValidatorIdx, err := proposal.SignedBeaconBlockHeader.Header.ProposerIndex.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(encSlot, encValidatorIdx...), nil
}

func encodeSlasherChunk(chunk []uint16) []byte {
	val := make([]byte, 0)
	for i := 0; i < len(chunk); i++ {
		val = append(val, ssz.MarshalUint16(make([]byte, 0), chunk[i])...)
	}
	return snappy.Encode(nil, val)
}

func decodeSlasherChunk(enc []byte) ([]uint16, error) {
	chunkBytes, err := snappy.Decode(nil, enc)
	if err != nil {
		return nil, err
	}
	chunk := make([]uint16, 0)
	for i := 0; i < len(chunkBytes); i += 2 {
		distance := ssz.UnmarshallUint16(chunkBytes[i : i+2])
		chunk = append(chunk, distance)
	}
	return chunk, nil
}

// Decode attestation record from bytes.
func encodeAttestationRecord(att *slashertypes.IndexedAttestationWrapper) ([]byte, error) {
	if att == nil || att.IndexedAttestation == nil {
		return []byte{}, errors.New("nil proposal record")
	}
	encodedAtt, err := att.IndexedAttestation.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(att.SigningRoot[:], encodedAtt...), nil
}

// Decode attestation record from bytes.
func decodeAttestationRecord(encoded []byte) (*slashertypes.IndexedAttestationWrapper, error) {
	if len(encoded) < 32 {
		return nil, fmt.Errorf("wrong length for encoded attestation record, want 32, got %d", len(encoded))
	}
	signingRoot := encoded[:32]
	decodedAtt := &ethpb.IndexedAttestation{}
	if err := decodedAtt.UnmarshalSSZ(encoded[32:]); err != nil {
		return nil, err
	}
	return &slashertypes.IndexedAttestationWrapper{
		IndexedAttestation: decodedAtt,
		SigningRoot:        bytesutil.ToBytes32(signingRoot),
	}, nil
}

func encodeProposalRecord(blkHdr *slashertypes.SignedBlockHeaderWrapper) ([]byte, error) {
	if blkHdr == nil || blkHdr.SignedBeaconBlockHeader == nil {
		return []byte{}, errors.New("nil proposal record")
	}
	encodedHdr, err := blkHdr.SignedBeaconBlockHeader.MarshalSSZ()
	if err != nil {
		return nil, err
	}
	return append(blkHdr.SigningRoot[:], encodedHdr...), nil
}

func decodeProposalRecord(encoded []byte) (*slashertypes.SignedBlockHeaderWrapper, error) {
	if len(encoded) < 32 {
		return nil, fmt.Errorf("wrong length for encoded proposal record, want 32, got %d", len(encoded))
	}
	signingRoot := encoded[:32]
	decodedBlkHdr := &ethpb.SignedBeaconBlockHeader{}
	if err := decodedBlkHdr.UnmarshalSSZ(encoded[32:]); err != nil {
		return nil, err
	}
	return &slashertypes.SignedBlockHeaderWrapper{
		SignedBeaconBlockHeader: decodedBlkHdr,
		SigningRoot:             bytesutil.ToBytes32(signingRoot),
	}, nil
}
