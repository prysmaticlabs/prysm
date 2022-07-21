package validator

import (
	"bytes"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition/interop"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

func (vs *Server) getEip4844BeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BeaconBlockWithBlobKZGs, *ethpb.BlobsSidecar, error) {
	bellatrixBlk, payloadID, err := vs.buildBellatrixBeaconBlock(ctx, req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get bellatrix block")
	}

	blobsBundle, err := vs.ExecutionEngineCaller.GetBlobsBundle(ctx, payloadID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not get blobs")
	}
	// sanity check the blobs bundle
	if bytes.Compare(blobsBundle.BlockHash, bellatrixBlk.Body.ExecutionPayload.BlockHash) != 0 {
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
			BlobKzgs:          kzgs,
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
