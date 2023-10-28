package sync

import (
	"context"
	"fmt"
	"strings"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/crypto/rand"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
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

	blob, ok := m.(*eth.BlobSidecar)
	if !ok {
		log.WithField("message", m).Error("Message is not of type *eth.SignedBlobSidecar")
		return pubsub.ValidationReject, errWrongMessage
	}

	// [REJECT] The sidecar's index is consistent with `MAX_BLOBS_PER_BLOCK` -- i.e. `sidecar.index < MAX_BLOBS_PER_BLOCK`
	if blob.Index >= fieldparams.MaxBlobsPerBlock {
		log.WithFields(blobFields(blob)).Debug("Sidecar index > MAX_BLOBS_PER_BLOCK")
		return pubsub.ValidationReject, errors.New("incorrect blob sidecar index")
	}

	// [REJECT] The sidecar is for the correct subnet -- i.e. compute_subnet_for_blob_sidecar(sidecar.index) == subnet_id.
	want := fmt.Sprintf("blob_sidecar_%d", computeSubnetForBlobSidecar(blob.Index))
	if !strings.Contains(*msg.Topic, want) {
		log.WithFields(blobFields(blob)).Debug("Sidecar index  does not match topic")
		return pubsub.ValidationReject, fmt.Errorf("wrong topic name: %s", *msg.Topic)
	}

	// [IGNORE] The sidecar is not from a future slot (with a `MAXIMUM_GOSSIP_CLOCK_DISPARITY` allowance) -- i.e. validate that `block_header.slot <= current_slot`
	// (a client MAY queue future sidecars for processing at the appropriate slot).
	blockHeader := blob.SignedBlockHeader.Header
	genesisTime := uint64(s.cfg.chain.GenesisTime().Unix())
	if err := slots.VerifyTime(genesisTime, blockHeader.Slot, earlyBlockProcessingTolerance); err != nil {
		log.WithError(err).WithFields(blobFields(blob)).Debug("Ignored blob: too far into future")
		return pubsub.ValidationIgnore, errors.Wrap(err, "blob too far into future")
	}

	// [IGNORE] The sidecar is from a slot greater than the latest finalized slot --
	// i.e. validate that `block_header.slot > compute_start_slot_at_epoch(state.finalized_checkpoint.epoch)`
	startSlot, err := slots.EpochStart(s.cfg.chain.FinalizedCheckpt().Epoch)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	if startSlot >= blockHeader.Slot {
		err := fmt.Errorf("finalized slot %d greater or equal to blob slot %d", startSlot, blockHeader.Slot)
		log.WithFields(blobFields(blob)).Debug(err)
		return pubsub.ValidationIgnore, err
	}

	// Handle the parent status (not seen or invalid cases)
	parentRoot := bytesutil.ToBytes32(blockHeader.ParentRoot)
	switch parentStatus := s.handleBlobParentStatus(ctx, parentRoot); parentStatus {
	case pubsub.ValidationIgnore:
		log.WithFields(blobFields(blob)).Debug("Parent block not found - saving blob to cache")
		go func() {
			if err := s.sendBatchRootRequest(context.Background(), [][32]byte{parentRoot}, rand.NewGenerator()); err != nil {
				log.WithError(err).WithFields(blobFields(blob)).Debug("Failed to send batch root request")
			}
		}()
		s.pendingBlobSidecars.add(blob)
		return pubsub.ValidationIgnore, nil
	case pubsub.ValidationReject:
		log.WithFields(blobFields(blob)).Warning("Rejected blob: parent block is invalid")
		return pubsub.ValidationReject, nil
	default:
	}

	pubsubResult, err := s.validateBlobPostSeenParent(ctx, blob)
	if err != nil {
		return pubsubResult, err
	}
	if pubsubResult != pubsub.ValidationAccept {
		return pubsubResult, nil
	}

	startTime, err := slots.ToTime(genesisTime, blob.SignedBlockHeader.Header.Slot)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	fields := blobFields(blob)
	sinceSlotStartTime := receivedTime.Sub(startTime)
	fields["sinceSlotStartTime"] = sinceSlotStartTime
	fields["validationTime"] = s.cfg.clock.Now().Sub(receivedTime)
	log.WithFields(fields).Debug("Received blob sidecar gossip")

	blobSidecarArrivalGossipSummary.Observe(float64(sinceSlotStartTime.Milliseconds()))

	msg.ValidatorData = blob

	return pubsub.ValidationAccept, nil
}

