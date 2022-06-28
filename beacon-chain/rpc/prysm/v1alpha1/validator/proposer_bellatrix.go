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
	"github.com/sirupsen/logrus"
)

func (vs *Server) getBellatrixBeaconBlock(ctx context.Context, req *ethpb.BlockRequest) (*ethpb.GenericBeaconBlock, error) {
	altairBlk, err := vs.buildAltairBeaconBlock(ctx, req)
	if err != nil {
		return nil, err
	}

	log.WithFields(logrus.Fields{
		"NilStatus":  vs.BlockBuilder != nil,
		"Configured": vs.BlockBuilder.Configured(),
	}).Info("Checking builder status")

	// Did the user specify block builder
	if vs.BlockBuilder != nil && vs.BlockBuilder.Configured() {
		// Does the protocol allow node to use block builder to construct payload header now
		ready, err := vs.readyForBuilder(ctx)
		if err == nil && ready {
			log.Info("Builder is ready")
			// Retrieve header from block builder and construct the head block if there's no error.
			h, err := vs.getPayloadHeader(ctx, req.Slot, altairBlk.ProposerIndex)
			if err == nil {
				log.WithFields(logrus.Fields{
					"BlockHash":    fmt.Sprintf("%#x", h.BlockHash),
					"TxRoot":       fmt.Sprintf("%#x", h.TransactionsRoot),
					"FeeRecipient": fmt.Sprintf("%#x", h.FeeRecipient),
					"GasUsed":      h.GasUsed,
					"GasLimit":     h.GasLimit,
				}).Info("Retrieved payload header from builder")
				return vs.buildHeaderBlock(ctx, altairBlk, h)
			}
			// If there's an error at retrieving header, default back to using local EE.
			log.WithError(err).Warning("Could not construct block using the header from builders, using local execution engine instead")
		}
		if err != nil {
			log.WithError(err).Warning("Could not decide builder is ready")
		}
		if !ready {
			log.Debug("Can't use builder yet. Using local execution engine to propose")
		}
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

// readyForBuilder returns true if builder is allowed to be used. Builder is allowed to be use after the
// first finalized checkpt has been execution-enabled.
func (vs *Server) readyForBuilder(ctx context.Context) (bool, error) {
	cp := vs.FinalizationFetcher.FinalizedCheckpt()
	if bytesutil.ToBytes32(cp.Root) == params.BeaconConfig().ZeroHash {
		return false, nil
	}
	b, err := vs.BeaconDB.Block(ctx, bytesutil.ToBytes32(cp.Root))
	if err != nil {
		return false, err
	}
	if err := coreBlock.BeaconBlockIsNil(b); err != nil {
		return false, err
	}
	return blocks.IsExecutionBlock(b.Block().Body())
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

	log.WithFields(logrus.Fields{
		"BlockHash":    fmt.Sprintf("%#x", h.BlockHash),
		"TxRoot":       fmt.Sprintf("%#x", h.TransactionsRoot),
		"FeeRecipient": fmt.Sprintf("%#x", h.FeeRecipient),
		"GasUsed":      h.GasUsed,
		"GasLimit":     h.GasLimit,
	}).Info("Submitting blind block")

	payload, err := vs.BlockBuilder.SubmitBlindedBlock(ctx, sb)
	if err != nil {
		return nil, errors.Wrap(err, "could not submit blind block")
	}

	log.WithFields(logrus.Fields{
		"BlockHash":    fmt.Sprintf("%#x", payload.BlockHash),
		"Txs":          len(payload.Transactions),
		"FeeRecipient": fmt.Sprintf("%#x", payload.FeeRecipient),
		"GasUsed":      payload.GasUsed,
		"GasLimit":     payload.GasLimit,
	}).Info("Retrieved payload")

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
