package sync

import (
	"context"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/verification"
	payloadattestation "github.com/prysmaticlabs/prysm/v5/consensus-types/epbs/payload-attestation"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing/trace"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"google.golang.org/protobuf/proto"
)

var (
	errAlreadySeenPayloadAttestation = errors.New("payload attestation already seen for validator index")
)

func (s *Service) validatePayloadAttestation(ctx context.Context, pid peer.ID, msg *pubsub.Message) (pubsub.ValidationResult, error) {
	if pid == s.cfg.p2p.PeerID() {
		return pubsub.ValidationAccept, nil
	}
	if s.cfg.initialSync.Syncing() {
		return pubsub.ValidationIgnore, nil
	}
	ctx, span := trace.StartSpan(ctx, "sync.validatePayloadAttestation")
	defer span.End()
	if msg.Topic == nil {
		return pubsub.ValidationReject, errInvalidTopic
	}
	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationReject, err
	}
	att, ok := m.(*eth.PayloadAttestationMessage)
	if !ok {
		return pubsub.ValidationReject, errWrongMessage
	}
	pa, err := payloadattestation.NewReadOnly(att)
	if err != nil {
		log.WithError(err).Error("failed to create read only payload attestation")
		return pubsub.ValidationIgnore, err
	}
	v := s.newPayloadAttestationVerifier(pa, verification.GossipPayloadAttestationMessageRequirements)

	if err := v.VerifyCurrentSlot(); err != nil {
		return pubsub.ValidationIgnore, err
	}

	if err := v.VerifyPayloadStatus(); err != nil {
		return pubsub.ValidationReject, err
	}

	if err := v.VerifyBlockRootSeen(s.seenBlockRoot); err != nil {
		return pubsub.ValidationIgnore, err
	}

	if err := v.VerifyBlockRootValid(s.hasBadBlock); err != nil {
		return pubsub.ValidationReject, err
	}

	st, err := s.cfg.chain.HeadState(ctx)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}

	if err := v.VerifyValidatorInPTC(ctx, st); err != nil {
		return pubsub.ValidationReject, err
	}

	if err := v.VerifySignature(st); err != nil {
		return pubsub.ValidationReject, err
	}

	if s.payloadAttestationCache.Seen(pa.BeaconBlockRoot(), uint64(pa.ValidatorIndex())) {
		return pubsub.ValidationIgnore, errAlreadySeenPayloadAttestation
	}

	return pubsub.ValidationAccept, nil
}

func (s *Service) payloadAttestationSubscriber(ctx context.Context, msg proto.Message) error {
	a, ok := msg.(*eth.PayloadAttestationMessage)
	if !ok {
		return errWrongMessage
	}
	return s.cfg.chain.ReceivePayloadAttestationMessage(ctx, a)
}