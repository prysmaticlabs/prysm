package stateutil

import (
	"encoding/binary"

	"github.com/pkg/errors"
	htrutils2 "github.com/prysmaticlabs/prysm/encoding/htrutils"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// PendingAttRootWithHasher describes a method from which the hash tree root
// of a pending attestation is returned.
func PendingAttRootWithHasher(hasher htrutils2.HashFn, att *ethpb.PendingAttestation) ([32]byte, error) {
	var fieldRoots [][32]byte

	// Bitfield.
	aggregationRoot, err := htrutils2.BitlistRoot(hasher, att.AggregationBits, params.BeaconConfig().MaxValidatorsPerCommittee)
	if err != nil {
		return [32]byte{}, err
	}
	// Attestation data.
	attDataRoot, err := attDataRootWithHasher(hasher, att.Data)
	if err != nil {
		return [32]byte{}, err
	}
	inclusionBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(inclusionBuf, uint64(att.InclusionDelay))
	// Inclusion delay.
	inclusionRoot := bytesutil.ToBytes32(inclusionBuf)

	proposerBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(proposerBuf, uint64(att.ProposerIndex))
	// Proposer index.
	proposerRoot := bytesutil.ToBytes32(proposerBuf)

	fieldRoots = [][32]byte{aggregationRoot, attDataRoot, inclusionRoot, proposerRoot}

	return htrutils2.BitwiseMerkleizeArrays(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

// PendingAttEncKey returns the encoded key in bytes of input `pendingAttestation`,
// the returned key bytes can be used for caching purposes.
func PendingAttEncKey(att *ethpb.PendingAttestation) []byte {
	enc := make([]byte, 2192)

	if att != nil {
		copy(enc[0:2048], att.AggregationBits)

		inclusionBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(inclusionBuf, uint64(att.InclusionDelay))
		copy(enc[2048:2056], inclusionBuf)

		attDataBuf := marshalAttData(att.Data)
		copy(enc[2056:2184], attDataBuf)

		proposerBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(proposerBuf, uint64(att.ProposerIndex))
		copy(enc[2184:2192], proposerBuf)
	}

	return enc
}

func attDataRootWithHasher(hasher htrutils2.HashFn, data *ethpb.AttestationData) ([32]byte, error) {
	fieldRoots := make([][]byte, 5)

	if data != nil {
		// Slot.
		slotBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(slotBuf, uint64(data.Slot))
		slotRoot := bytesutil.ToBytes32(slotBuf)
		fieldRoots[0] = slotRoot[:]

		// CommitteeIndex.
		indexBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(indexBuf, uint64(data.CommitteeIndex))
		interRoot := bytesutil.ToBytes32(indexBuf)
		fieldRoots[1] = interRoot[:]

		// Beacon block root.
		blockRoot := bytesutil.ToBytes32(data.BeaconBlockRoot)
		fieldRoots[2] = blockRoot[:]

		// Source
		sourceRoot, err := htrutils2.CheckpointRoot(hasher, data.Source)
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute source checkpoint merkleization")
		}
		fieldRoots[3] = sourceRoot[:]

		// Target
		targetRoot, err := htrutils2.CheckpointRoot(hasher, data.Target)
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute target checkpoint merkleization")
		}
		fieldRoots[4] = targetRoot[:]
	}

	return htrutils2.BitwiseMerkleize(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func marshalAttData(data *ethpb.AttestationData) []byte {
	enc := make([]byte, 128)

	if data != nil {
		// Slot.
		slotBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(slotBuf, uint64(data.Slot))
		copy(enc[0:8], slotBuf)

		// Committee index.
		indexBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(indexBuf, uint64(data.CommitteeIndex))
		copy(enc[8:16], indexBuf)

		copy(enc[16:48], data.BeaconBlockRoot)

		// Source epoch and root.
		if data.Source != nil {
			sourceEpochBuf := make([]byte, 8)
			binary.LittleEndian.PutUint64(sourceEpochBuf, uint64(data.Source.Epoch))
			copy(enc[48:56], sourceEpochBuf)
			copy(enc[56:88], data.Source.Root)
		}

		// Target.
		if data.Target != nil {
			targetEpochBuf := make([]byte, 8)
			binary.LittleEndian.PutUint64(targetEpochBuf, uint64(data.Target.Epoch))
			copy(enc[88:96], targetEpochBuf)
			copy(enc[96:128], data.Target.Root)
		}
	}

	return enc
}
