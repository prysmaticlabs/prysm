package sync

import (
	"context"
	"errors"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	opfeed "github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	"github.com/prysmaticlabs/prysm/config/features"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/monitoring/tracing"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"go.opencensus.io/trace"
)

// validateSyncContributionAndProof verifies the aggregated signature and the selection proof is valid before forwarding to the
// network and downstream services.
// Gossip Validation Conditions:
// [IGNORE] The contribution's slot is for the current slot (with a MAXIMUM_GOSSIP_CLOCK_DISPARITY allowance), i.e. contribution.slot == current_slot.
// [REJECT] The subcommittee index is in the allowed range, i.e. contribution.subcommittee_index < SYNC_COMMITTEE_SUBNET_COUNT.
// [REJECT] The contribution has participants -- that is, any(contribution.aggregation_bits).
// [REJECT] contribution_and_proof.selection_proof selects the validator as an aggregator for the slot -- i.e.
// is_sync_committee_aggregator(contribution_and_proof.selection_proof) returns True.
// [REJECT] The aggregator's validator index is in the declared subcommittee of the current sync committee -- i.e.
// state.validators[contribution_and_proof.aggregator_index].pubkey in get_sync_subcommittee_pubkeys(state, contribution.subcommittee_index).
// [IGNORE] The sync committee contribution is the first valid contribution received for the aggregator with
// index contribution_and_proof.aggregator_index for the slot contribution.slot and subcommittee index contribution.subcommittee_index
// (this requires maintaining a cache of size SYNC_COMMITTEE_SIZE for this topic that can be flushed after each slot).
// [REJECT] The contribution_and_proof.selection_proof is a valid signature of the SyncAggregatorSelectionData derived from
// the contribution by the validator with index contribution_and_proof.aggregator_index.
// [REJECT] The aggregator signature, signed_contribution_and_proof.signature, is valid.
// [REJECT] The aggregate signature is valid for the message beacon_block_root and aggregate pubkey derived from the participation
// info in aggregation_bits for the subcommittee specified by the contribution.subcommittee_index.
func (s *Service) validateSyncContributionAndProof(ctx context.Context, pid peer.ID, msg *pubsub.Message) (pubsub.ValidationResult, error) {
	ctx, span := trace.StartSpan(ctx, "sync.validateSyncContributionAndProof")
	defer span.End()

	// Accept the sync committee contribution if the contribution came from itself.
	if pid == s.cfg.p2p.PeerID() {
		return pubsub.ValidationAccept, nil
	}

	// Ignore the sync committee contribution if the beacon node is syncing.
	if s.cfg.initialSync.Syncing() {
		return pubsub.ValidationIgnore, nil
	}
	m, err := s.readSyncContributionMessage(msg)
	if err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationReject, err
	}

	// The contribution's slot is for the current slot (with a `MAXIMUM_GOSSIP_CLOCK_DISPARITY` allowance).
	if err := altair.ValidateSyncMessageTime(m.Message.Contribution.Slot, s.cfg.chain.GenesisTime(), params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationIgnore, err
	}
	// Validate the message's data according to the p2p specification.
	if result, err := validationPipeline(
		ctx,
		rejectIncorrectSubcommitteeIndex(m),
		rejectEmptyContribution(m),
		s.ignoreSeenSyncContribution(m),
		rejectInvalidAggregator(m),
		s.rejectInvalidIndexInSubCommittee(m),
		s.rejectInvalidSelectionProof(m),
		s.rejectInvalidContributionSignature(m),
		s.rejectInvalidSyncAggregateSignature(m),
	); result != pubsub.ValidationAccept {
		return result, err
	}

	s.setSyncContributionIndexSlotSeen(m.Message.Contribution.Slot, m.Message.AggregatorIndex, types.CommitteeIndex(m.Message.Contribution.SubcommitteeIndex))

	msg.ValidatorData = m

	// Broadcast the contribution on a feed to notify other services in the beacon node
	// of a received contribution.
	s.cfg.operationNotifier.OperationFeed().Send(&feed.Event{
		Type: opfeed.SyncCommitteeContributionReceived,
		Data: &opfeed.SyncCommitteeContributionReceivedData{
			Contribution: m,
		},
	})

	return pubsub.ValidationAccept, nil
}

