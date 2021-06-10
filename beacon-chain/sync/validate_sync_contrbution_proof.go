package sync

import (
	"bytes"
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

// validateSyncContributionAndProof verifies the aggregated signature and the selection proof is valid before forwarding to the
// network and downstream services.
func (s *Service) validateSyncContributionAndProof(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	if pid == s.cfg.P2P.PeerID() {
		return pubsub.ValidationAccept
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateSyncContributionAndProof")
	defer span.End()

	// To process the following it requires the recent blocks to be present in the database, so we'll skip
	// validating or processing aggregated attestations until fully synced.
	if s.cfg.InitialSync.Syncing() {
		return pubsub.ValidationIgnore
	}

	raw, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Debug("Could not decode message")
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}
	m, ok := raw.(*ethpb.SignedContributionAndProof)
	if !ok {
		return pubsub.ValidationReject
	}
	if m.Message == nil {
		return pubsub.ValidationReject
	}
	if err := altair.ValidateNilSyncContribution(m); err != nil {
		return pubsub.ValidationReject
	}

	// Broadcast the aggregated attestation on a feed to notify other services in the beacon node
	// of a received aggregated attestation.
	s.cfg.OperationNotifier.OperationFeed().Send(&feed.Event{
		Type: operation.SyncContributionReceived,
		Data: &operation.SyncContributionReceivedData{
			Contribution: m.Message,
		},
	})

	if err := helpers.VerifySlotTime(uint64(s.cfg.Chain.GenesisTime().Unix()), m.Message.Contribution.Slot, params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}
	if !s.hasBlockAndState(ctx, bytesutil.ToBytes32(m.Message.Contribution.BlockRoot)) {
		return pubsub.ValidationIgnore
	}
	if m.Message.Contribution.SubcommitteeIndex >= params.BeaconConfig().SyncCommitteeSubnetCount {
		return pubsub.ValidationReject
	}

	if s.hasSeenSyncContributionIndexSlot(m.Message.Contribution.Slot, m.Message.AggregatorIndex, types.CommitteeIndex(m.Message.Contribution.SubcommitteeIndex)) {
		return pubsub.ValidationIgnore
	}
	if !altair.IsSyncCommitteeAggregator(m.Message.SelectionProof) {
		return pubsub.ValidationReject
	}
	// This could be better, retrieving the state multiple times with copies can
	// easily lead to higher resource consumption by the node.
	blkState, err := s.cfg.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(m.Message.Contribution.BlockRoot))
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}
	bState, ok := blkState.(iface.BeaconStateAltair)
	if !ok {
		log.Errorf("Sync contribution referencing non-altair state")
		return pubsub.ValidationReject
	}
	syncPubkeys, err := altair.SyncSubCommitteePubkeys(bState, types.CommitteeIndex(m.Message.Contribution.SubcommitteeIndex))
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}
	aggregator, err := bState.ValidatorAtIndexReadOnly(m.Message.AggregatorIndex)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}
	aggPubkey := aggregator.PublicKey()
	keyIsValid := false
	for _, pk := range syncPubkeys {
		if bytes.Equal(pk, aggPubkey[:]) {
			keyIsValid = true
			break
		}
	}
	if !keyIsValid {
		return pubsub.ValidationReject
	}
	if err := altair.VerifySyncSelectionData(bState, m.Message); err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}
	if err := helpers.VerifySigningRoot(m.Message, aggPubkey[:], m.Signature, params.BeaconConfig().DomainContributionAndProof[:]); err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}
	activePubkeys := [][]byte{}
	bVector := m.Message.Contribution.AggregationBits

	for i, pk := range syncPubkeys {
		if bVector.BitAt(uint64(i)) {
			activePubkeys = append(activePubkeys, pk)
		}
	}
	aggKey, err := bls.AggregatePublicKeys(activePubkeys)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}

	set := &bls.SignatureSet{
		PublicKeys: []bls.PublicKey{aggKey},
		Messages:   [][32]byte{bytesutil.ToBytes32(m.Message.Contribution.BlockRoot)},
		Signatures: [][]byte{m.Message.Contribution.Signature},
	}
	verified, err := set.Verify()
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}
	if !verified {
		return pubsub.ValidationReject
	}

	s.setSyncContributionIndexSlotSeen(m.Message.Contribution.Slot, m.Message.AggregatorIndex, types.CommitteeIndex(m.Message.Contribution.SubcommitteeIndex))

	msg.ValidatorData = m

	return pubsub.ValidationAccept
}

// Returns true if the node has received sync contribution for the aggregator with index,slot and subcommittee index.
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
