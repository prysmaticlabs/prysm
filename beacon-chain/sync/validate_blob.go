package sync

import (
	"context"
	"fmt"
	"strings"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/crypto/rand"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/network/forks"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	prysmTime "github.com/prysmaticlabs/prysm/v4/time"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"github.com/sirupsen/logrus"
)

func (s *Service) handleBlobParentStatus(ctx context.Context, root [32]byte) pubsub.ValidationResult {
	if s.cfg.chain.HasBlock(ctx, root) {
		// the parent will not be kept if it's invalid
		return pubsub.ValidationAccept
	}
	if s.hasBadBlock(root) {
		// [REJECT] The sidecar's block's parent (defined by sidecar.block_parent_root) passes validation.
		return pubsub.ValidationReject
	}
	// [IGNORE] The sidecar's block's parent (defined by sidecar.block_parent_root) has been seen (via both gossip and non-gossip sources)
	return pubsub.ValidationIgnore
}

func (s *Service) validateBlob(ctx context.Context, pid peer.ID, msg *pubsub.Message) (pubsub.ValidationResult, error) {
	receivedTime := prysmTime.Now()

	if pid == s.cfg.p2p.PeerID() {
		return pubsub.ValidationAccept, nil
	}
	if s.cfg.initialSync.Syncing() {
		return pubsub.ValidationIgnore, nil
	}
	if msg.Topic == nil {
		return pubsub.ValidationReject, errInvalidTopic
	}
	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Error("Failed to decode message")
		return pubsub.ValidationReject, err
	}

	sBlob, ok := m.(*eth.SignedBlobSidecar)
	if !ok {
		log.WithField("message", m).Error("Message is not of type *eth.SignedBlobSidecar")
		return pubsub.ValidationReject, errWrongMessage
	}
	blob := sBlob.Message

	// [REJECT] The sidecar is for the correct topic -- i.e. sidecar.index matches the topic {index}.
	want := fmt.Sprintf("blob_sidecar_%d", blob.Index)
	if !strings.Contains(*msg.Topic, want) {
		log.WithFields(blobFields(blob)).Debug("Sidecar blob does not match topic")
		return pubsub.ValidationReject, fmt.Errorf("wrong topic name: %s", *msg.Topic)
	}

	// [IGNORE] The sidecar is not from a future slot (with a MAXIMUM_GOSSIP_CLOCK_DISPARITY allowance) --
	// i.e. validate that sidecar.slot <= current_slot (a client MAY queue future blocks for processing at the appropriate slot).
	genesisTime := uint64(s.cfg.chain.GenesisTime().Unix())
	if err := slots.VerifyTime(genesisTime, blob.Slot, earlyBlockProcessingTolerance); err != nil {
		log.WithError(err).WithFields(blobFields(blob)).Debug("Ignored blob: too far into future")
		return pubsub.ValidationIgnore, errors.Wrap(err, "blob too far into future")
	}

	// [IGNORE] The sidecar is from a slot greater than the latest finalized slot --
	// i.e. validate that sidecar.slot > compute_start_slot_at_epoch(state.finalized_checkpoint.epoch)
	startSlot, err := slots.EpochStart(s.cfg.chain.FinalizedCheckpt().Epoch)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	if startSlot >= blob.Slot {
		err := fmt.Errorf("finalized slot %d greater or equal to blob slot %d", startSlot, blob.Slot)
		log.WithFields(blobFields(blob)).Debug(err)
		return pubsub.ValidationIgnore, err
	}

	// Handle the parent status (not seen or invalid cases)
	parentRoot := bytesutil.ToBytes32(blob.BlockParentRoot)
	switch parentStatus := s.handleBlobParentStatus(ctx, parentRoot); parentStatus {
	case pubsub.ValidationIgnore:
		log.WithFields(blobFields(blob)).Debug("Parent block not found - saving blob to cache")
		go func() {
			if err := s.sendBatchRootRequest(context.Background(), [][32]byte{parentRoot}, rand.NewGenerator()); err != nil {
				log.WithError(err).WithFields(blobFields(blob)).Debug("Failed to send batch root request")
			}
		}()
		s.pendingBlobSidecars.add(sBlob)
		return pubsub.ValidationIgnore, nil
	case pubsub.ValidationReject:
		log.WithFields(blobFields(blob)).Warning("Rejected blob: parent block is invalid")
		return pubsub.ValidationReject, nil
	default:
	}

	pubsubResult, err := s.validateBlobPostSeenParent(ctx, sBlob)
	if err != nil {
		return pubsubResult, err
	}
	if pubsubResult != pubsub.ValidationAccept {
		return pubsubResult, nil
	}

	startTime, err := slots.ToTime(genesisTime, blob.Slot)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	fields := blobFields(blob)
	sinceSlotStartTime := receivedTime.Sub(startTime)
	fields["sinceSlotStartTime"] = sinceSlotStartTime
	fields["validationTime"] = s.cfg.clock.Now().Sub(receivedTime)
	log.WithFields(fields).Debug("Received blob sidecar gossip")

	blobSidecarArrivalGossipSummary.Observe(float64(sinceSlotStartTime.Milliseconds()))

	msg.ValidatorData = sBlob

	return pubsub.ValidationAccept, nil
}

