package sync

import (
	"context"
	"fmt"

	"github.com/dgraph-io/ristretto"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

var seenExitsCacheSize = int64(1 << 10)

// seenExits tracks exits we've already seen to prevent feedback loop.
var seenExits, _ = ristretto.NewCache(&ristretto.Config{
	NumCounters: seenExitsCacheSize,
	MaxCost:     seenExitsCacheSize,
	BufferItems: 64,
})

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

	ctx, span := trace.StartSpan(ctx, "sync.validateVoluntaryExit")
	defer span.End()

	exit, ok := msg.(*ethpb.VoluntaryExit)
	if !ok {
		return false, nil
	}

	cacheKey := exitCacheKey(exit)
	invalidKey := invalid + cacheKey
	if _, ok := seenExits.Get(invalidKey); ok {
		return false, errors.New("previously seen invalid validator exit received")
	}
	if _, ok := seenExits.Get(cacheKey); ok {
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
		seenExits.Set(invalidKey, true /*value*/, 1 /*cost*/)
		return false, errors.Wrap(err, "Received invalid validator exit")
	}
	seenExits.Set(cacheKey, true /*value*/, 1 /*cost*/)

	if fromSelf {
		return false, nil
	}

	if err := p.Broadcast(ctx, exit); err != nil {
		log.WithError(err).Error("Failed to propagate voluntary exit")
	}
	return true, nil
}
