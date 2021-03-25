package stateutil

import (
	"encoding/binary"
	"fmt"

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
func PendingAttRootWithHasher(hasher htrutils.HashFn, att *pb.PendingAttestation) ([32]byte, error) {
	var fieldRoots [][32]byte

	// Bitfield.
	aggregationRoot, err := htrutils.BitlistRoot(hasher, att.AggregationBits, params.BeaconConfig().MaxValidatorsPerCommittee)
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

	return htrutils.BitwiseMerkleizeArrays(hasher, fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

// PendingAttEncKey returns the encoded key in bytes of input `pendingAttestation`,
// the returned key bytes can be used for caching purposes.
func PendingAttEncKey(att *pb.PendingAttestation) []byte {
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

// HandlePendingAttestation handles a list of pending attestations and returns the state root for each in a list back.
func HandlePendingAttestation(val []*pb.PendingAttestation, indices []uint64, convertAll bool) ([][32]byte, error) {
	length := len(indices)
	if convertAll {
		length = len(val)
	}
	roots := make([][32]byte, 0, length)
	hasher := hashutil.CustomSHA256Hasher()
	rootCreator := func(input *pb.PendingAttestation) error {
		newRoot, err := PendingAttRootWithHasher(hasher, input)
		if err != nil {
			return err
		}
		roots = append(roots, newRoot)
		return nil
	}
	if convertAll {
		for i := range val {
			err := rootCreator(val[i])
			if err != nil {
				return nil, err
			}
		}
		return roots, nil
	}
	if len(val) > 0 {
		for _, idx := range indices {
			if idx > uint64(len(val))-1 {
				return nil, fmt.Errorf("index %d greater than number of pending attestations %d", idx, len(val))
			}
			err := rootCreator(val[idx])
			if err != nil {
				return nil, err
			}
		}
	}
	return roots, nil
}

func attDataRootWithHasher(hasher htrutils.HashFn, data *ethpb.AttestationData) ([32]byte, error) {
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
