package sync

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var errPointsToBlockNotInDatabase = errors.New("attestation points to a block which is not in the database")

// validateBeaconAttestation validates that the block being voted for passes validation before forwarding to the
// network.
func (r *Service) validateBeaconAttestation(ctx context.Context, pid peer.ID, msg *pubsub.Message) bool {
	// Attestation processing requires the target block to be present in the database, so we'll skip
	// validating or processing attestations until fully synced.
	if r.initialSync.Syncing() {
		return false
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateBeaconAttestation")
	defer span.End()

	// TODO(1332): Add blocks.VerifyAttestation before processing further.
	// Discussion: https://github.com/ethereum/eth2.0-specs/issues/1332

	if msg == nil || msg.TopicIDs == nil || len(msg.TopicIDs) == 0 {
		return false
	}
	topic := msg.TopicIDs[0]
	topic = strings.TrimSuffix(topic, r.p2p.Encoding().ProtocolSuffix())
	base, ok := p2p.GossipTopicMappings[topic]
	if !ok {
		return false
	}
	m := proto.Clone(base)
	if err := r.p2p.Encoding().Decode(msg.Data, m); err != nil {
		traceutil.AnnotateError(span, err)
		log.WithError(err).Warn("Failed to decode pubsub message")
		return false
	}

	att, ok := m.(*ethpb.Attestation)
	if !ok {
		return false
	}

	span.AddAttributes(
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", att.Data.BeaconBlockRoot)),
	)

	// Only valid blocks are saved in the database.
	if !r.db.HasBlock(ctx, bytesutil.ToBytes32(att.Data.BeaconBlockRoot)) {
		log.WithField(
			"blockRoot",
			fmt.Sprintf("%#x", att.Data.BeaconBlockRoot),
		).WithError(errPointsToBlockNotInDatabase).Debug("Ignored incoming attestation that points to a block which is not in the database")
		traceutil.AnnotateError(span, errPointsToBlockNotInDatabase)
		return false
	}

	if pid == r.p2p.PeerID() {
		return false
	}

	finalizedEpoch := r.chain.FinalizedCheckpt().Epoch
	attestationDataEpochOld := finalizedEpoch >= att.Data.Source.Epoch || finalizedEpoch >= att.Data.Target.Epoch
	if finalizedEpoch != 0 && attestationDataEpochOld {
		log.WithFields(logrus.Fields{
			"TargetEpoch": att.Data.Target.Epoch,
			"SourceEpoch": att.Data.Source.Epoch,
		}).Debug("Rejecting old attestation")
		return false
	}

	msg.VaidatorData = att // Used in downstream subscriber

	return true
}
