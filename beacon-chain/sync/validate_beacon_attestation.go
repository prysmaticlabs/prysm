package sync

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var errPointsToBlockNotInDatabase = errors.New("attestation points to a block which is not in the database")

// validateBeaconAttestation validates that the block being voted for passes validation before forwarding to the
// network.
func (r *Service) validateBeaconAttestation(ctx context.Context, pid peer.ID, msg *pubsub.Message) bool {
	// Validation runs on publish (not just subscriptions), so we should approve any message from
	// ourselves.
	if pid == r.p2p.PeerID() {
		return true
	}

	// Attestation processing requires the target block to be present in the database, so we'll skip
	// validating or processing attestations until fully synced.
	if r.initialSync.Syncing() {
		return false
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateBeaconAttestation")
	defer span.End()

	// TODO(1332): Add blocks.VerifyAttestation before processing further.
	// Discussion: https://github.com/ethereum/eth2.0-specs/issues/1332

	m, err := r.decodePubsubMessage(msg)
	if err != nil {
		log.WithError(err).Error("Failed to decode message")
		traceutil.AnnotateError(span, err)
		return false
	}
	att, ok := m.(*ethpb.Attestation)
	if !ok {
		traceutil.AnnotateError(span, errors.New("wrong proto message type"))
		log.Error("Wrong proto message type")
		return false
	}

	span.AddAttributes(
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", att.Data.BeaconBlockRoot)),
	)

	// Only valid blocks are saved in the database.
	if !r.db.HasBlock(ctx, bytesutil.ToBytes32(att.Data.BeaconBlockRoot)) {
		r.savePendingAtt(att)
		return false
	}

	finalizedEpoch := r.chain.FinalizedCheckpt().Epoch
	attestationDataEpochOld := finalizedEpoch >= att.Data.Source.Epoch || finalizedEpoch >= att.Data.Target.Epoch
	if finalizedEpoch != 0 && attestationDataEpochOld {
		traceutil.AnnotateError(span, errors.New("wrong proto message type"))
		log.WithFields(logrus.Fields{
			"TargetEpoch": att.Data.Target.Epoch,
			"SourceEpoch": att.Data.Source.Epoch,
		}).Debug("Rejecting old attestation")
		return false
	}

	msg.ValidatorData = att // Used in downstream subscriber

	return true
}
