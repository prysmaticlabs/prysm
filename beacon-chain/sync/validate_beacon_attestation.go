package sync

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

// validateBeaconAttestation validates that the block being voted for passes validation before forwarding to the
// network.
func (r *RegularSync) validateBeaconAttestation(ctx context.Context, msg proto.Message, p p2p.Broadcaster, fromSelf bool) bool {
	att := msg.(*ethpb.Attestation)

	attRoot, err := ssz.HashTreeRoot(att)
	if err != nil {
		log.WithError(err).Error("Failed to hash attestation")
	}

	// Only valid blocks are saved in the database.
	if !r.db.HasBlock(ctx, bytesutil.ToBytes32(att.Data.BeaconBlockRoot)) {
		log.Warn("Ignored incoming attestation that points to a block that isn't in the database.")
		return false
	}

	if recentlySeenRoots.Get(string(attRoot[:])) != nil {
		return false
	}

	recentlySeenRoots.Set(string(attRoot[:]), true /*value*/, 365*24*time.Hour /*TTL*/)

	if fromSelf {
		return false
	}

	if err := p.Broadcast(ctx, msg); err != nil {
		log.WithError(err).Error("Failed to broadcast message")
	}
	return true
}