func (s *Service) validateBlobPostSeenParent(ctx context.Context, sBlob *eth.SignedBlobSidecar) (pubsub.ValidationResult, error) {
	blob := sBlob.Message
	parentRoot := bytesutil.ToBytes32(blob.BlockParentRoot)

	// [REJECT] The sidecar is from a higher slot than the sidecar's block's parent (defined by sidecar.block_parent_root).
	parentSlot, err := s.cfg.chain.RecentBlockSlot(parentRoot)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	if parentSlot >= blob.Slot {
		err := fmt.Errorf("parent block slot %d greater or equal to blob slot %d", parentSlot, blob.Slot)
		log.WithFields(blobFields(blob)).Debug(err)
		return pubsub.ValidationReject, err
	}

	// [REJECT] The proposer signature, signed_blob_sidecar.signature,
	// is valid with respect to the sidecar.proposer_index pubkey.
	parentState, err := s.cfg.stateGen.StateByRoot(ctx, parentRoot)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	if err := verifyBlobSignature(parentState, sBlob); err != nil {
		log.WithError(err).WithFields(blobFields(blob)).Debug("Failed to verify blob signature")
		return pubsub.ValidationReject, err
	}

	// [IGNORE] The sidecar is the only sidecar with valid signature received for the tuple (sidecar.block_root, sidecar.index).
	if s.hasSeenBlobIndex(blob.BlockRoot, blob.Index) {
		return pubsub.ValidationIgnore, nil
	}

	// [REJECT] The sidecar is proposed by the expected proposer_index for the block's slot in the context of the current shuffling (defined by block_parent_root/slot)
	parentState, err = transition.ProcessSlotsUsingNextSlotCache(ctx, parentState, parentRoot[:], blob.Slot)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	idx, err := helpers.BeaconProposerIndex(ctx, parentState)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	if blob.ProposerIndex != idx {
		err := fmt.Errorf("expected proposer index %d, got %d", idx, blob.ProposerIndex)
		log.WithFields(blobFields(blob)).Debug(err)
		return pubsub.ValidationReject, err
	}
	return pubsub.ValidationAccept, nil
}

func verifyBlobSignature(st state.BeaconState, blob *eth.SignedBlobSidecar) error {
	currentEpoch := slots.ToEpoch(blob.Message.Slot)
	fork, err := forks.Fork(currentEpoch)
	if err != nil {
		return err
	}
	domain, err := signing.Domain(fork, currentEpoch, params.BeaconConfig().DomainBlobSidecar, st.GenesisValidatorsRoot())
	if err != nil {
		return err
	}
	proposer, err := st.ValidatorAtIndex(blob.Message.ProposerIndex)
	if err != nil {
		return err
	}
	pb, err := bls.PublicKeyFromBytes(proposer.PublicKey)
	if err != nil {
		return err
	}
	sig, err := bls.SignatureFromBytes(blob.Signature)
	if err != nil {
		return err
	}
	sr, err := signing.ComputeSigningRoot(blob.Message, domain)
	if err != nil {
		return err
	}
	if !sig.Verify(pb, sr[:]) {
		return signing.ErrSigFailedToVerify
	}

	return nil
}

// Returns true if the blob with the same root and index has been seen before.
func (s *Service) hasSeenBlobIndex(root []byte, index uint64) bool {
	s.seenBlobLock.RLock()
	defer s.seenBlobLock.RUnlock()
	b := append(root, bytesutil.Bytes32(index)...)
	_, seen := s.seenBlobCache.Get(string(b))
	return seen
}

// Set blob index and root as seen.
func (s *Service) setSeenBlobIndex(root []byte, index uint64) {
	s.seenBlobLock.Lock()
	defer s.seenBlobLock.Unlock()
	b := append(root, bytesutil.Bytes32(index)...)
	s.seenBlobCache.Add(string(b), true)
}

func blobFields(b *eth.BlobSidecar) logrus.Fields {
	return logrus.Fields{
		"slot":          b.Slot,
		"proposerIndex": b.ProposerIndex,
		"blockRoot":     fmt.Sprintf("%#x", b.BlockRoot),
		"index":         b.Index,
	}
}
