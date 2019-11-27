package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/traceutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var errPointsToBlockNotInDatabase = errors.New("attestation points to a block which is not in the database")

// validateBeaconAttestation validates that the block being voted for passes validation before forwarding to the
// network.
func (r *RegularSync) validateBeaconAttestation(ctx context.Context, msg proto.Message, p p2p.Broadcaster, fromSelf bool) (bool, error) {
	// Attestation processing requires the target block to be present in the database, so we'll skip
	// validating or processing attestations until fully synced.
	if r.initialSync.Syncing() {
		return false, nil
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateBeaconAttestation")
	defer span.End()

	// TODO(1332): Add blocks.VerifyAttestation before processing further.
	// Discussion: https://github.com/ethereum/eth2.0-specs/issues/1332

	att := msg.(*ethpb.Attestation)

	attRoot, err := ssz.HashTreeRoot(att)
	if err != nil {
		return false, errors.Wrap(err, "could not hash attestation")
	}

	span.AddAttributes(
		trace.StringAttribute("blockRoot", fmt.Sprintf("%#x", att.Data.BeaconBlockRoot)),
		trace.StringAttribute("attRoot", fmt.Sprintf("%#x", attRoot)),
	)

	// Only valid blocks are saved in the database.
	if !r.db.HasBlock(ctx, bytesutil.ToBytes32(att.Data.BeaconBlockRoot)) {
		log.WithField(
			"blockRoot",
			fmt.Sprintf("%#x", att.Data.BeaconBlockRoot),
		).WithError(errPointsToBlockNotInDatabase).Debug("Ignored incoming attestation that points to a block which is not in the database")
		traceutil.AnnotateError(span, errPointsToBlockNotInDatabase)
		return false, nil
	}

	if recentlySeenRoots.Get(string(attRoot[:])) != nil {
		return false, nil
	}

	recentlySeenRoots.Set(string(attRoot[:]), true /*value*/, 365*24*time.Hour /*TTL*/)

	if fromSelf {
		return false, nil
	}

	finalizedEpoch := r.chain.FinalizedCheckpt().Epoch
	attestationDataEpochOld := finalizedEpoch >= att.Data.Source.Epoch || finalizedEpoch >= att.Data.Target.Epoch
	if finalizedEpoch != 0 && attestationDataEpochOld {
		log.WithFields(logrus.Fields{
			"AttestationRoot": fmt.Sprintf("%#x", attRoot),
			"TargetEpoch":     att.Data.Target.Epoch,
			"SourceEpoch":     att.Data.Source.Epoch,
		}).Debug("Rejecting old attestation")
		return false, nil
	}

	if err := p.Broadcast(ctx, msg); err != nil {
		log.WithError(err).Error("Failed to broadcast message")
		traceutil.AnnotateError(span, err)
	}
	return true, nil
}
