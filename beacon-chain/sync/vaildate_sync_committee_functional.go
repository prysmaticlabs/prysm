package sync

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

func (s *Service) validateSyncCommitteeMessageFunctional(
	ctx context.Context, pid peer.ID, msg *pubsub.Message,
) pubsub.ValidationResult {
	ctx, span := trace.StartSpan(ctx, "sync.validateSyncCommitteeMessage")
	defer span.End()

	// Accept if the message came from itself.
	if pid == s.cfg.P2P.PeerID() {
		return pubsub.ValidationAccept
	}

	// Ignore the sync committee message if the beacon node is syncing.
	if s.cfg.InitialSync.Syncing() {
		return pubsub.ValidationIgnore
	}

	if msg.Topic == nil {
		return pubsub.ValidationReject
	}
	raw, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Debug("Could not decode message")
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}
	m, ok := raw.(*ethpb.SyncCommitteeMessage)
	if !ok {
		return pubsub.ValidationReject
	}
	if m == nil {
		return pubsub.ValidationReject
	}

	return withValidationPipeline(
		ctx,
		s.ifSyncing(pubsub.ValidationReject),
		s.ifInvalidSyncMsgTime(m, pubsub.ValidationIgnore),
		s.ifIncorrectCommittee(m, pubsub.ValidationReject),
		s.ifHasSeenSyncMsg(pubsub.ValidationIgnore),
		s.ifInvalidSignature(pubsub.ValidationIgnore),
	)
}

type validationFn func(ctx context.Context) pubsub.ValidationResult

func (s *Service) ifSyncing(onErr pubsub.ValidationResult) validationFn {
	return func(ctx context.Context) pubsub.ValidationResult {
		if s.cfg.InitialSync.Syncing() {
			return pubsub.ValidationIgnore
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
func (s *Service) ifIncorrectCommittee(m *ethpb.SyncCommitteeMessage, onErr pubsub.ValidationResult) validationFn {
	return func(ctx context.Context) pubsub.ValidationResult {
		ctx, span := trace.StartSpan(ctx, "sync.ifIncorrectCommittee")
		defer span.End()
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
		if len(committeeIndices) == 0 {
			return pubsub.ValidationIgnore
		}

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
			if strings.HasPrefix(*msg.Topic, fmt.Sprintf(format, digest, subnet)) {
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

func (s *Service) ifHasSeenSyncMsg(onErr pubsub.ValidationResult) validationFn {
	return func(ctx context.Context) pubsub.ValidationResult {
		//for _, idx := range committeeIndices {
		//	subnet := uint64(idx) / subCommitteeSize
		//	if s.hasSeenSyncMessageIndexSlot(m.Slot, m.ValidatorIndex, subnet) {
		//		isValid = false
		//	} else {
		//		isValid = true
		//	}
		//}
		//if !isValid {
		//	return onErr
		//}
		return pubsub.ValidationAccept
	}
}

func (s *Service) ifInvalidSignature(onErr pubsub.ValidationResult) validationFn {
	return func(ctx context.Context) pubsub.ValidationResult {
		// The signature is valid for the message `beacon_block_root` for the validator referenced by `validator_index`.
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
			return pubsub.ValidationReject
		}

		for _, idx := range committeeIndices {
			subnet := uint64(idx) / subCommitteeSize
			s.setSeenSyncMessageIndexSlot(m.Slot, m.ValidatorIndex, subnet)
		}
		return onErr
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
