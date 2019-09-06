package sync

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/karlseguin/ccache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/p2p"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
)

// seenProposerSlashings represents a cache of all the seen slashings
var seenProposerSlashings = ccache.New(ccache.Configure())

func propSlashingCacheKey(slashing *ethpb.ProposerSlashing) (string, error) {
	hash, err := hashutil.HashProto(slashing)
	if err != nil {
		return "", err
	}
	return string(hash[:]), nil
}

// Clients who receive a proposer slashing on this topic MUST validate the conditions within VerifyProposerSlashing before
// forwarding it across the network.
func (r *RegularSync) validateProposerSlashing(ctx context.Context, msg proto.Message, p p2p.Broadcaster, fromSelf bool) bool {
	slashing, ok := msg.(*ethpb.ProposerSlashing)
	if !ok {
		return false
	}
	cacheKey, err := propSlashingCacheKey(slashing)
	if err != nil {
		log.WithError(err).Warn("could not hash proposer slashing")
		return false
	}

	invalidKey := invalid + cacheKey
	if seenProposerSlashings.Get(invalidKey) != nil {
		return false
	}
	if seenProposerSlashings.Get(cacheKey) != nil {
		return false
	}

	// Retrieve head state, advance state to the epoch slot used specified in slashing message.
	s := r.chain.HeadState()
	slashSlot := slashing.Header_1.Slot
	if s.Slot < slashSlot {
		var err error
		s, err = state.ProcessSlots(ctx, s, slashSlot)
		if err != nil {
			log.WithError(err).Errorf("Failed to advance state to slot %d", slashSlot)
			return false
		}
	}

	if err := blocks.VerifyProposerSlashing(s, slashing); err != nil {
		log.WithError(err).Warn("Received invalid proposer slashing")
		seenProposerSlashings.Set(invalidKey, true /*value*/, oneYear /*TTL*/)
		return false
	}
	seenProposerSlashings.Set(cacheKey, true /*value*/, oneYear /*TTL*/)

	if fromSelf {
		return false
	}

	if err := p.Broadcast(ctx, slashing); err != nil {
		log.WithError(err).Error("Failed to propagate proposer slashing")
	}
	return true
}
