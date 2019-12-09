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
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

var seenAttSlashingCacheSize = int64(1 << 10)

// seenAttesterSlashings represents a cache of all the seen slashings
var seenAttesterSlashings, _ = ristretto.NewCache(&ristretto.Config{
	NumCounters: seenAttSlashingCacheSize,
	MaxCost:     seenAttSlashingCacheSize,
	BufferItems: 64,
})

func attSlashingCacheKey(slashing *ethpb.AttesterSlashing) (string, error) {
	hash, err := hashutil.HashProto(slashing)
	if err != nil {
		return "", err
	}
	return string(hash[:]), nil
}

// Clients who receive an attester slashing on this topic MUST validate the conditions within VerifyAttesterSlashing before
// forwarding it across the network.
func (r *RegularSync) validateAttesterSlashing(ctx context.Context, msg proto.Message, p p2p.Broadcaster, fromSelf bool) (bool, error) {
	// The head state will be too far away to validate any slashing.
	if r.initialSync.Syncing() {
		return false, nil
	}

	ctx, span := trace.StartSpan(ctx, "sync.validateAttesterSlashing")
	defer span.End()

	slashing, ok := msg.(*ethpb.AttesterSlashing)
	if !ok {
		return false, fmt.Errorf("message was not type *eth.AttesterSlashing, type=%T", msg)
	}

	cacheKey, err := attSlashingCacheKey(slashing)
	if err != nil {
		return false, errors.Wrapf(err, "could not hash attestation slashing")
	}

	invalidKey := invalid + cacheKey
	if _, ok := seenAttesterSlashings.Get(invalidKey); ok {
		return false, errors.New("previously seen invalid attester slashing received")
	}
	if _, ok := seenAttesterSlashings.Get(cacheKey); ok {
		return false, nil
	}

	// Retrieve head state, advance state to the epoch slot used specified in slashing message.
	s, err := r.chain.HeadState(ctx)
	if err != nil {
		return false, err
	}
	slashSlot := slashing.Attestation_1.Data.Target.Epoch * params.BeaconConfig().SlotsPerEpoch
	if s.Slot < slashSlot {
		if ctx.Err() != nil {
			return false, errors.Wrapf(ctx.Err(),
				"Failed to advance state to slot %d to process attester slashing", slashSlot)
		}

		var err error
		s, err = state.ProcessSlots(ctx, s, slashSlot)
		if err != nil {
			return false, errors.Wrapf(err, "Failed to advance state to slot %d", slashSlot)
		}
	}

	if err := blocks.VerifyAttesterSlashing(ctx, s, slashing); err != nil {
		seenAttesterSlashings.Set(invalidKey, true /*value*/, 1 /*cost*/)
		return false, errors.Wrap(err, "Received invalid attester slashing")
	}
	seenAttesterSlashings.Set(cacheKey, true /*value*/, 1 /*cost*/)

	if fromSelf {
		return false, nil
	}

	if err := p.Broadcast(ctx, slashing); err != nil {
		log.WithError(err).Error("Failed to propagate attester slashing")
	}
	return true, nil
}
