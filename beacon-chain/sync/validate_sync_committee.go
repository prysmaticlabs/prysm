package sync

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed/operation"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	p2ptypes "github.com/prysmaticlabs/prysm/beacon-chain/p2p/types"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	prysmv2 "github.com/prysmaticlabs/prysm/proto/prysm/v2"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"go.opencensus.io/trace"
)

func (s *Service) validateSyncCommittee(ctx context.Context, pid peer.ID, msg *pubsub.Message) pubsub.ValidationResult {
	if pid == s.cfg.P2P.PeerID() {
		return pubsub.ValidationAccept
	}
	// Attestation processing requires the target block to be present in the database, so we'll skip
	// validating or processing attestations until fully synced.
	if s.cfg.InitialSync.Syncing() {
		return pubsub.ValidationIgnore
	}
	ctx, span := trace.StartSpan(ctx, "sync.validateSyncCommittee")
	defer span.End()

	if msg.Topic == nil {
		return pubsub.ValidationReject
	}

	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Debug("Could not decode message")
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}

	comMsg, ok := m.(*prysmv2.SyncCommitteeMessage)
	if !ok {
		return pubsub.ValidationReject
	}
	if comMsg == nil {
		return pubsub.ValidationReject
	}

	// Broadcast the sync committee on a feed to notify other services in the beacon node
	// of a received sync committee.
	s.cfg.OperationNotifier.OperationFeed().Send(&feed.Event{
		Type: operation.SyncCommMessageReceived,
		Data: &operation.SyncCommReceivedData{
			Message: comMsg,
		},
	})

	if err := helpers.VerifySlotTime(uint64(s.cfg.Chain.GenesisTime().Unix()), comMsg.Slot, params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}
	// Verify this the first attestation received for the participating validator for the slot.
	if s.hasSeenSyncMessageIndexSlot(comMsg.Slot, comMsg.ValidatorIndex) {
		return pubsub.ValidationIgnore
	}

	// Verify the block being voted and the processed state is in DB and. The block should have passed validation if it's in the DB.
	blockRoot := bytesutil.ToBytes32(comMsg.BlockRoot)
	if !s.hasBlockAndState(ctx, blockRoot) {
		return pubsub.ValidationIgnore
	}

	// This could be better, retrieving the state multiple times with copies can
	// easily lead to higher resource consumption by the node.
	blkState, err := s.cfg.StateGen.StateByRoot(ctx, blockRoot)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}
	bState, ok := blkState.(iface.BeaconStateAltair)
	if !ok {
		log.Errorf("Sync contribution referencing non-altair state")
		return pubsub.ValidationReject
	}
	// Check for validity of validator index.
	_, err = bState.ValidatorAtIndexReadOnly(comMsg.ValidatorIndex)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}
	subs, err := altair.SubnetsForSyncCommittee(bState, comMsg.ValidatorIndex)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}

	isValid := false
	digest, err := s.currentForkDigest()
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}
	format := p2p.GossipTypeMapping[reflect.TypeOf(&prysmv2.SyncCommitteeMessage{})]

	// Validate that the validator is in the correct committee.
	for _, idx := range subs {
		if strings.HasPrefix(*msg.Topic, fmt.Sprintf(format, digest, idx)) {
			isValid = true
			break
		}
	}
	if !isValid {
		return pubsub.ValidationReject
	}

	val, err := bState.ValidatorAtIndexReadOnly(comMsg.ValidatorIndex)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}
	d, err := helpers.Domain(bState.Fork(), helpers.SlotToEpoch(bState.Slot()), params.BeaconConfig().DomainSyncCommittee, bState.GenesisValidatorRoot())
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}
	rawBytes := p2ptypes.SSZBytes(blockRoot[:])
	sigRoot, err := helpers.ComputeSigningRoot(&rawBytes, d)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}

	rawKey := val.PublicKey()
	blsSig, err := bls.SignatureFromBytes(comMsg.Signature)
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationReject
	}
	pKey, err := bls.PublicKeyFromBytes(rawKey[:])
	if err != nil {
		traceutil.AnnotateError(span, err)
		return pubsub.ValidationIgnore
	}
	verified := blsSig.Verify(pKey, sigRoot[:])
	if !verified {
		return pubsub.ValidationReject
	}

	s.setSeenSyncMessageIndexSlot(comMsg.Slot, comMsg.ValidatorIndex)

	msg.ValidatorData = comMsg

	return pubsub.ValidationAccept
}

// Returns true if the node has received sync committee for the validator with index and slot.
func (s *Service) hasSeenSyncMessageIndexSlot(slot types.Slot, valIndex types.ValidatorIndex) bool {
	s.seenSyncMessageLock.RLock()
	defer s.seenSyncMessageLock.RUnlock()

	b := append(bytesutil.Bytes32(uint64(slot)), bytesutil.Bytes32(uint64(valIndex))...)
	_, seen := s.seenSyncMessageCache.Get(string(b))
	return seen
}

// Set sync committee message validator index and slot as seen.
func (s *Service) setSeenSyncMessageIndexSlot(slot types.Slot, valIndex types.ValidatorIndex) {
	s.seenSyncMessageLock.Lock()
	defer s.seenSyncMessageLock.Unlock()
	b := append(bytesutil.Bytes32(uint64(slot)), bytesutil.Bytes32(uint64(valIndex))...)
	s.seenSyncMessageCache.Add(string(b), true)
}
