package sync

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/karlseguin/ccache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// seenExits tracks exits we've already seen to prevent feedback loop.
var seenExits = ccache.New(ccache.Configure())

func exitCacheKey(exit *ethpb.VoluntaryExit) string {
	return fmt.Sprintf("%d-%d", exit.Epoch, exit.ValidatorIndex)
}

// Clients who receive a voluntary exit on this topic MUST validate the conditions within process_voluntary_exit before
// forwarding it across the network.
func (r *RegularSync) validateVoluntaryExit(ctx context.Context, msg proto.Message, p p2p.Broadcaster, fromSelf bool) bool {
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

	// Retrieve head state, advance state to the epoch slot used specified in exit message.
	s := r.chain.HeadState()
	exitedEpochSlot := exit.Epoch * params.BeaconConfig().SlotsPerEpoch
	if s.Slot < exitedEpochSlot {
		var err error
		s, err = state.ProcessSlots(ctx, s, exitedEpochSlot)
		if err != nil {
			log.WithError(err).Errorf("Failed to advance state to slot %d", exitedEpochSlot)
			return false
		}
	}

	if err := blocks.VerifyExit(s, exit); err != nil {
		log.WithError(err).Warn("Received invalid voluntary exit")
		seenExits.Set(invalidKey, true /*value*/, oneYear /*TTL*/)
		return false
	}
	seenExits.Set(cacheKey, true /*value*/, oneYear /*TTL*/)

	if fromSelf {
		return false
	}

	if err := p.Broadcast(ctx, exit); err != nil {
		log.WithError(err).Error("Failed to propagate voluntary exit")
	}
	return true
}
