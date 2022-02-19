package validator

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition/interop"
	"github.com/prysmaticlabs/prysm/config/params"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
	"github.com/sirupsen/logrus"
)

func (vs *Server) getShanghaiBeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BeaconBlockAndBlobs, error) {
	bellatrixBlk, err := vs.getBellatrixBeaconBlock(ctx, req)
	if err != nil {
		return nil, err
	}
	payload, err := vs.getExecutionPayload(ctx, req.Slot)
	if err != nil {
		return nil, err
	}

	log.WithFields(logrus.Fields{
		"hash":       fmt.Sprintf("%#x", payload.BlockHash),
		"parentHash": fmt.Sprintf("%#x", payload.ParentHash),
		"number":     payload.BlockNumber,
		"txCount":    len(payload.Transactions),
	}).Info("Received payload")

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
			BlobKzgs:          nil, // TODO: Add blob KZGs here.
		},
	}
	// Compute state root with the newly constructed block.
	wsb, err := wrapper.WrappedSignedBeaconBlock(
		&ethpb.SignedBeaconBlockAndBlobs{
			Block: &ethpb.SignedBeaconBlockWithBlobKZGs{
				Block:     blk,
				Signature: make([]byte, 96),
			},
			Blobs: nil, // TODO: Add blobs.
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
	blockWithBlobs := &ethpb.BeaconBlockAndBlobs{
		Block: blk,
		Blobs: nil, // TODO: Add blobs here.
	}
	return blockWithBlobs, nil
}
