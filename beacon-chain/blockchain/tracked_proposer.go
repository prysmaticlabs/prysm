package blockchain

import (
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v4/time/slots"
)

// trackedProposer returns whether the beacon node was informed, via the
// validators/prepare_proposer endpoint, of the proposer at the given slot.
// It only returns true if the tracked proposer is present and active.
func (s *Service) trackedProposer(st state.ReadOnlyBeaconState, slot primitives.Slot) (cache.TrackedValidator, bool) {
	// if the head state and slot are in different epochs, try first the
	// precomputed cache one epoch in advance
	var id primitives.ValidatorIndex
	var err error
	stateEpoch := slots.ToEpoch(st.Slot())
	e := slots.ToEpoch(slot)
	if stateEpoch != e {
		id, err = helpers.UnsafeBeaconProposerIndexAtSlot(st, slot)
	}
	if err != nil || stateEpoch == e {
		id, err = helpers.BeaconProposerIndexAtSlot(s.ctx, st, slot)
		if err != nil {
			return cache.TrackedValidator{}, false
		}
	}
	val, ok := s.cfg.TrackedValidatorsCache.Validator(id)
	if !ok {
		return cache.TrackedValidator{}, false
	}
	return val, val.Active
}
