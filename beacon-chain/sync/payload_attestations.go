package sync

import (
	"context"
	"slices"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v5/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v5/monitoring/tracing"
	eth "github.com/prysmaticlabs/prysm/v5/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
	"google.golang.org/protobuf/proto"
)

var (
	errInvalidPayloadStatus          = errors.New("invalid PayloadStatus")
	errInvalidBeaconBlockRoot        = errors.New("invalid BeaconBlockRoot")
	errInvalidValidatorIndex         = errors.New("invalid validator index")
	errUnkownBeaconBlockRoot         = errors.New("unkonwn beacon block")
	errAlreadySeenPayloadAttestation = errors.New("payload attestation already seen for validator index")
	errNotInPTC                      = errors.New("validator index not in Payload Timeliness Committee")
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
	if err := helpers.ValidateNilPayloadAttestationMessage(att); err != nil {
		return pubsub.ValidationReject, err
	}
	if att.Data.Slot != s.cfg.clock.CurrentSlot() {
		return pubsub.ValidationIgnore, nil
	}
	if att.Data.PayloadStatus >= primitives.PAYLOAD_INVALID_STATUS {
		return pubsub.ValidationReject, errInvalidPayloadStatus
	}
	root := [32]byte(att.Data.BeaconBlockRoot)
	if s.hasBadBlock(root) {
		return pubsub.ValidationReject, errInvalidBeaconBlockRoot
	}
	if !s.cfg.chain.InForkchoice(root) {
		return pubsub.ValidationIgnore, errUnkownBeaconBlockRoot
	}
	st, err := s.cfg.chain.HeadState(ctx)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	if err := helpers.ValidatePayloadAttestationMessageSignature(ctx, st, att); err != nil {
		return pubsub.ValidationReject, err
	}
	ptc, err := helpers.GetPayloadTimelinessCommittee(ctx, st, att.Data.Slot)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	idx := slices.Index(ptc, att.ValidatorIndex)
	if idx == -1 {
		return pubsub.ValidationReject, errNotInPTC
	}
	if s.payloadAttestationCache.Seen(root, uint64(idx)) {
		return pubsub.ValidationIgnore, errAlreadySeenPayloadAttestation
	}
	return pubsub.ValidationAccept, nil
}

func (s *Service) payloadAttestationSubscriber(ctx context.Context, msg proto.Message) error {
	a, ok := msg.(*eth.PayloadAttestationMessage)
	if !ok {
		return errWrongMessage
	}
	if err := helpers.ValidateNilPayloadAttestationMessage(a); err != nil {
		return err
	}
	root := [32]byte(a.Data.BeaconBlockRoot)
	st, err := s.cfg.chain.HeadState(ctx)
	if err != nil {
		return err
	}
	ptc, err := helpers.GetPayloadTimelinessCommittee(ctx, st, a.Data.Slot)
	if err != nil {
		return err
	}
	idx := slices.Index(ptc, a.ValidatorIndex)
	if idx == -1 {
		return errInvalidValidatorIndex
	}
	if s.payloadAttestationCache.Seen(root, uint64(idx)) {
		return nil
	}

	return s.payloadAttestationCache.Add(a, uint64(idx))
}