// Parse a sync contribution message from a pubsub message.
func (s *Service) readSyncContributionMessage(msg *pubsub.Message) (*ethpb.SignedContributionAndProof, error) {
	raw, err := s.decodePubsubMessage(msg)
	if err != nil {
		return nil, err
	}
	m, ok := raw.(*ethpb.SignedContributionAndProof)
	if !ok {
		return nil, errWrongMessage
	}
	if err := altair.ValidateNilSyncContribution(m); err != nil {
		return nil, errNilMessage
	}
	return m, nil
}

func rejectIncorrectSubcommitteeIndex(
	m *ethpb.SignedContributionAndProof,
) validationFn {
	return func(ctx context.Context) (pubsub.ValidationResult, error) {
		_, span := trace.StartSpan(ctx, "sync.rejectIncorrectSubcommitteeIndex")
		defer span.End()
		// The subcommittee index is in the allowed range, i.e. `contribution.subcommittee_index < SYNC_COMMITTEE_SUBNET_COUNT`.
		if m.Message.Contribution.SubcommitteeIndex >= params.BeaconConfig().SyncCommitteeSubnetCount {
			return pubsub.ValidationReject, errors.New("subcommittee index is invalid")
		}

		return pubsub.ValidationAccept, nil
	}
}

func rejectEmptyContribution(m *ethpb.SignedContributionAndProof) validationFn {
	return func(ctx context.Context) (pubsub.ValidationResult, error) {
		bVector := m.Message.Contribution.AggregationBits
		// In the event no bit is set for the
		// sync contribution, we reject the message.
		if bVector.Count() == 0 {
			return pubsub.ValidationReject, errors.New("bitvector count is 0")
		}
		return pubsub.ValidationAccept, nil
	}
}

func (s *Service) ignoreSeenSyncContribution(m *ethpb.SignedContributionAndProof) validationFn {
	return func(ctx context.Context) (pubsub.ValidationResult, error) {
		seen := s.hasSeenSyncContributionIndexSlot(m.Message.Contribution.Slot, m.Message.AggregatorIndex, types.CommitteeIndex(m.Message.Contribution.SubcommitteeIndex))
		if seen {
			return pubsub.ValidationIgnore, nil
		}
		return pubsub.ValidationAccept, nil
	}
}

func rejectInvalidAggregator(m *ethpb.SignedContributionAndProof) validationFn {
	return func(ctx context.Context) (pubsub.ValidationResult, error) {
		// The `contribution_and_proof.selection_proof` selects the validator as an aggregator for the slot.
		if isAggregator, err := altair.IsSyncCommitteeAggregator(m.Message.SelectionProof); err != nil || !isAggregator {
			return pubsub.ValidationReject, err
		}
		return pubsub.ValidationAccept, nil
	}
}

func (s *Service) rejectInvalidIndexInSubCommittee(m *ethpb.SignedContributionAndProof) validationFn {
	return func(ctx context.Context) (pubsub.ValidationResult, error) {
		_, span := trace.StartSpan(ctx, "sync.rejectInvalidIndexInSubCommittee")
		defer span.End()
		// The aggregator's validator index is in the declared subcommittee of the current sync committee.
		committeeIndices, err := s.cfg.chain.HeadSyncCommitteeIndices(ctx, m.Message.AggregatorIndex, m.Message.Contribution.Slot)
		if err != nil {
			tracing.AnnotateError(span, err)
			return pubsub.ValidationIgnore, err
		}
		if len(committeeIndices) == 0 {
			tracing.AnnotateError(span, err)
			return pubsub.ValidationIgnore, err
		}
		isValid := false
		subCommitteeSize := params.BeaconConfig().SyncCommitteeSize / params.BeaconConfig().SyncCommitteeSubnetCount
		for _, i := range committeeIndices {
			if uint64(i)/subCommitteeSize == m.Message.Contribution.SubcommitteeIndex {
				isValid = true
				break
			}
		}
		if !isValid {
			return pubsub.ValidationReject, errors.New("invalid subcommittee index")
		}
		return pubsub.ValidationAccept, nil
	}
}