func (s *Service) validateBlobPostSeenParent(ctx context.Context, blob *eth.BlobSidecar) (pubsub.ValidationResult, error) {
	header := blob.SignedBlockHeader.Header
	parentRoot := bytesutil.ToBytes32(header.ParentRoot)

	// [REJECT] The sidecar is from a higher slot than the sidecar's block's parent (defined by sidecar.block_parent_root).
	parentSlot, err := s.cfg.chain.RecentBlockSlot(parentRoot)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	if parentSlot >= header.Slot {
		err := fmt.Errorf("parent block slot %d greater or equal to blob slot %d", parentSlot, header.Slot)
		log.WithFields(blobFields(blob)).Debug(err)
		return pubsub.ValidationReject, err
	}

	parentState, err := s.cfg.stateGen.StateByRoot(ctx, parentRoot)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}

	// [REJECT] The proposer signature of `blob_sidecar.signed_block_header`, is valid with respect to the `block_header.proposer_index` pubkey.
	if err := blocks.VerifyBlockSignature(parentState, header.ProposerIndex, blob.SignedBlockHeader.Signature, header.HashTreeRoot); err != nil {
		return pubsub.ValidationReject, err
	}

	// TODO: The sidecar's inclusion proof is valid as verified by `verify_blob_sidecar_inclusion_proof`.

	// [IGNORE] The sidecar is the first sidecar for the tuple (block_header.slot, block_header.proposer_index, sidecar.index)
	// with valid header signature and sidecar inclusion proof
	if s.hasSeenBlobIndex(header.Slot, header.ProposerIndex, blob.Index) {
		return pubsub.ValidationIgnore, nil
	}

	// [REJECT] The sidecar is proposed by the expected proposer_index for the block's slot in the context of the current shuffling (defined by block_parent_root/slot)
	parentState, err = transition.ProcessSlotsUsingNextSlotCache(ctx, parentState, parentRoot[:], header.Slot)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	idx, err := helpers.BeaconProposerIndex(ctx, parentState)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	if header.ProposerIndex != idx {
		err := fmt.Errorf("expected proposer index %d, got %d", idx, header.ProposerIndex)
		log.WithFields(blobFields(blob)).Debug(err)
		return pubsub.ValidationReject, err
	}
	return pubsub.ValidationAccept, nil
}

// Returns true if the blob with the same root and index has been seen before.
func (s *Service) hasSeenBlobIndex(slot primitives.Slot, proposerIndex primitives.ValidatorIndex, index uint64) bool {
	s.seenBlobLock.RLock()
	defer s.seenBlobLock.RUnlock()
	b := append(bytesutil.Bytes32(uint64(slot)), bytesutil.Bytes32(uint64(proposerIndex))...)
	b = append(b, bytesutil.Bytes32(index)...)
	_, seen := s.seenBlobCache.Get(string(b))
	return seen
}

// Set blob index and root as seen.
func (s *Service) setSeenBlobIndex(slot primitives.Slot, proposerIndex primitives.ValidatorIndex, index uint64) {
	s.seenBlobLock.Lock()
	defer s.seenBlobLock.Unlock()
	b := append(bytesutil.Bytes32(uint64(slot)), bytesutil.Bytes32(uint64(proposerIndex))...)
	b = append(b, bytesutil.Bytes32(index)...)
	s.seenBlobCache.Add(string(b), true)
}

func blobFields(b *eth.BlobSidecar) logrus.Fields {
	h := b.SignedBlockHeader.Header
	return logrus.Fields{
		"slot":          h.Slot,
		"proposerIndex": h.ProposerIndex,
		"kzgCommitment": fmt.Sprintf("%#x", b.KzgCommitment),
		"index":         b.Index,
	}
}

func computeSubnetForBlobSidecar(index uint64) uint64 {
	return index % params.BeaconConfig().BlobsidecarSubnetCount
}
