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

func (vs *Server) getBellatrixBeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BeaconBlockBellatrix, error) {
	altairBlk, err := vs.buildAltairBeaconBlock(ctx, req)
	if err != nil {
		return nil, err
	}

	payload, err := vs.getExecutionPayload(ctx, req.Slot)
	if err != nil {
		return nil, err
	}

	log.WithFields(logrus.Fields{
		"hash": fmt.Sprintf("%#x", payload.BlockHash),
		"parentHash": fmt.Sprintf("%#x", payload.ParentHash),
		"number": payload.BlockNumber,
		"txCount": len(payload.Transactions),
	}).Info("Received payload")

	blk := &ethpb.BeaconBlockBellatrix{
		Slot:          altairBlk.Slot,
		ProposerIndex: altairBlk.ProposerIndex,
		ParentRoot:    altairBlk.ParentRoot,
		StateRoot:     params.BeaconConfig().ZeroHash[:],
		Body: &ethpb.BeaconBlockBodyBellatrix{
			RandaoReveal:      altairBlk.Body.RandaoReveal,
			Eth1Data:          altairBlk.Body.Eth1Data,
			Graffiti:          altairBlk.Body.Graffiti,
			ProposerSlashings: altairBlk.Body.ProposerSlashings,
			AttesterSlashings: altairBlk.Body.AttesterSlashings,
			Attestations:      altairBlk.Body.Attestations,
			Deposits:          altairBlk.Body.Deposits,
			VoluntaryExits:    altairBlk.Body.VoluntaryExits,
			SyncAggregate:     altairBlk.Body.SyncAggregate,
			ExecutionPayload:  payload,
		},
	}
	// Compute state root with the newly constructed block.
	wsb, err := wrapper.WrappedBellatrixSignedBeaconBlock(
		&ethpb.SignedBeaconBlockBellatrix{Block: blk, Signature: make([]byte, 96)},
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
