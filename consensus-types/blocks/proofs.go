package blocks

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/crypto/hash/htr"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"go.opencensus.io/trace"
)

func ComputeBlockBodyFieldRoots(ctx context.Context, blockBody *BeaconBlockBody) ([][]byte, error) {
	_, span := trace.StartSpan(ctx, "blocks.ComputeBlockBodyFieldRoots")
	defer span.End()

	if blockBody == nil {
		return nil, errNilBlockBody
	}

	var fieldRoots [][]byte
	switch blockBody.version {
	case version.Phase0:
		fieldRoots = make([][]byte, 8)
	case version.Altair:
		fieldRoots = make([][]byte, 9)
	case version.Bellatrix:
		fieldRoots = make([][]byte, 10)
	case version.Capella:
		fieldRoots = make([][]byte, 11)
	case version.Deneb:
		fieldRoots = make([][]byte, 12)
	case version.Electra:
		fieldRoots = make([][]byte, 12)
	default:
		return nil, fmt.Errorf("unknown block body version %s", version.String(blockBody.version))
	}

	for i := range fieldRoots {
		fieldRoots[i] = make([]byte, 32)
	}

	// Randao Reveal
	randao := blockBody.RandaoReveal()
	root, err := ssz.MerkleizeByteSliceSSZ(randao[:])
	if err != nil {
		return nil, err
	}
	copy(fieldRoots[0], root[:])

	// eth1_data
	eth1 := blockBody.Eth1Data()
	root, err = eth1.HashTreeRoot()
	if err != nil {
		return nil, err
	}
	copy(fieldRoots[1], root[:])

	// graffiti
	root = blockBody.Graffiti()
	copy(fieldRoots[2], root[:])

	// Proposer slashings
	ps := blockBody.ProposerSlashings()
	root, err = ssz.MerkleizeListSSZ(ps, params.BeaconConfig().MaxProposerSlashings)
	if err != nil {
		return nil, err
	}
	copy(fieldRoots[3], root[:])

	// Attester slashings
	as := blockBody.AttesterSlashings()
	bodyVersion := blockBody.Version()
	if bodyVersion < version.Electra {
		root, err = ssz.MerkleizeListSSZ(as, params.BeaconConfig().MaxAttesterSlashings)
	} else {
		root, err = ssz.MerkleizeListSSZ(as, params.BeaconConfig().MaxAttesterSlashingsElectra)
	}
	if err != nil {
		return nil, err
	}
	copy(fieldRoots[4], root[:])

	// Attestations
	att := blockBody.Attestations()
	if bodyVersion < version.Electra {
		root, err = ssz.MerkleizeListSSZ(att, params.BeaconConfig().MaxAttestations)
	} else {
		root, err = ssz.MerkleizeListSSZ(att, params.BeaconConfig().MaxAttestationsElectra)
	}
	if err != nil {
		return nil, err
	}
	copy(fieldRoots[5], root[:])

	// Deposits
	dep := blockBody.Deposits()
	root, err = ssz.MerkleizeListSSZ(dep, params.BeaconConfig().MaxDeposits)
	if err != nil {
		return nil, err
	}
	copy(fieldRoots[6], root[:])

	// Voluntary Exits
	ve := blockBody.VoluntaryExits()
	root, err = ssz.MerkleizeListSSZ(ve, params.BeaconConfig().MaxVoluntaryExits)
	if err != nil {
		return nil, err
	}
	copy(fieldRoots[7], root[:])

	if blockBody.version >= version.Altair {
		// Sync Aggregate
		sa, err := blockBody.SyncAggregate()
		if err != nil {
			return nil, err
		}
		root, err = sa.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		copy(fieldRoots[8], root[:])
	}

	if blockBody.version >= version.Bellatrix {
		// Execution Payload
		ep, err := blockBody.Execution()
		if err != nil {
			return nil, err
		}
		root, err = ep.HashTreeRoot()
		if err != nil {
			return nil, err
		}
		copy(fieldRoots[9], root[:])
	}

	if blockBody.version >= version.Capella {
		// BLS Changes
		bls, err := blockBody.BLSToExecutionChanges()
		if err != nil {
			return nil, err
		}
		root, err = ssz.MerkleizeListSSZ(bls, params.BeaconConfig().MaxBlsToExecutionChanges)
		if err != nil {
			return nil, err
		}
		copy(fieldRoots[10], root[:])
	}

	if blockBody.version >= version.Deneb {
		// KZG commitments
		roots := make([][32]byte, len(blockBody.blobKzgCommitments))
		for i, commitment := range blockBody.blobKzgCommitments {
			chunks, err := ssz.PackByChunk([][]byte{commitment})
			if err != nil {
				return nil, err
			}
			roots[i] = htr.VectorizedSha256(chunks)[0]
		}
		commitmentsRoot, err := ssz.BitwiseMerkleize(roots, uint64(len(roots)), 4096)
		if err != nil {
			return nil, err
		}
		length := make([]byte, 32)
		binary.LittleEndian.PutUint64(length[:8], uint64(len(roots)))
		root = ssz.MixInLength(commitmentsRoot, length)
		copy(fieldRoots[11], root[:])
	}

	return fieldRoots, nil
}
