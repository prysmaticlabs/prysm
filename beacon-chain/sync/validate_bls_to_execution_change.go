package sync

import (
	"context"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v3/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"go.opencensus.io/trace"
)

func (s *Service) validateBlsToExecutionChange(ctx context.Context, pid peer.ID, msg *pubsub.Message) (pubsub.ValidationResult, error) {
	// Validation runs on publish (not just subscriptions), so we should approve any message from
	// ourselves.
	if pid == s.cfg.p2p.PeerID() {
		return pubsub.ValidationAccept, nil
	}

	// The head state will be too far away to validate any execution change.
	if s.cfg.initialSync.Syncing() {
		return pubsub.ValidationIgnore, nil
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateBlsToExecutionChange")
	defer span.End()

	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationReject, err
	}

	blsChange, ok := m.(*ethpb.SignedBLSToExecutionChange)
	if !ok {
		return pubsub.ValidationReject, errWrongMessage
	}

	// Check that the validator hasn't submitted a previous execution change.
	if s.cfg.blsToExecPool.ValidatorExists(blsChange.Message.ValidatorIndex) {
		return pubsub.ValidationIgnore, nil
	}
	val, err := s.cfg.chain.HeadValidatorIndex(blsChange.Message.ValidatorIndex)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	// Validate that the execution change object is valid.
	err = blocks.ValidateBLSToExecutionChange(val.WithdrawalCredentials(), blsChange)
	if err != nil {
		return pubsub.ValidationReject, err
	}

	// Validate the signature of the message using our batch gossip verifier.
	gvr := s.cfg.chain.GenesisValidatorsRoot()
	sigBatch, err := blocks.BLSChangesSignatureBatch(ctx, slots.ToEpoch(s.cfg.chain.HeadSlot()), s.cfg.chain.CurrentFork(), gvr[:], []*ethpb.SignedBLSToExecutionChange{blsChange})
	if err != nil {
		return pubsub.ValidationReject, err
	}
	res, err := s.validateWithBatchVerifier(ctx, "bls to execution change", sigBatch)
	if res != pubsub.ValidationAccept {
		return res, err
	}
	msg.ValidatorData = blsChange // Used in downstream subscriber
	return pubsub.ValidationAccept, nil
}
