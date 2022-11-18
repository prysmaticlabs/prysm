package sync

import (
	"context"

	"github.com/ethereum/go-ethereum/crypto/kzg"
	gethParams "github.com/ethereum/go-ethereum/params"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	kbls "github.com/protolambda/go-kzg/bls"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	eth "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
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
	signed, ok := m.(*eth.SignedBeaconBlockAndBlobsSidecar)
	if !ok {
		return pubsub.ValidationReject, errWrongMessage
	}

	result, err := s.validateBlockPubsubHelper(ctx, receivedTime, msg, signed.BeaconBlock)
	if err != nil || result != pubsub.ValidationAccept {
		return result, err
	}

	err = s.validateBeaconBlockKzgs(signed.BeaconBlock.Block)
	if err != nil {
		// TODO(EIP4844): Differentiate better between ignore and reject
		return pubsub.ValidationReject, err
	}

	err = s.validateBlobsSidecar(signed.BlobsSidecar)
	if err != nil {
		return pubsub.ValidationReject, err
	}

	return pubsub.ValidationAccept, nil
}

func (s *Service) validateBeaconBlockKzgs(blk *eth.BeaconBlockCapella) error {
	body := blk.Body
	payload := body.ExecutionPayload
	if payload == nil {
		return errors.New("execution payload is nil")
	}

	blobKzgs := body.BlobKzgCommitments
	blobKzgsInput := make(kzg.KZGCommitmentSequenceImpl, len(blobKzgs))
	for i := range blobKzgs {
		blobKzgsInput[i] = bytesutil.ToBytes48(blobKzgs[i])
	}

	txs := payload.Transactions
	return kzg.VerifyKZGCommitmentsAgainstTransactions(txs, blobKzgsInput)
}

func (s *Service) validateBlobsSidecar(b *eth.BlobsSidecar) error {
	if err := altair.ValidateSyncMessageTime(b.BeaconBlockSlot, s.cfg.chain.GenesisTime(), params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
		return err
	}
	if err := validateBlobFr(b.Blobs); err != nil {
		log.WithError(err).WithField("slot", b.BeaconBlockSlot).Warn("Sidecar contains invalid BLS field elements")
		return err
	}

	// TODO(EIP4844): The KZG proof is a correctly encoded compressed BLS G1 Point -- i.e. bls.KeyValidate(blobs_sidecar.kzg_aggregated_proof)

	return nil
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
