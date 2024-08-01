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
		fieldRoots = make([][]byte, 11)
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
	a, _ := ssz.BitwiseMerkleize(chunks, uint64(len(chunks)), uint64(len(chunks)))
	fieldRoots[0] = a[:]

	eth1DataRoot, err := stateutil.Eth1Root(blockBody.eth1Data)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute eth1data merkleization")
	}
	fieldRoots[1] = eth1DataRoot[:]

	var b [field_params.RootLength]byte
	copy(b[:], blockBody.graffiti[:])
	fieldRoots[2] = b[:]

	proposerSlashingsRoot, err := ssz.SliceRoot(blockBody.proposerSlashings, field_params.SlashingsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute proposer slashings merkleization")
	}
	fieldRoots[3] = proposerSlashingsRoot[:]

	attesterSlashingsRoot, err := ssz.SliceRoot(blockBody.attesterSlashings, field_params.SlashingsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute attester slashings merkleization")
	}
	fieldRoots[4] = attesterSlashingsRoot[:]

	if blockBody.version == version.Electra {
		attesterSlashingsElectraRoot, err := ssz.SliceRoot(blockBody.attesterSlashingsElectra, field_params.SlashingsLength)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute attester slashings electra merkleization")
		}
		fieldRoots[5] = attesterSlashingsElectraRoot[:]
	}

	attestationsRoot, err := ssz.SliceRoot(blockBody.attestations, field_params.CurrentEpochAttestationsLength)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute attestations merkleization")
	}
	fieldRoots[6] = attestationsRoot[:]

	if blockBody.version == version.Electra {
		attestationsElectraRoot, err := ssz.SliceRoot(blockBody.attestationsElectra, field_params.CurrentEpochAttestationsLength)
		if err != nil {
			return nil, errors.Wrap(err, "could not compute attestations electra merkleization")
		}
		fieldRoots[7] = attestationsElectraRoot[:]
	}

	depositsRoot, err := ssz.SliceRoot(blockBody.deposits, field_params.MaxDepositRequestsPerPayload)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute deposits merkleization")
	}
	fieldRoots[8] = depositsRoot[:]
}
