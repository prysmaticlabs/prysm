package sync

import (
	"context"

	gethParams "github.com/ethereum/go-ethereum/params"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	kbls "github.com/protolambda/go-kzg/bls"
	"github.com/protolambda/go-kzg/eth"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blobs"
	"github.com/prysmaticlabs/prysm/v3/crypto/bls"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	prysmTime "github.com/prysmaticlabs/prysm/v3/time"
	"go.opencensus.io/trace"
)

// validateBeaconBlockPubSub checks that the incoming block has a valid BLS signature.
// Blocks that have already been seen are ignored. If the BLS signature is any valid signature,
// this method rebroadcasts the message.
func (s *Service) validateBeaconBlockAndBlobsPubSub(ctx context.Context, pid peer.ID, msg *pubsub.Message) (pubsub.ValidationResult, error) {
	ctx, span := trace.StartSpan(ctx, "sync.validateBeaconBlockPubSub")
	defer span.End()

	receivedTime := prysmTime.Now()

	if pid == s.cfg.p2p.PeerID() {
		return pubsub.ValidationAccept, nil
	}
	if s.cfg.initialSync.Syncing() {
		return pubsub.ValidationIgnore, nil
	}
	m, err := s.decodePubsubMessage(msg)
	if err != nil {
		return pubsub.ValidationReject, errors.Wrap(err, "Could not decode message")
	}
	signed, ok := m.(*ethpb.SignedBeaconBlockAndBlobsSidecar)
	if !ok {
		return pubsub.ValidationReject, errWrongMessage
	}

	sb := signed.BeaconBlock
	result, err := s.validateBlockPubsubHelper(ctx, receivedTime, msg, sb)
	if err != nil || result != pubsub.ValidationAccept {
		return result, err
	}

	b := sb.Block
	err = s.validateBeaconBlockKzgs(b)
	if err != nil {
		return pubsub.ValidationReject, err
	}

	sc := signed.BlobsSidecar
	status, err := s.validateBlobsSidecar(signed.BlobsSidecar)
	if err != nil {
		return status, err
	}

	// [REJECT] The KZG commitments in the block are valid against the provided blobs sidecar.
	r, err := b.HashTreeRoot()
	if err != nil {
		return pubsub.ValidationIgnore, err
	}
	if err := blobs.ValidateBlobsSidecar(b.Slot, r, b.Body.BlobKzgCommitments, sc); err != nil {
		return pubsub.ValidationReject, err
	}

	return pubsub.ValidationAccept, nil
}

func (s *Service) validateBeaconBlockKzgs(blk *ethpb.BeaconBlock4844) error {
	body := blk.Body
	payload := body.ExecutionPayload
	if payload == nil {
		return errors.New("execution payload is nil")
	}

	blobKzgs := body.BlobKzgCommitments
	blobKzgsInput := make(eth.KZGCommitmentSequenceImpl, len(blobKzgs))
	for i := range blobKzgs {
		// [REJECT] The KZG commitments of the blobs are all correctly encoded compressed BLS G1 Points.
		_, err := bls.PublicKeyFromBytes(blobKzgs[i])
		if err != nil {
			return errors.Wrap(err, "invalid blob kzg public key")
		}
		blobKzgsInput[i] = bytesutil.ToBytes48(blobKzgs[i])
	}

	// [REJECT] The KZG commitments correspond to the versioned hashes in the transactions list.
	txs := payload.Transactions
	return eth.VerifyKZGCommitmentsAgainstTransactions(txs, blobKzgsInput)
}

func (s *Service) validateBlobsSidecar(b *ethpb.BlobsSidecar) (pubsub.ValidationResult, error) {
	// [IGNORE] the sidecar.beacon_block_slot is for the current slot (with a MAXIMUM_GOSSIP_CLOCK_DISPARITY allowance)
	if err := altair.ValidateSyncMessageTime(b.BeaconBlockSlot, s.cfg.chain.GenesisTime(), params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
		return pubsub.ValidationIgnore, err
	}

	// [REJECT] the sidecar.blobs are all well formatted, i.e. the BLSFieldElement in valid range
	if err := validateBlobFr(b.Blobs); err != nil {
		log.WithError(err).WithField("slot", b.BeaconBlockSlot).Warn("Sidecar contains invalid BLS field elements")
		return pubsub.ValidationReject, err
	}

	// [REJECT] The KZG proof is a correctly encoded compressed BLS G1 Point
	_, err := bls.PublicKeyFromBytes(b.AggregatedProof)
	if err != nil {
		return pubsub.ValidationReject, errors.Wrap(err, "invalid blob kzg public key")
	}

	return pubsub.ValidationAccept, nil
}

func validateBlobFr(blobs []*enginev1.Blob) error {
	for _, blob := range blobs {
		fe := gethParams.FieldElementsPerBlob
		if len(blob.Data) != fe*32 {
			return errors.New("Incorrect field element length")
		}
		for i := 0; i < fe; i++ {
			b := bytesutil.ToBytes32(blob.Data[i*32 : i*32+31])
			if !kbls.ValidFr(b) {
				return errors.New("invalid blob field element")
			}
		}
	}
	return nil
}
