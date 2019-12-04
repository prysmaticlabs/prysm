package stateutil

import (
	"encoding/binary"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func attestationDataRoot(data *ethpb.AttestationData) ([32]byte, error) {
	fieldRoots := make([][]byte, 5)

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
	fieldRoots[2] = data.BeaconBlockRoot

	// Source
	sourceRoot, err := checkpointRoot(data.Source)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute source checkpoint merkleization")
	}
	fieldRoots[3] = sourceRoot[:]

	// Target
	targetRoot, err := checkpointRoot(data.Target)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute target checkpoint merkleization")
	}
	fieldRoots[4] = targetRoot[:]

	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, nil
	}
	return root, nil
}

func pendingAttestationRoot(att *pb.PendingAttestation) ([32]byte, error) {
	fieldRoots := make([][]byte, 4)

	// Bitfield.
	aggregationRoot, err := bitlistRoot(att.AggregationBits, 2048)
	if err != nil {
		panic(err)
	}
	fieldRoots[0] = aggregationRoot[:]

	// Attestation data.
	attDataRoot, err := attestationDataRoot(att.Data)
	if err != nil {
		return [32]byte{}, nil
	}
	fieldRoots[1] = attDataRoot[:]

	// Inclusion delay.
	inclusionBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(inclusionBuf, att.InclusionDelay)
	inclusionRoot := bytesutil.ToBytes32(inclusionBuf)
	fieldRoots[2] = inclusionRoot[:]

	// Proposer index.
	proposerBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(proposerBuf, att.ProposerIndex)
	proposerRoot := bytesutil.ToBytes32(proposerBuf)
	fieldRoots[3] = proposerRoot[:]

	root, err := bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
	if err != nil {
		return [32]byte{}, nil
	}
	return root, nil
}
