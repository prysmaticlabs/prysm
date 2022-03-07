package validator

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition/interop"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
)

func (vs *Server) getEip4844BeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BeaconBlockWithBlobKZGs, error) {
	bellatrixBlk, err := vs.getBellatrixBeaconBlock(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "could not get bellatrix block")
	}

	blobs, err := vs.ExecutionEngineCaller.GetBlobs(ctx, [8]byte{})
	if err != nil {
		return nil, errors.Wrap(err, "could not get blobs")
	}

	// TODO: Save blobs, broadcast, and convert to KZGs

	kzgs := make([][]byte, len(blobs))
	for i := range blobs {
		kzgs[i] = bytesutil.PadTo([]byte{}, 48)
	}
	blk := &ethpb.BeaconBlockWithBlobKZGs{
		Slot:          bellatrixBlk.Slot,
		ProposerIndex: bellatrixBlk.ProposerIndex,
		ParentRoot:    bellatrixBlk.ParentRoot,
		StateRoot:     params.BeaconConfig().ZeroHash[:],
		Body: &ethpb.BeaconBlockBodyWithBlobKZGs{
			RandaoReveal:      bellatrixBlk.Body.RandaoReveal,
			Eth1Data:          bellatrixBlk.Body.Eth1Data,
			Graffiti:          bellatrixBlk.Body.Graffiti,
			ProposerSlashings: bellatrixBlk.Body.ProposerSlashings,
			AttesterSlashings: bellatrixBlk.Body.AttesterSlashings,
			Attestations:      bellatrixBlk.Body.Attestations,
			Deposits:          bellatrixBlk.Body.Deposits,
			VoluntaryExits:    bellatrixBlk.Body.VoluntaryExits,
			SyncAggregate:     bellatrixBlk.Body.SyncAggregate,
			ExecutionPayload:  bellatrixBlk.Body.ExecutionPayload,
			BlobKzgs:          kzgs, // TODO: Add blob KZGs here.,
		},
	}
	// Compute state root with the newly constructed block.
	wsb, err := wrapper.WrappedSignedBeaconBlock(
		&ethpb.SignedBeaconBlockWithBlobKZGs{
			Block:     blk,
			Signature: make([]byte, 96),
		},
	)
	if err != nil {
		return nil, err
	}
	stateRoot, err := vs.computeStateRoot(ctx, wsb)
	if err != nil {
		interop.WriteBlockToDisk(wsb, true /*failed*/)
		return nil, fmt.Errorf("could not compute state root: %v", err)
	}
	blk.StateRoot = stateRoot

	return blk, nil
}
