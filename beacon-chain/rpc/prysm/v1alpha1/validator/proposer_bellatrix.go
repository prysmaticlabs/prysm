package validator

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition/interop"
	"github.com/prysmaticlabs/prysm/beacon-chain/db/kv"
	"github.com/prysmaticlabs/prysm/config/params"
	coreBlock "github.com/prysmaticlabs/prysm/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/consensus-types/wrapper"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/runtime/version"
	"github.com/sirupsen/logrus"
)

// blockBuilderTimeout is the maximum amount of time allowed for a block builder to respond to a
// block request. This value is known as `BUILDER_PROPOSAL_DELAY_TOLERANCE` in builder spec.
const blockBuilderTimeout = 1 * time.Second

func (vs *Server) getBellatrixBeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.GenericBeaconBlock, error) {
	altairBlk, err := vs.buildAltairBeaconBlock(ctx, req)
	if err != nil {
		return nil, err
	}

	registered, err := vs.validatorRegistered(ctx, altairBlk.ProposerIndex)
	if registered && err == nil {
		builderReady, b, err := vs.getAndBuildHeaderBlock(ctx, altairBlk)
		if err != nil {
			// In the event of an error, the node should fall back to default execution engine for building block.
			log.WithError(err).Error("Failed to build a block from external builder, falling " +
				"back to local execution client")
		} else if builderReady {
			return b, nil
		}
	} else if err != nil {
		log.WithFields(logrus.Fields{
			"slot":           req.Slot,
			"validatorIndex": altairBlk.ProposerIndex,
		}).Errorf("Could not determine validator has registered. Default to local execution client: %v", err)
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

// This function retrieves the payload header given the slot number and the validator index.
// It's a no-op if the latest head block is not versioned bellatrix.
func (vs *Server) getPayloadHeader(ctx context.Context, slot types.Slot, idx types.ValidatorIndex) (*enginev1.ExecutionPayloadHeader, error) {
	b, err := vs.HeadFetcher.HeadBlock(ctx)
	if err != nil {
		return nil, err
	}
	if blocks.IsPreBellatrixVersion(b.Version()) {
		return nil, nil
	}
	h, err := b.Block().Body().Execution()
	if err != nil {
		return nil, err
	}
	pk, err := vs.HeadFetcher.HeadValidatorIndexToPublicKey(ctx, idx)
	if err != nil {
		return nil, err
	}
	bid, err := vs.BlockBuilder.GetHeader(ctx, slot, bytesutil.ToBytes32(h.BlockHash()), pk)
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
func (vs *Server) unblindBuilderBlock(ctx context.Context, b interfaces.SignedBeaconBlock) (interfaces.SignedBeaconBlock, error) {
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

	agg, err := b.Block().Body().SyncAggregate()
	if err != nil {
		return nil, err
	}
	h, err := b.Block().Body().Execution()
	if err != nil {
		return nil, err
	}
	header, ok := h.Proto().(*enginev1.ExecutionPayloadHeader)
	if !ok {
		return nil, errors.New("execution data must be execution payload header")
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
				ExecutionPayloadHeader: header,
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

	log.WithFields(logrus.Fields{
		"blockHash":    fmt.Sprintf("%#x", h.BlockHash()),
		"feeRecipient": fmt.Sprintf("%#x", h.FeeRecipient()),
		"gasUsed":      h.GasUsed,
		"slot":         b.Block().Slot(),
		"txs":          len(payload.Transactions),
	}).Info("Retrieved full payload from builder")

	return wb, nil
}

// readyForBuilder returns true if builder is allowed to be used. Builder is only allowed to be use after the
// first finalized checkpt has been execution-enabled.
func (vs *Server) readyForBuilder(ctx context.Context) (bool, error) {
	cp := vs.FinalizationFetcher.FinalizedCheckpt()
	// Checkpoint root is zero means we are still at genesis epoch.
	if bytesutil.ToBytes32(cp.Root) == params.BeaconConfig().ZeroHash {
		return false, nil
	}
	b, err := vs.BeaconDB.Block(ctx, bytesutil.ToBytes32(cp.Root))
	if err != nil {
		return false, err
	}
	if err = coreBlock.BeaconBlockIsNil(b); err != nil {
		return false, err
	}
	return blocks.IsExecutionBlock(b.Block().Body())
}

// Get and builder header block. Returns a boolean status, built block and error.
// If the status is false that means builder the header block is disallowed.
// This routine is time limited by `blockBuilderTimeout`.
func (vs *Server) getAndBuildHeaderBlock(ctx context.Context, b *ethpb.BeaconBlockAltair) (bool, *ethpb.GenericBeaconBlock, error) {
	// No op. Builder is not defined. User did not specify a user URL. We should use local EE.
	if vs.BlockBuilder == nil || !vs.BlockBuilder.Configured() {
		return false, nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, blockBuilderTimeout)
	defer cancel()
	// Does the protocol allow for builder at this current moment. Builder is only allowed post merge after finalization.
	ready, err := vs.readyForBuilder(ctx)
	if err != nil {
		return false, nil, errors.Wrap(err, "could not determine if builder is ready")
	}
	if !ready {
		return false, nil, nil
	}
	h, err := vs.getPayloadHeader(ctx, b.Slot, b.ProposerIndex)
	if err != nil {
		return false, nil, errors.Wrap(err, "could not get payload header")
	}
	log.WithFields(logrus.Fields{
		"blockHash":    fmt.Sprintf("%#x", h.BlockHash),
		"feeRecipient": fmt.Sprintf("%#x", h.FeeRecipient),
		"gasUsed":      h.GasUsed,
		"slot":         b.Slot,
	}).Info("Retrieved header from builder")
	gb, err := vs.buildHeaderBlock(ctx, b, h)
	if err != nil {
		return false, nil, errors.Wrap(err, "could not combine altair block with payload header")
	}
	return true, gb, nil
}

// validatorRegistered returns true if validator with index `id` was previously registered in the database.
func (vs *Server) validatorRegistered(ctx context.Context, id types.ValidatorIndex) (bool, error) {
	if vs.BeaconDB == nil {
		return false, errors.New("nil beacon db")
	}
	_, err := vs.BeaconDB.RegistrationByValidatorID(ctx, id)
	switch {
	case errors.Is(err, kv.ErrNotFoundFeeRecipient):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}