func (s *Service) rejectInvalidSelectionProof(m *ethpb.SignedContributionAndProof) validationFn {
	return func(ctx context.Context) (pubsub.ValidationResult, error) {
		_, span := trace.StartSpan(ctx, "sync.rejectInvalidSelectionProof")
		defer span.End()
		// The `contribution_and_proof.selection_proof` is a valid signature of the `SyncAggregatorSelectionData`.
		if err := s.verifySyncSelectionData(ctx, m.Message); err != nil {
			tracing.AnnotateError(span, err)
			return pubsub.ValidationReject, err
		}
		return pubsub.ValidationAccept, nil
	}
}

func (s *Service) rejectInvalidContributionSignature(m *ethpb.SignedContributionAndProof) validationFn {
	return func(ctx context.Context) (pubsub.ValidationResult, error) {
		_, span := trace.StartSpan(ctx, "sync.rejectInvalidContributionSignature")
		defer span.End()
		// The aggregator signature, `signed_contribution_and_proof.signature`, is valid.
		d, err := s.cfg.chain.HeadSyncContributionProofDomain(ctx, m.Message.Contribution.Slot)
		if err != nil {
			tracing.AnnotateError(span, err)
			return pubsub.ValidationIgnore, err
		}
		pubkey, err := s.cfg.chain.HeadValidatorIndexToPublicKey(ctx, m.Message.AggregatorIndex)
		if err != nil {
			return pubsub.ValidationIgnore, err
		}
		if features.Get().EnableBatchVerification {
			publicKey, err := bls.PublicKeyFromBytes(pubkey[:])
			if err != nil {
				tracing.AnnotateError(span, err)
				return pubsub.ValidationReject, err
			}
			root, err := signing.ComputeSigningRoot(m.Message, d)
			if err != nil {
				tracing.AnnotateError(span, err)
				return pubsub.ValidationReject, err
			}
			set := &bls.SignatureBatch{
				Messages:   [][32]byte{root},
				PublicKeys: []bls.PublicKey{publicKey},
				Signatures: [][]byte{m.Signature},
			}
			return s.validateWithBatchVerifier(ctx, "sync contribution signature", set)
		}

		if err := signing.VerifySigningRoot(m.Message, pubkey[:], m.Signature, d); err != nil {
			tracing.AnnotateError(span, err)
			return pubsub.ValidationReject, err
		}
		return pubsub.ValidationAccept, nil
	}
}

func (s *Service) rejectInvalidSyncAggregateSignature(m *ethpb.SignedContributionAndProof) validationFn {
	return func(ctx context.Context) (pubsub.ValidationResult, error) {
		_, span := trace.StartSpan(ctx, "sync.rejectInvalidSyncAggregateSignature")
		defer span.End()
		// The aggregate signature is valid for the message `beacon_block_root` and aggregate pubkey
		// derived from the participation info in `aggregation_bits` for the subcommittee specified by the `contribution.subcommittee_index`.
		var activePubkeys []bls.PublicKey
		var activeRawPubkeys [][]byte
		syncPubkeys, err := s.cfg.chain.HeadSyncCommitteePubKeys(ctx, m.Message.Contribution.Slot, types.CommitteeIndex(m.Message.Contribution.SubcommitteeIndex))
		if err != nil {
			return pubsub.ValidationIgnore, err
		}
		bVector := m.Message.Contribution.AggregationBits
		// In the event no bit is set for the
		// sync contribution, we reject the message.
		if bVector.Count() == 0 {
			return pubsub.ValidationReject, errors.New("bitvector count is 0")
		}
		for i, pk := range syncPubkeys {
			if bVector.BitAt(uint64(i)) {
				pubK, err := bls.PublicKeyFromBytes(pk)
				if err != nil {
					tracing.AnnotateError(span, err)
					return pubsub.ValidationIgnore, err
				}
				activePubkeys = append(activePubkeys, pubK)
				activeRawPubkeys = append(activeRawPubkeys, pk)
			}
		}
		d, err := s.cfg.chain.HeadSyncCommitteeDomain(ctx, m.Message.Contribution.Slot)
		if err != nil {
			tracing.AnnotateError(span, err)
			return pubsub.ValidationIgnore, err
		}
		rawBytes := p2ptypes.SSZBytes(m.Message.Contribution.BlockRoot)
		sigRoot, err := signing.ComputeSigningRoot(&rawBytes, d)
		if err != nil {
			tracing.AnnotateError(span, err)
			return pubsub.ValidationIgnore, err
		}
		// Aggregate pubkeys separately again to allow
		// for signature sets to be created for batch verification.
		if features.Get().EnableBatchVerification {
			aggKey, err := bls.AggregatePublicKeys(activeRawPubkeys)
			if err != nil {
				tracing.AnnotateError(span, err)
				return pubsub.ValidationIgnore, err
			}
			set := &bls.SignatureBatch{
				Messages:   [][32]byte{sigRoot},
				PublicKeys: []bls.PublicKey{aggKey},
				Signatures: [][]byte{m.Message.Contribution.Signature},
			}
			return s.validateWithBatchVerifier(ctx, "sync contribution aggregate signature", set)
		}
		sig, err := bls.SignatureFromBytes(m.Message.Contribution.Signature)
		if err != nil {
			tracing.AnnotateError(span, err)
			return pubsub.ValidationReject, err
		}
		verified := sig.Eth2FastAggregateVerify(activePubkeys, sigRoot)
		if !verified {
			return pubsub.ValidationReject, errors.New("verification failed")
		}
		return pubsub.ValidationAccept, nil
	}
}

