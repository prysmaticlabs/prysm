package sync

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/karlseguin/ccache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

// seenExits tracks exits we've already seen to prevent feedback loop.
var seenExits = ccache.New(ccache.Configure())

func exitCacheKey(exit *ethpb.VoluntaryExit) string {
	return fmt.Sprintf("%d-%d", exit.Epoch, exit.ValidatorIndex)
}

// Clients who receive a voluntary exit on this topic MUST validate the conditions within process_voluntary_exit before
// forwarding it across the network.
func (r *RegularSync) validateVoluntaryExit(ctx context.Context, msg proto.Message, p p2p.Broadcaster) bool {
	exit, ok := msg.(*ethpb.VoluntaryExit)
	if !ok {
		return false
	}
	cacheKey := exitCacheKey(exit)
	invalidKey := invalid + cacheKey
	if seenExits.Get(invalidKey) != nil {
		return false
	}
	if seenExits.Get(cacheKey) != nil {
		return false
	}
	state, err := r.db.HeadState(ctx)
	if err != nil {
		log.WithError(err).Error("Failed to get head state")
		return false
	}
	if err := blocks.VerifyExit(state, exit); err != nil {
		log.WithError(err).Warn("Received invalid voluntary exit")
		seenExits.Set(invalidKey, true /*value*/, oneYear /*TTL*/)
		return false
	}
	seenExits.Set(cacheKey, true /*value*/, oneYear /*TTL*/)

	if err := p.Broadcast(ctx, exit); err != nil {
		log.WithError(err).Error("Failed to propagate voluntary exit")
	}
	return true
}
