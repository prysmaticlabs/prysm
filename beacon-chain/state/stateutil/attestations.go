package stateutil

import (
	"encoding/binary"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/htrutils"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// PendingAttestationRoot describes a method from which the hash tree root
// of a pending attestation is returned.
func PendingAttestationRoot(hasher htrutils.HashFn, att *pb.PendingAttestation) ([32]byte, error) {
	var fieldRoots [][32]byte
	if att != nil {
		// Bitfield.
		aggregationRoot, err := htrutils.BitlistRoot(hasher, att.AggregationBits, params.BeaconConfig().MaxValidatorsPerCommittee)
		if err != nil {
			return [32]byte{}, err
		}
		// Attestation data.
		attDataRoot, err := attestationDataRoot(hasher, att.Data)
		if err != nil {
			return [32]byte{}, err
		}
		inclusionBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(inclusionBuf, att.InclusionDelay)
		// Inclusion delay.
		inclusionRoot := bytesutil.ToBytes32(inclusionBuf)

		proposerBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(proposerBuf, att.ProposerIndex)
		// Proposer index.
		proposerRoot := bytesutil.ToBytes32(proposerBuf)

		fieldRoots = [][32]byte{aggregationRoot, attDataRoot, inclusionRoot, proposerRoot}
	}
	return htrutils.BitwiseMerkleizeArrays(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

// AttestationDataRoot describes a method that serves as a HashTreeRoot function for attestation data.
func AttestationDataRoot(data *ethpb.AttestationData) ([32]byte, error) {
	return attestationDataRoot(hashutil.CustomSHA256Hasher(), data)
}

func marshalAttestationData(data *ethpb.AttestationData) []byte {
	enc := make([]byte, 128)

	if data != nil {
		// Slot.
		slotBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(slotBuf, data.Slot)
		copy(enc[0:8], slotBuf)

		// Committee index.
		indexBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(indexBuf, data.CommitteeIndex)
		copy(enc[8:16], indexBuf)

		copy(enc[16:48], data.BeaconBlockRoot)

		// Source epoch and root.
		if data.Source != nil {
			sourceEpochBuf := make([]byte, 8)
			binary.LittleEndian.PutUint64(sourceEpochBuf, data.Source.Epoch)
			copy(enc[48:56], sourceEpochBuf)
			copy(enc[56:88], data.Source.Root)
		}

		// Target.
		if data.Target != nil {
			targetEpochBuf := make([]byte, 8)
			binary.LittleEndian.PutUint64(targetEpochBuf, data.Target.Epoch)
			copy(enc[88:96], targetEpochBuf)
			copy(enc[96:128], data.Target.Root)
		}
	}

	return enc
}

func attestationDataRoot(hasher htrutils.HashFn, data *ethpb.AttestationData) ([32]byte, error) {
	fieldRoots := make([][]byte, 5)

	if data != nil {
		// Slot.
		slotBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(slotBuf, data.Slot)
		slotRoot := bytesutil.ToBytes32(slotBuf)
		fieldRoots[0] = slotRoot[:]

		// CommitteeIndex.
		indexBuf := make([]byte, 8)
		binary.LittleEndian.PutUint64(indexBuf, data.CommitteeIndex)
		interRoot := bytesutil.ToBytes32(indexBuf)
		fieldRoots[1] = interRoot[:]

		// Beacon block root.
		blockRoot := bytesutil.ToBytes32(data.BeaconBlockRoot)
		fieldRoots[2] = blockRoot[:]

		// Source
		sourceRoot, err := htrutils.CheckpointRoot(hasher, data.Source)
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute source checkpoint merkleization")
		}
		fieldRoots[3] = sourceRoot[:]

		// Target
		targetRoot, err := htrutils.CheckpointRoot(hasher, data.Target)
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not compute target checkpoint merkleization")
		}
		fieldRoots[4] = targetRoot[:]
	}

	return htrutils.BitwiseMerkleize(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}