// Returns true if the node has received sync contribution for the aggregator with index, slot and subcommittee index.
func (s *Service) hasSeenSyncContributionIndexSlot(slot types.Slot, aggregatorIndex types.ValidatorIndex, subComIdx types.CommitteeIndex) bool {
	s.seenSyncContributionLock.RLock()
	defer s.seenSyncContributionLock.RUnlock()

	b := append(bytesutil.Bytes32(uint64(aggregatorIndex)), bytesutil.Bytes32(uint64(slot))...)
	b = append(b, bytesutil.Bytes32(uint64(subComIdx))...)
	_, seen := s.seenSyncContributionCache.Get(string(b))
	return seen
}

// Set sync contributor's aggregate index, slot and subcommittee index as seen.
func (s *Service) setSyncContributionIndexSlotSeen(slot types.Slot, aggregatorIndex types.ValidatorIndex, subComIdx types.CommitteeIndex) {
	s.seenSyncContributionLock.Lock()
	defer s.seenSyncContributionLock.Unlock()
	b := append(bytesutil.Bytes32(uint64(aggregatorIndex)), bytesutil.Bytes32(uint64(slot))...)
	b = append(b, bytesutil.Bytes32(uint64(subComIdx))...)
	s.seenSyncContributionCache.Add(string(b), true)
}

// verifySyncSelectionData verifies that the provided sync contribution has a valid
// selection proof.
func (s *Service) verifySyncSelectionData(ctx context.Context, m *ethpb.ContributionAndProof) error {
	selectionData := &ethpb.SyncAggregatorSelectionData{Slot: m.Contribution.Slot, SubcommitteeIndex: m.Contribution.SubcommitteeIndex}
	domain, err := s.cfg.chain.HeadSyncSelectionProofDomain(ctx, m.Contribution.Slot)
	if err != nil {
		return err
	}
	pubkey, err := s.cfg.chain.HeadValidatorIndexToPublicKey(ctx, m.AggregatorIndex)
	if err != nil {
		return err
	}
	if features.Get().EnableBatchVerification {
		publicKey, err := bls.PublicKeyFromBytes(pubkey[:])
		if err != nil {
			return err
		}
		root, err := signing.ComputeSigningRoot(selectionData, domain)
		if err != nil {
			return err
		}
		set := &bls.SignatureBatch{
			Messages:   [][32]byte{root},
			PublicKeys: []bls.PublicKey{publicKey},
			Signatures: [][]byte{m.SelectionProof},
		}
		valid, err := s.validateWithBatchVerifier(ctx, "sync contribution selection signature", set)
		if err != nil {
			return err
		}
		if valid != pubsub.ValidationAccept {
			return errors.New("invalid sync selection proof provided")
		}
		return nil
	}
	return signing.VerifySigningRoot(selectionData, pubkey[:], m.SelectionProof, domain)
}
