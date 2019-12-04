package stateutil

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
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

	return bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func pendingAttestationRoot(att *pb.PendingAttestation) ([32]byte, error) {
	fieldRoots := make([][]byte, 4)

	// Bitfield.
	aggregationRoot, err := bitlistRoot(att.AggregationBits, 2048)
	if err != nil {
		return [32]byte{}, err
	}
	fieldRoots[0] = aggregationRoot[:]

	// Attestation data.
	attDataRoot, err := attestationDataRoot(att.Data)
	if err != nil {
		return [32]byte{}, err
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

	return bitwiseMerkleize(fieldRoots, uint64(len(fieldRoots)), uint64(len(fieldRoots)))
}

func epochAttestationsRoot(atts []*pb.PendingAttestation) ([32]byte, error) {
	attsRoots := make([][]byte, 0)
	for i := 0; i < len(atts); i++ {
		pendingRoot, err := pendingAttestationRoot(atts[i])
		if err != nil {
			return [32]byte{}, errors.Wrap(err, "could not attestation merkleization")
		}
		attsRoots = append(attsRoots, pendingRoot[:])
	}
	attsRootsRoot, err := bitwiseMerkleize(attsRoots, uint64(len(attsRoots)), params.BeaconConfig().MaxAttestations*params.BeaconConfig().SlotsPerEpoch)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "could not compute epoch attestations merkleization")
	}
	attsLenBuf := new(bytes.Buffer)
	if err := binary.Write(attsLenBuf, binary.LittleEndian, uint64(len(atts))); err != nil {
		return [32]byte{}, errors.Wrap(err, "could not marshal epoch attestations length")
	}
	// We need to mix in the length of the slice.
	attsLenRoot := make([]byte, 32)
	copy(attsLenRoot, attsLenBuf.Bytes())
	return mixInLength(attsRootsRoot, attsLenRoot), nil
}
