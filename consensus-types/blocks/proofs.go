package blocks

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/state/stateutil"
	field_params "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/encoding/ssz"
	"github.com/prysmaticlabs/prysm/v5/runtime/version"
	"go.opencensus.io/trace"
)

func ComputeFieldRootsForBlockBody(ctx context.Context, blockBody *BeaconBlockBody) ([][]byte, error) {
	ctx, span := trace.StartSpan(ctx, "ComputeFieldRootsForBlockBody")
	defer span.End()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

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

	chunks, err := ssz.PackByChunk([][]byte{blockBody.randaoReveal[:]})
	if err != nil {
		return nil, errors.Wrap(err, "could not pack randao reveal into chunks")
	}
	var a [32]byte
	a, err = ssz.BitwiseMerkleize(chunks, uint64(len(chunks)), uint64(len(chunks)))
	if err != nil {
		return nil, errors.Wrap(err, "could not compute randao reveal merkleization")
	}
	fieldRoots[0] = a[:]

	eth1DataRoot, err := stateutil.Eth1Root(blockBody.eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[1] = eth1DataRoot[:]

	var b [field_params.RootLength]byte
	copy(b[:], blockBody.graffiti[:])
	fieldRoots[2] = b[:]

	proposerSlashingsRoot, err := ssz.SliceRoot(blockBody.proposerSlashings, field_params.MaxProposerSlashings)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute proposer slashings merkleization")
	}
	fieldRoots[3] = proposerSlashingsRoot[:]

	if blockBody.version < version.Electra {
		attesterSlashingsRoot, err := ssz.SliceRoot(blockBody.attesterSlashings, field_params.MaxAttesterSlashings)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute attester slashings merkleization")
		}
		fieldRoots[4] = attesterSlashingsRoot[:]
	} else {
		attesterSlashingsElectraRoot, err := ssz.SliceRoot(blockBody.attesterSlashingsElectra, field_params.MaxAttesterSlashingsElectra)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute attester slashings electra merkleization")
		}
		fieldRoots[4] = attesterSlashingsElectraRoot[:]
	}

	if blockBody.version < version.Electra {
		attestationsRoot, err := ssz.SliceRoot(blockBody.attestations, field_params.MaxAttestations)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute attestations merkleization")
		}
		fieldRoots[5] = attestationsRoot[:]
	} else {
		attestationsElectraRoot, err := ssz.SliceRoot(blockBody.attestationsElectra, field_params.MaxAttestationsElectra)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute attestations electra merkleization")
		}
		fieldRoots[5] = attestationsElectraRoot[:]
	}

	depositsRoot, err := ssz.SliceRoot(blockBody.deposits, field_params.MaxDeposits)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute deposits merkleization")
	}
	fieldRoots[6] = depositsRoot[:]

	voluntaryExitsRoot, err := ssz.SliceRoot(blockBody.voluntaryExits, field_params.MaxVoluntaryExits)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute voluntary exits merkleization")
	}
	fieldRoots[7] = voluntaryExitsRoot[:]
}
