package sync

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

var (
	errWrongMessage = errors.New("wrong pubsub message")
	errNilMessage   = errors.New("nil pubsub message")
)

type validationFn func(ctx context.Context) pubsub.ValidationResult

func (s *Service) validateSyncCommitteeMessage(
	ctx context.Context, pid peer.ID, msg *pubsub.Message,
) pubsub.ValidationResult {
	ctx, span := trace.StartSpan(ctx, "sync.validateSyncCommitteeMessage")
	defer span.End()

	if pid == s.cfg.P2P.PeerID() {
		return pubsub.ValidationAccept
	}

	// Basic validations for the topic.
	if result := withValidationPipeline(
		ctx,
		s.ifSyncing(pubsub.ValidationIgnore),
		s.ifNilTopic(msg, pubsub.ValidationReject),
	); result != pubsub.ValidationAccept {
		return result
	}

	m, err := s.readSyncCommitteeMessage(msg)
	if err != nil {
		log.WithError(err).Debug("Could not decode message")
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}

	pubKey, err := s.cfg.Chain.HeadValidatorIndexToPublicKey(ctx, m.ValidatorIndex)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}
	committeeIndices, err := s.cfg.Chain.HeadCurrentSyncCommitteeIndices(ctx, m.ValidatorIndex, m.Slot)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}

	// Validate the message's data according to the p2p specification.
	if result := withValidationPipeline(
		ctx,
		s.ifEmptyCommittee(committeeIndices, pubsub.ValidationIgnore),
		s.ifInvalidSyncMsgTime(m, pubsub.ValidationIgnore),
		s.ifIncorrectCommittee(committeeIndices, *msg.Topic, pubsub.ValidationReject),
		s.ifHasSeenSyncMsg(m, committeeIndices, pubsub.ValidationIgnore),
		s.ifInvalidSignature(m, pubKey, pubsub.ValidationIgnore),
	); result != pubsub.ValidationAccept {
		return result
	}

	s.markSyncCommitteeMessagesSeen(committeeIndices, m)

	msg.ValidatorData = m
	return pubsub.ValidationAccept
}

func (s *Service) readSyncCommitteeMessage(msg *pubsub.Message) (*ethpb.SyncCommitteeMessage, error) {
	raw, err := s.decodePubsubMessage(msg)
	if err != nil {
		return nil, err
	}
	m, ok := raw.(*ethpb.SyncCommitteeMessage)
	if !ok {
		return nil, errWrongMessage
	}
	if m == nil {
		return nil, errNilMessage
	}
	return m, nil
}

func (s *Service) markSyncCommitteeMessagesSeen(committeeIndices []types.CommitteeIndex, m *ethpb.SyncCommitteeMessage) {
	subCommitteeSize := params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount
	for _, idx := range committeeIndices {
		subnet := uint64(idx) / subCommitteeSize
		s.setSeenSyncMessageIndexSlot(m.Slot, m.ValidatorIndex, subnet)
	}
}

// Returns true if the node has received sync committee for the validator with index and slot.
func (s *Service) hasSeenSyncMessageIndexSlot(slot types.Slot, valIndex types.ValidatorIndex, subCommitteeIndex uint64) bool {
	s.seenSyncMessageLock.RLock()
	defer s.seenSyncMessageLock.RUnlock()

	b := append(bytesutil.Bytes32(uint64(slot)), bytesutil.Bytes32(uint64(valIndex))...)
	b = append(b, bytesutil.Bytes32(subCommitteeIndex)...)
	_, seen := s.seenSyncMessageCache.Get(string(b))
	return seen
}

// Set sync committee message validator index and slot as seen.
func (s *Service) setSeenSyncMessageIndexSlot(slot types.Slot, valIndex types.ValidatorIndex, subCommitteeIndex uint64) {
	s.seenSyncMessageLock.Lock()
	defer s.seenSyncMessageLock.Unlock()

	b := append(bytesutil.Bytes32(uint64(slot)), bytesutil.Bytes32(uint64(valIndex))...)
	b = append(b, bytesutil.Bytes32(subCommitteeIndex)...)
	s.seenSyncMessageCache.Add(string(b), true)
}

func (s *Service) ifSyncing(onErr pubsub.ValidationResult) validationFn {
	return func(ctx context.Context) pubsub.ValidationResult {
		if s.cfg.InitialSync.Syncing() {
			return onErr
		}
		return pubsub.ValidationAccept
	}
}

func (s *Service) ifNilTopic(msg *pubsub.Message, onErr pubsub.ValidationResult) validationFn {
	return func(ctx context.Context) pubsub.ValidationResult {
		if msg.Topic == nil {
			return onErr
		}
		return pubsub.ValidationAccept
	}
}

func (s *Service) ifEmptyCommittee(indices []types.CommitteeIndex, onErr pubsub.ValidationResult) validationFn {
	return func(ctx context.Context) pubsub.ValidationResult {
		if len(indices) == 0 {
			return onErr
		}
		return pubsub.ValidationAccept
	}
}

