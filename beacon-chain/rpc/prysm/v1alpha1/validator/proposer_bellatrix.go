package validator

import (
	"context"
	"fmt"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition/interop"
	"github.com/prysmaticlabs/prysm/config/params"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

func (vs *Server) getBellatrixBeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.GenericBeaconBlock, error) {
	altairBlk, err := vs.buildAltairBeaconBlock(ctx, req)
	if err != nil {
		return nil, err
	}

	h, exists, err := vs.getBuilderHeader(ctx, req.Slot, altairBlk.ProposerIndex)
	if err != nil {
		return nil, err
	}
	if exists {
		blk := &ethpb.BlindedBeaconBlockBellatrix{
			Slot:          altairBlk.Slot,
			ProposerIndex: altairBlk.ProposerIndex,
			ParentRoot:    altairBlk.ParentRoot,
			StateRoot:     params.BeaconConfig().ZeroHash[:],
			Body: &ethpb.BlindedBeaconBlockBodyBellatrix{
				RandaoReveal:           altairBlk.Body.RandaoReveal,
				Eth1Data:               altairBlk.Body.Eth1Data,
				Graffiti:               altairBlk.Body.Graffiti,
				ProposerSlashings:      altairBlk.Body.ProposerSlashings,
				AttesterSlashings:      altairBlk.Body.AttesterSlashings,
				Attestations:           altairBlk.Body.Attestations,
				Deposits:               altairBlk.Body.Deposits,
				VoluntaryExits:         altairBlk.Body.VoluntaryExits,
				SyncAggregate:          altairBlk.Body.SyncAggregate,
				ExecutionPayloadHeader: h,
			},
		}
		wsb, err := wrapper.WrappedSignedBeaconBlock(
			&ethpb.SignedBlindedBeaconBlockBellatrix{Block: blk, Signature: make([]byte, 96)},
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
		return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: blk}}, nil
	}

	payload, err := vs.getExecutionPayload(ctx, req.Slot, altairBlk.ProposerIndex)
	if err != nil {
		return nil, err
	}

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
	wsb, err := wrapper.WrappedSignedBeaconBlock(
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
	return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_Bellatrix{Bellatrix: blk}}, nil
}

func (vs *Server) getBuilderHeader(ctx context.Context, slot types.Slot, idx types.ValidatorIndex) (*ethpb.ExecutionPayloadHeader, bool, error) {
	if vs.BlockBuilder.Status() != nil {
		log.WithError(vs.BlockBuilder.Status()).Error("Could not get builder status")
		return nil, false, nil
	}
	b, err := vs.HeadFetcher.HeadBlock(ctx)
	if err != nil {
		return nil, false, err
	}
	if blocks.IsPreBellatrixVersion(b.Version()) {
		return nil, false, nil
	}
	h, err := b.Block().Body().ExecutionPayload()
	if err != nil {
		return nil, false, err
	}
	pk, err := vs.HeadFetcher.HeadValidatorIndexToPublicKey(ctx, idx)
	if err != nil {
		return nil, false, err
	}
	bid, err := vs.BlockBuilder.GetHeader(ctx, slot, bytesutil.ToBytes32(h.BlockHash), pk)
	if err != nil {
		log.WithError(err).Error("Could not get builder header")
		return nil, false, nil
	}
	return bid.Message.Header, true, nil
}
