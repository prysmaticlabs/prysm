package sync

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	kbls "github.com/protolambda/go-kzg/bls"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	"github.com/prysmaticlabs/prysm/monitoring/tracing"
	"github.com/prysmaticlabs/prysm/network/forks"
	enginev1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/time/slots"
	"go.opencensus.io/trace"
)

// Gossip Validation Conditions:
// [IGNORE] the sidecar.beacon_block_slot is for the current slot (with a MAXIMUM_GOSSIP_CLOCK_DISPARITY allowance)
//  -- i.e. blobs_sidecar.beacon_block_slot == current_slot.
// [REJECT] the sidecar.blobs are all well formatted, i.e. the BLSFieldElement in valid range (x < BLS_MODULUS).
// [REJECT] the beacon proposer signature, signed_blobs_sidecar.signature, is valid
// [IGNORE] The sidecar is the first sidecar with valid signature received for the (proposer_index, sidecar.beacon_block_slot)
// combination, where proposer_index is the validator index of the beacon block proposer of blobs_sidecar.beacon_block_slot
func (s *Service) validateBlobsSidecar(ctx context.Context, pid peer.ID, msg *pubsub.Message) (pubsub.ValidationResult, error) {
	ctx, span := trace.StartSpan(ctx, "sync.validateBlobsSidecar")
	defer span.End()

	// Accept the blob if it came from itself.
	if pid == s.cfg.p2p.PeerID() {
		return pubsub.ValidationAccept, nil
	}

	// Ignore the blobs sidecar if the beacon node is syncing.
	if s.cfg.initialSync.Syncing() {
		return pubsub.ValidationIgnore, nil
	}

	// TODO(inphi): Handle received blobs when optimistic.

	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationReject, errors.Wrap(err, "Could not decode message")
	}

	signed, ok := m.(*ethpb.SignedBlobsSidecar)
	if !ok {
		return pubsub.ValidationReject, errWrongMessage
	}
	if signed.Message == nil {
		return pubsub.ValidationReject, errors.New("nil blob message")
	}
	if signed.Signature == nil {
		return pubsub.ValidationReject, errors.New("nil blob signature")
	}

	if err := altair.ValidateSyncMessageTime(signed.Message.GetBeaconBlockSlot(), s.cfg.chain.GenesisTime(), params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
		tracing.AnnotateError(span, err)
		return pubsub.ValidationIgnore, err
	}

	blobs := signed.Message.GetBlobs()
	if err := validateBlobFr(blobs); err != nil {
		return pubsub.ValidationReject, err
	}

	blk, err := s.cfg.beaconDB.Block(ctx, bytesutil.ToBytes32(signed.Message.BeaconBlockRoot))
	if err != nil {
		return pubsub.ValidationIgnore, errors.Wrap(err, "Could not retrieve block")
	}

	validationResult, err := s.validateBlobsSidecarSignature(ctx, blk, signed)
	if err != nil {
		tracing.AnnotateError(span, err)
		return validationResult, err
	}

	if s.hasSeenBlobsSidecarIndexSlot(blk.Block().ProposerIndex(), signed.Message.BeaconBlockSlot) {
		return pubsub.ValidationIgnore, nil
	}

	s.setBlobsSidecarIndexSlotSeen(blk.Block().ProposerIndex(), signed.Message.BeaconBlockSlot)
	msg.ValidatorData = signed

	return pubsub.ValidationAccept, nil
}

func (s *Service) validateBlobsSidecarSignature(ctx context.Context, blk interfaces.SignedBeaconBlock, m *ethpb.SignedBlobsSidecar) (pubsub.ValidationResult, error) {
	currentEpoch := slots.ToEpoch(m.Message.BeaconBlockSlot)
	fork, err := forks.Fork(currentEpoch)
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	state, err := s.cfg.stateGen.StateByRoot(ctx, bytesutil.ToBytes32(m.Message.BeaconBlockRoot))
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	proposer, err := state.ValidatorAtIndex(blk.Block().ProposerIndex())
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	proposerPubKey := proposer.PublicKey
	blobSigning := &ethpb.BlobsSidecar{
		BeaconBlockRoot: m.Message.BeaconBlockRoot,
		BeaconBlockSlot: m.Message.BeaconBlockSlot,
		Blobs:           m.Message.Blobs,
	}

	domain, err := signing.Domain(fork, currentEpoch, params.BeaconConfig().DomainBlobsSidecar, state.GenesisValidatorsRoot())
	if err != nil {
		return pubsub.ValidationReject, err
	}
	pKey, err := bls.PublicKeyFromBytes(proposerPubKey[:])
	sigRoot, err := signing.ComputeSigningRoot(blobSigning, domain)
	if err != nil {
		return pubsub.ValidationReject, err
	}

	set := &bls.SignatureBatch{
		Messages:   [][32]byte{sigRoot},
		PublicKeys: []bls.PublicKey{pKey},
		Signatures: [][]byte{m.Signature},
	}
	return s.validateWithBatchVerifier(ctx, "blobs sidecar signature", set)
}

func validateBlobFr(blobs []*enginev1.Blob) error {
	for _, blob := range blobs {
		for _, b := range blob.Blob {
			if len(b) != 32 {
				return errors.New("invalid blob field element size")
			}
			if !kbls.ValidFr(bytesutil.ToBytes32(b)) {
				return errors.New("invalid blob field element")
			}
		}
	}
	return nil
}

func (s *Service) hasSeenBlobsSidecarIndexSlot(proposerIndex types.ValidatorIndex, slot types.Slot) bool {
	s.seenBlobsSidecarLock.RLock()
	defer s.seenBlobsSidecarLock.RUnlock()

	b := append(bytesutil.Bytes32(uint64(proposerIndex)), bytesutil.Bytes32(uint64(slot))...)
	_, seen := s.seenBlobsSidecarCache.Get(string(b))
	return seen
}

func (s *Service) setBlobsSidecarIndexSlotSeen(proposerIndex types.ValidatorIndex, slot types.Slot) {
	s.seenBlobsSidecarLock.RLock()
	defer s.seenBlobsSidecarLock.RUnlock()

	b := append(bytesutil.Bytes32(uint64(proposerIndex)), bytesutil.Bytes32(uint64(slot))...)
	s.seenBlobsSidecarCache.Add(string(b), true)
}