func (s *Service) ifInvalidDecodedPubsub(onErr pubsub.ValidationResult) validationFn {
	return func(ctx context.Context) pubsub.ValidationResult {
		return onErr
	}
}

// The message's `slot` is for the current slot (with a MAXIMUM_GOSSIP_CLOCK_DISPARITY allowance).
func (s *Service) ifInvalidSyncMsgTime(m *ethpb.SyncCommitteeMessage, onErr pubsub.ValidationResult) validationFn {
	return func(ctx context.Context) pubsub.ValidationResult {
		ctx, span := trace.StartSpan(ctx, "sync.ifInvalidSyncMsgTime")
		defer span.End()
		if err := altair.ValidateSyncMessageTime(
			m.Slot,
			s.cfg.Chain.GenesisTime(),
			params.BeaconNetworkConfig().MaximumGossipClockDisparity,
		); err != nil {
			traceutil.AnnotateError(span, err)
			return onErr
		}
		return pubsub.ValidationAccept
	}
}

// The `subnet_id` is valid for the given validator. This implies the validator is part of the broader
// current sync committee along with the correct subcommittee.
// Check for validity of validator index.
func (s *Service) ifIncorrectCommittee(
	committeeIndices []types.CommitteeIndex, topic string, onErr pubsub.ValidationResult,
) validationFn {
	return func(ctx context.Context) pubsub.ValidationResult {
		ctx, span := trace.StartSpan(ctx, "sync.ifIncorrectCommittee")
		defer span.End()
		isValid := false
		digest, err := s.forkDigest()
		if err != nil {
			traceutil.AnnotateError(span, err)
			return pubsub.ValidationIgnore
		}

		format := p2p.GossipTypeMapping[reflect.TypeOf(&ethpb.SyncCommitteeMessage{})]
		// Validate that the validator is in the correct committee.
		subCommitteeSize := params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount
		for _, idx := range committeeIndices {
			subnet := uint64(idx) / subCommitteeSize
			if strings.HasPrefix(topic, fmt.Sprintf(format, digest, subnet)) {
				isValid = true
				break
			}
		}
		if !isValid {
			return onErr
		}
		return pubsub.ValidationAccept
	}
}

// There has been no other valid sync committee signature for the declared `slot`, `validator_index`,
// and `subcommittee_index`. In the event of `validator_index` belongs to multiple subnets, as long
// as one subnet has not been seen, we should let it in.
func (s *Service) ifHasSeenSyncMsg(
	m *ethpb.SyncCommitteeMessage, committeeIndices []types.CommitteeIndex, onErr pubsub.ValidationResult,
) validationFn {
	return func(ctx context.Context) pubsub.ValidationResult {
		var isValid bool
		subCommitteeSize := params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount
		for _, idx := range committeeIndices {
			subnet := uint64(idx) / subCommitteeSize
			if s.hasSeenSyncMessageIndexSlot(m.Slot, m.ValidatorIndex, subnet) {
				isValid = false
			} else {
				isValid = true
			}
		}
		if !isValid {
			return onErr
		}
		return pubsub.ValidationAccept
	}
}

// The signature is valid for the message `beacon_block_root` for the validator referenced by `validator_index`.
func (s *Service) ifInvalidSignature(
	m *ethpb.SyncCommitteeMessage, pubKey [48]byte, onErr pubsub.ValidationResult,
) validationFn {
	return func(ctx context.Context) pubsub.ValidationResult {
		ctx, span := trace.StartSpan(ctx, "sync.ifInvalidSignature")
		defer span.End()
		d, err := s.cfg.Chain.HeadSyncCommitteeDomain(ctx, m.Slot)
		if err != nil {
			traceutil.AnnotateError(span, err)
			return pubsub.ValidationIgnore
		}
		rawBytes := p2ptypes.SSZBytes(m.BlockRoot)
		sigRoot, err := helpers.ComputeSigningRoot(&rawBytes, d)
		if err != nil {
			traceutil.AnnotateError(span, err)
			return pubsub.ValidationIgnore
		}

		blsSig, err := bls.SignatureFromBytes(m.Signature)
		if err != nil {
			traceutil.AnnotateError(span, err)
			return pubsub.ValidationReject
		}
		pKey, err := bls.PublicKeyFromBytes(pubKey[:])
		if err != nil {
			traceutil.AnnotateError(span, err)
			return pubsub.ValidationIgnore
		}
		verified := blsSig.Verify(pKey, sigRoot[:])
		if !verified {
			return onErr
		}
		return pubsub.ValidationAccept
	}
}

func withValidationPipeline(ctx context.Context, fns ...validationFn) pubsub.ValidationResult {
	for _, fn := range fns {
		if result := fn(ctx); result != pubsub.ValidationAccept {
			return result
		}
	}
	return pubsub.ValidationAccept
}
