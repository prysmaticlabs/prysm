package sync

import (
	"context"
	"fmt"
	"strings"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/config/params"
	"github.com/prysmaticlabs/prysm/v4/crypto/bls"
	"github.com/prysmaticlabs/prysm/v4/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/v4/network/forks"
	eth "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
	"github.com/sirupsen/logrus"
)

func (s *Service) validateBlob(ctx context.Context, pid peer.ID, msg *pubsub.Message) (pubsub.ValidationResult, error) {
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
		log.WithFields(blobFields(blob)).Error("Sidecar blob does not match topic")
		return pubsub.ValidationReject, fmt.Errorf("wrong topic name: %s", *msg.Topic)
	}

	// [IGNORE] The sidecar is not from a future slot (with a MAXIMUM_GOSSIP_CLOCK_DISPARITY allowance) --
	// i.e. validate that sidecar.slot <= current_slot (a client MAY queue future blocks for processing at the appropriate slot).
	genesisTime := uint64(s.cfg.chain.GenesisTime().Unix())
	if err := slots.VerifyTime(genesisTime, blob.Slot, earlyBlockProcessingTolerance); err != nil {
		log.WithError(err).WithFields(blobFields(blob)).Error("Ignored blob: too far into future")
		return pubsub.ValidationIgnore, err
	}

	// [IGNORE] The sidecar is from a slot greater than the latest finalized slot --
	// i.e. validate that sidecar.slot > compute_start_slot_at_epoch(state.finalized_checkpoint.epoch)
	startSlot, err := slots.EpochStart(s.cfg.chain.FinalizedCheckpt().Epoch)
	if err != nil {
		log.WithError(err).WithFields(blobFields(blob)).Error("Ignored block: could not calculate epoch start slot")
		return pubsub.ValidationIgnore, err
	}
	if startSlot >= blob.Slot {
		err := fmt.Errorf("finalized slot %d greater or equal to blob slot %d", startSlot, blob.Slot)
		log.WithFields(blobFields(blob)).Warn(err)
		return pubsub.ValidationIgnore, err
	}

	// [IGNORE] The blob's block's parent (defined by sidecar.block_parent_root) has been seen (via both gossip and non-gossip sources)
	parentRoot := bytesutil.ToBytes32(blob.BlockParentRoot)
	if !s.cfg.chain.HasBlock(ctx, parentRoot) {
		if err := s.blockAndBlobs.addBlob(sBlob.Message); err != nil {
			log.WithError(err).WithFields(blobFields(blob)).Error("Failed to add blob to queue")
			return pubsub.ValidationIgnore, err
		}
		log.WithFields(blobFields(blob)).Warn("Ignored blob: parent block not found")
		return pubsub.ValidationIgnore, nil
	}

	// [REJECT] The sidecar's block's parent (defined by sidecar.block_parent_root) passes validation.
	// TODO: I'm not sure how to deal with this special case.

	// [REJECT] The sidecar is from a higher slot than the sidecar's block's parent (defined by sidecar.block_parent_root).
	blk, err := s.cfg.beaconDB.Block(ctx, parentRoot)
	if err != nil {
		log.WithError(err).WithFields(blobFields(blob)).Error("Failed to get parent block")
		return pubsub.ValidationIgnore, err
	}
	if blk.Block().Slot() >= blob.Slot {
		err := fmt.Errorf("parent block slot %d greater or equal to blob slot %d", blk.Block().Slot(), blob.Slot)
		log.WithFields(blobFields(blob)).Error(err)
		return pubsub.ValidationReject, err
	}

	// [REJECT] The proposer signature, signed_blob_sidecar.signature,
	// is valid with respect to the sidecar.proposer_index pubkey.
	parentState, err := s.cfg.stateGen.StateByRoot(ctx, parentRoot)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	if err := verifyBlobSignature(parentState, sBlob); err != nil {
		log.WithError(err).WithFields(blobFields(blob)).Error("Failed to verify blob signature")
		return pubsub.ValidationReject, err
	}

	// [IGNORE] The sidecar is the only sidecar with valid signature received for the tuple (sidecar.block_root, sidecar.index).
	blockRoot := bytesutil.ToBytes32(blob.BlockRoot)
	b, err := s.blockAndBlobs.getBlob(blockRoot, blob.Index)
	if err == nil || b != nil {
		log.WithFields(blobFields(blob)).Warn("Ignored blob: blob already exists")
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
		log.WithFields(blobFields(blob)).Error(err)
		return pubsub.ValidationReject, err
	}

	msg.ValidatorData = sBlob

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

func blobFields(b *eth.BlobSidecar) logrus.Fields {
	return logrus.Fields{
		"slot":          b.Slot,
		"proposerIndex": b.ProposerIndex,
		"blockRoot":     fmt.Sprintf("%#x", b.BlockRoot),
		"index":         b.Index,
	}
}
