package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/karlseguin/ccache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// seenAttesterSlashings represents a cache of all the seen slashings
var seenAttesterSlashings = ccache.New(ccache.Configure())

func attSlashingCacheKey(slashing *ethpb.AttesterSlashing) (string, error) {
	hash, err := hashutil.HashProto(slashing)
	if err != nil {
		return "", err
	}
	return string(hash[:]), nil
}

// Clients who receive an attester slashing on this topic MUST validate the conditions within VerifyAttesterSlashing before
// forwarding it across the network.
func (r *RegularSync) validateAttesterSlashing(ctx context.Context, msg proto.Message, p p2p.Broadcaster) bool {
	slashing, ok := msg.(*ethpb.AttesterSlashing)
	if !ok {
		return false
	}
	cacheKey, err := attSlashingCacheKey(slashing)
	if err != nil {
		log.WithError(err).Warn("could not hash attester slashing")
		return false
	}

	invalidKey := invalid + cacheKey
	if seenAttesterSlashings.Get(invalidKey) != nil {
		return false
	}
	if seenAttesterSlashings.Get(cacheKey) != nil {
		return false
	}
	state, err := r.db.HeadState(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get head state")
		return false
	}

	if err := blocks.VerifyAttesterSlashing(state, slashing); err != nil {
		log.WithError(err).Warn("Received invalid attester slashing")
		seenAttesterSlashings.Set(invalidKey, true /*value*/, oneYear /*TTL*/)
		return false
	}
	seenAttesterSlashings.Set(cacheKey, true /*value*/, oneYear /*TTL*/)

	if err := p.Broadcast(ctx, slashing); err != nil {
		log.WithError(err).Error("Failed to propagate attester slashing")
	}
	return true
}
