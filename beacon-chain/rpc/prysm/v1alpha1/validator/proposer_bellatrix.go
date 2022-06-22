package validator

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition/interop"
	"github.com/prysmaticlabs/prysm/config/params"
	coreBlock "github.com/prysmaticlabs/prysm/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
)

func (vs *Server) getBellatrixBeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.BeaconBlockBellatrix, error) {
	altairBlk, err := vs.buildAltairBeaconBlock(ctx, req)
	if err != nil {
		return nil, err
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
	return blk, nil
}

// This function retrieves the payload header given the slot number and the validator index.
// It's a no-op if the latest head block is not versioned bellatrix.
func (vs *Server) getPayloadHeader(ctx context.Context, slot types.Slot, idx types.ValidatorIndex) (*enginev1.ExecutionPayloadHeader, error) {
	if err := vs.BlockBuilder.Status(); err != nil {
		return nil, err
	}
	b, err := vs.HeadFetcher.HeadBlock(ctx)
	if err != nil {
		return nil, err
	}
	if blocks.IsPreBellatrixVersion(b.Version()) {
		return nil, nil
	}
	h, err := b.Block().Body().ExecutionPayload()
	if err != nil {
		return nil, err
	}
	pk, err := vs.HeadFetcher.HeadValidatorIndexToPublicKey(ctx, idx)
	if err != nil {
		return nil, err
	}
	bid, err := vs.BlockBuilder.GetHeader(ctx, slot, bytesutil.ToBytes32(h.BlockHash), pk)
	if err != nil {
		return nil, err
	}
	return bid.Message.Header, nil
}

// This function constructs the builder block given the input altair block and the header. It returns a generic beacon block for signing
func (vs *Server) buildHeaderBlock(ctx context.Context, b *ethpb.BeaconBlockAltair, h *enginev1.ExecutionPayloadHeader) (*ethpb.GenericBeaconBlock, error) {
	if b == nil || b.Body == nil {
		return nil, errors.New("nil block")
	}
	if h == nil {
		return nil, errors.New("nil header")
	}

	blk := &ethpb.BlindedBeaconBlockBellatrix{
		Slot:          b.Slot,
		ProposerIndex: b.ProposerIndex,
		ParentRoot:    b.ParentRoot,
		StateRoot:     params.BeaconConfig().ZeroHash[:],
		Body: &ethpb.BlindedBeaconBlockBodyBellatrix{
			RandaoReveal:           b.Body.RandaoReveal,
			Eth1Data:               b.Body.Eth1Data,
			Graffiti:               b.Body.Graffiti,
			ProposerSlashings:      b.Body.ProposerSlashings,
			AttesterSlashings:      b.Body.AttesterSlashings,
			Attestations:           b.Body.Attestations,
			Deposits:               b.Body.Deposits,
			VoluntaryExits:         b.Body.VoluntaryExits,
			SyncAggregate:          b.Body.SyncAggregate,
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
		return nil, errors.Wrap(err, "could not compute state root")
	}
	blk.StateRoot = stateRoot
	return &ethpb.GenericBeaconBlock{Block: &ethpb.GenericBeaconBlock_BlindedBellatrix{BlindedBellatrix: blk}}, nil
}

// This function retrieves the full payload block using the input blind block. This input must be versioned as
// bellatrix blind block. The output block will contain the full payload. The original header block
// will be returned the block builder is not configured.
func (vs *Server) getBuilderBlock(ctx context.Context, b interfaces.SignedBeaconBlock) (interfaces.SignedBeaconBlock, error) {
	if err := coreBlock.BeaconBlockIsNil(b); err != nil {
		return nil, err
	}

	// No-op if the input block is not version blind and bellatrix.
	if b.Version() != version.BellatrixBlind {
		return b, nil
	}
	// No-op nothing if the builder has not been configured.
	if !vs.BlockBuilder.Configured() {
		return b, nil
	}
	if err := vs.BlockBuilder.Status(); err != nil {
		return nil, err
	}
	agg, err := b.Block().Body().SyncAggregate()
	if err != nil {
		return nil, err
	}
	h, err := b.Block().Body().ExecutionPayloadHeader()
	if err != nil {
		return nil, err
	}
	sb := &ethpb.SignedBlindedBeaconBlockBellatrix{
		Block: &ethpb.BlindedBeaconBlockBellatrix{
			Slot:          b.Block().Slot(),
			ProposerIndex: b.Block().ProposerIndex(),
			ParentRoot:    b.Block().ParentRoot(),
			StateRoot:     b.Block().StateRoot(),
			Body: &ethpb.BlindedBeaconBlockBodyBellatrix{
				RandaoReveal:           b.Block().Body().RandaoReveal(),
				Eth1Data:               b.Block().Body().Eth1Data(),
				Graffiti:               b.Block().Body().Graffiti(),
				ProposerSlashings:      b.Block().Body().ProposerSlashings(),
				AttesterSlashings:      b.Block().Body().AttesterSlashings(),
				Attestations:           b.Block().Body().Attestations(),
				Deposits:               b.Block().Body().Deposits(),
				VoluntaryExits:         b.Block().Body().VoluntaryExits(),
				SyncAggregate:          agg,
				ExecutionPayloadHeader: h,
			},
		},
		Signature: b.Signature(),
	}

	payload, err := vs.BlockBuilder.SubmitBlindedBlock(ctx, sb)
	if err != nil {
		return nil, err
	}
	bb := &ethpb.SignedBeaconBlockBellatrix{
		Block: &ethpb.BeaconBlockBellatrix{
			Slot:          sb.Block.Slot,
			ProposerIndex: sb.Block.ProposerIndex,
			ParentRoot:    sb.Block.ParentRoot,
			StateRoot:     sb.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyBellatrix{
				RandaoReveal:      sb.Block.Body.RandaoReveal,
				Eth1Data:          sb.Block.Body.Eth1Data,
				Graffiti:          sb.Block.Body.Graffiti,
				ProposerSlashings: sb.Block.Body.ProposerSlashings,
				AttesterSlashings: sb.Block.Body.AttesterSlashings,
				Attestations:      sb.Block.Body.Attestations,
				Deposits:          sb.Block.Body.Deposits,
				VoluntaryExits:    sb.Block.Body.VoluntaryExits,
				SyncAggregate:     agg,
				ExecutionPayload:  payload,
			},
		},
		Signature: sb.Signature,
	}
	wb, err := wrapper.WrappedSignedBeaconBlock(bb)
	if err != nil {
		return nil, err
	}
	return wb, nil
}
