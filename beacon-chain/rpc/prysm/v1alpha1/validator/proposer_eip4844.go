package validator

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition/interop"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
)

func (vs *Server) getEip4844BeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BeaconBlockWithBlobKZGs, *ethpb.BlobsSidecar, error) {
	generic, payloadID, err := vs.buildBellatrixBeaconBlock(ctx, req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get bellatrix block")
	}

	block := generic.GetBellatrix()

	blobsBundle, err := vs.ExecutionEngineCaller.GetBlobsBundle(ctx, payloadID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get blobs")
	}

	payload, _, err := vs.getExecutionPayload4844(
		ctx,
		req.Slot,
		block.ProposerIndex,
		bytesutil.ToBytes32(block.ParentRoot),
	)
	if err != nil {
		return nil, nil, err
	}

	// sanity check the blobs bundle
	if !bytes.Equal(blobsBundle.BlockHash, payload.BlockHash) {
		return nil, nil, errors.New("invalid blobs bundle received")
	}

	if len(blobsBundle.Blobs) != len(blobsBundle.Kzgs) {
		return nil, nil, errors.New("mismatched blobs and kzgs length")
	}

	var (
		kzgs  [][]byte
		blobs []*enginev1.Blob
	)
	if len(blobsBundle.Kzgs) != 0 {
		kzgs = blobsBundle.Kzgs
		blobs = blobsBundle.Blobs
	}


	blk := &ethpb.BeaconBlockWithBlobKZGs{
		Slot:          block.Slot,
		ProposerIndex: block.ProposerIndex,
		ParentRoot:    block.ParentRoot,
		StateRoot:     params.BeaconConfig().ZeroHash[:],
		Body: &ethpb.BeaconBlockBodyWithBlobKZGs{
			RandaoReveal:      block.Body.RandaoReveal,
			Eth1Data:          block.Body.Eth1Data,
			Graffiti:          block.Body.Graffiti,
			ProposerSlashings: block.Body.ProposerSlashings,
			AttesterSlashings: block.Body.AttesterSlashings,
			Attestations:      block.Body.Attestations,
			Deposits:          block.Body.Deposits,
			VoluntaryExits:    block.Body.VoluntaryExits,
			SyncAggregate:     block.Body.SyncAggregate,
			ExecutionPayload:  payload,
			BlobKzgs:          kzgs,
		},
	}

	// Compute state root with the newly constructed block.
	wsb, err := consensusblocks.NewSignedBeaconBlock(
		&ethpb.SignedBeaconBlockWithBlobKZGs{
			Block:     blk,
			Signature: make([]byte, 96),
		},
	)
	if err != nil {
		return nil, nil, err
	}
	stateRoot, err := vs.computeStateRoot(ctx, wsb)
	if err != nil {
		interop.WriteBlockToDisk(wsb, true /*failed*/)
		return nil, nil, fmt.Errorf("could not compute state root: %v", err)
	}
	blk.StateRoot = stateRoot
	r, err := blk.HashTreeRoot()
	if err != nil {
		return nil, nil, err
	}

	var sideCar *ethpb.BlobsSidecar
	if len(blobs) != 0 {
		sideCar = &ethpb.BlobsSidecar{
			BeaconBlockRoot: r[:],
			BeaconBlockSlot: blk.Slot,
			Blobs:           blobs,
			AggregatedProof: blobsBundle.AggregatedProof,
		}
	}

	return blk, sideCar, nil
}
