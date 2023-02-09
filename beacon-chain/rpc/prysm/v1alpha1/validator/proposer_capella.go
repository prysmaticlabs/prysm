package validator

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	consensusblocks "github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/runtime/version"
	"github.com/sirupsen/logrus"
)

// Sets the bls to exec data for a block.
func (vs *Server) setBlsToExecData(blk interfaces.SignedBeaconBlockWriteable, headState state.BeaconState) {
	if blk.Version() < version.Capella {
		return
	}
	if err := blk.SetBLSToExecutionChanges([]*ethpb.SignedBLSToExecutionChange{}); err != nil {
		log.WithError(err).Error("Could not set bls to execution data in block")
		return
	}
	changes, err := vs.BLSChangesPool.BLSToExecChangesForInclusion(headState)
	if err != nil {
		log.WithError(err).Error("Could not get bls to execution changes")
		return
	} else {
		if err := blk.SetBLSToExecutionChanges(changes); err != nil {
			log.WithError(err).Error("Could not set bls to execution changes")
			return
		}
	}
}

func (vs *Server) unblindBuilderBlockCapella(ctx context.Context, b interfaces.SignedBeaconBlock) (interfaces.SignedBeaconBlock, error) {
	if err := consensusblocks.BeaconBlockIsNil(b); err != nil {
		return nil, errors.Wrap(err, "block is nil")
	}

	// No-op if the input block is not version blind and capella.
	if b.Version() != version.Capella || !b.IsBlinded() {
		return b, nil
	}
	// No-op nothing if the builder has not been configured.
	if !vs.BlockBuilder.Configured() {
		return b, nil
	}

	agg, err := b.Block().Body().SyncAggregate()
	if err != nil {
		return nil, errors.Wrap(err, "could not get sync aggregate")
	}
	h, err := b.Block().Body().Execution()
	if err != nil {
		return nil, errors.Wrap(err, "could not get execution header")
	}
	header, ok := h.Proto().(*enginev1.ExecutionPayloadHeaderCapella)
	if !ok {
		return nil, errors.New("execution data must be execution payload header capella")
	}
	parentRoot := b.Block().ParentRoot()
	stateRoot := b.Block().StateRoot()
	randaoReveal := b.Block().Body().RandaoReveal()
	graffiti := b.Block().Body().Graffiti()
	sig := b.Signature()
	sb := &ethpb.SignedBlindedBeaconBlockCapella{
		Block: &ethpb.BlindedBeaconBlockCapella{
			Slot:          b.Block().Slot(),
			ProposerIndex: b.Block().ProposerIndex(),
			ParentRoot:    parentRoot[:],
			StateRoot:     stateRoot[:],
			Body: &ethpb.BlindedBeaconBlockBodyCapella{
				RandaoReveal:           randaoReveal[:],
				Eth1Data:               b.Block().Body().Eth1Data(),
				Graffiti:               graffiti[:],
				ProposerSlashings:      b.Block().Body().ProposerSlashings(),
				AttesterSlashings:      b.Block().Body().AttesterSlashings(),
				Attestations:           b.Block().Body().Attestations(),
				Deposits:               b.Block().Body().Deposits(),
				VoluntaryExits:         b.Block().Body().VoluntaryExits(),
				SyncAggregate:          agg,
				ExecutionPayloadHeader: header,
			},
		},
		Signature: sig[:],
	}

	wrappedSb, err := consensusblocks.NewSignedBeaconBlock(sb)
	if err != nil {
		return nil, errors.Wrap(err, "could not create signed block")
	}

	payload, err := vs.BlockBuilder.SubmitBlindedBlock(ctx, wrappedSb)
	if err != nil {
		return nil, errors.Wrap(err, "could not submit blinded block")
	}

	capellaPayload, err := payload.PbCapella()
	if err != nil {
		return nil, errors.Wrap(err, "could not get payload")
	}
	bb := &ethpb.SignedBeaconBlockCapella{
		Block: &ethpb.BeaconBlockCapella{
			Slot:          sb.Block.Slot,
			ProposerIndex: sb.Block.ProposerIndex,
			ParentRoot:    sb.Block.ParentRoot,
			StateRoot:     sb.Block.StateRoot,
			Body: &ethpb.BeaconBlockBodyCapella{
				RandaoReveal:      sb.Block.Body.RandaoReveal,
				Eth1Data:          sb.Block.Body.Eth1Data,
				Graffiti:          sb.Block.Body.Graffiti,
				ProposerSlashings: sb.Block.Body.ProposerSlashings,
				AttesterSlashings: sb.Block.Body.AttesterSlashings,
				Attestations:      sb.Block.Body.Attestations,
				Deposits:          sb.Block.Body.Deposits,
				VoluntaryExits:    sb.Block.Body.VoluntaryExits,
				SyncAggregate:     agg,
				ExecutionPayload:  capellaPayload,
			},
		},
		Signature: sb.Signature,
	}
	wb, err := consensusblocks.NewSignedBeaconBlock(bb)
	if err != nil {
		return nil, errors.Wrap(err, "could not create signed block")
	}

	txs, err := payload.Transactions()
	if err != nil {
		return nil, errors.Wrap(err, "could not get transactions from payload")
	}
	log.WithFields(logrus.Fields{
		"blockHash":    fmt.Sprintf("%#x", h.BlockHash()),
		"feeRecipient": fmt.Sprintf("%#x", h.FeeRecipient()),
		"gasUsed":      h.GasUsed,
		"slot":         b.Block().Slot(),
		"txs":          len(txs),
	}).Info("Retrieved full capella payload from builder")

	return wb, nil
}
