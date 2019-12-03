package sync

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/karlseguin/ccache"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// seenExits tracks exits we've already seen to prevent feedback loop.
var seenExits = ccache.New(ccache.Configure())

func exitCacheKey(exit *ethpb.VoluntaryExit) string {
	return fmt.Sprintf("%d-%d", exit.Epoch, exit.ValidatorIndex)
}

// Clients who receive a voluntary exit on this topic MUST validate the conditions within process_voluntary_exit before
// forwarding it across the network.
func (r *RegularSync) validateVoluntaryExit(ctx context.Context, msg proto.Message, p p2p.Broadcaster, fromSelf bool) (bool, error) {
	// The head state will be too far away to validate any voluntary exit.
	if r.initialSync.Syncing() {
		return false, nil
	}

	exit, ok := msg.(*ethpb.VoluntaryExit)
	if !ok {
		return false, nil
	}
	cacheKey := exitCacheKey(exit)
	invalidKey := invalid + cacheKey
	if seenExits.Get(invalidKey) != nil {
		return false, errors.New("previously seen invalid validator exit received")
	}
	if seenExits.Get(cacheKey) != nil {
		return false, nil
	}

	// Retrieve head state, advance state to the epoch slot used specified in exit message.
	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return false, err
	}

	exitedEpochSlot := exit.Epoch * params.BeaconConfig().SlotsPerEpoch
	if s.Slot < exitedEpochSlot {
		var err error
		s, err = state.ProcessSlots(ctx, s, exitedEpochSlot)
		if err != nil {
			return false, errors.Wrapf(err, "Failed to advance state to slot %d", exitedEpochSlot)
		}
	}

	if err := blocks.VerifyExit(s, exit); err != nil {
		seenExits.Set(invalidKey, true /*value*/, oneYear /*TTL*/)
		return false, errors.Wrap(err, "Received invalid validator exit")
	}
	seenExits.Set(cacheKey, true /*value*/, oneYear /*TTL*/)

	if fromSelf {
		return false, nil
	}

	if err := p.Broadcast(ctx, exit); err != nil {
		log.WithError(err).Error("Failed to propagate voluntary exit")
	}
	return true, nil
}
