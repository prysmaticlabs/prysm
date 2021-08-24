package blockchain

import (
	"context"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	core "github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Initialize the state cache for sync committees.
var syncCommitteeHeadStateCache = cache.NewSyncCommitteeHeadState()

// HeadSyncCommitteeFetcher is the interface that wraps the head sync committee related functions.
// The head sync committee functions return callers sync committee indices and public keys with respect to current head state.
type HeadSyncCommitteeFetcher interface {
	HeadCurrentSyncCommitteeIndices(ctx context.Context, index types.ValidatorIndex, slot types.Slot) ([]types.CommitteeIndex, error)
	HeadNextSyncCommitteeIndices(ctx context.Context, index types.ValidatorIndex, slot types.Slot) ([]types.CommitteeIndex, error)
	HeadSyncCommitteePubKeys(ctx context.Context, slot types.Slot, committeeIndex types.CommitteeIndex) ([][]byte, error)
}

// HeadDomainFetcher is the interface that wraps the head sync domain related functions.
// The head sync committee domain functions return callers domain data with respect to slot and head state.
type HeadDomainFetcher interface {
	HeadSyncCommitteeDomain(ctx context.Context, slot types.Slot) ([]byte, error)
	HeadSyncSelectionProofDomain(ctx context.Context, slot types.Slot) ([]byte, error)
	HeadSyncContributionProofDomain(ctx context.Context, slot types.Slot) ([]byte, error)
}

// HeadSyncCommitteeDomain returns the head sync committee domain using current head state advanced up to `slot`.
func (s *Service) HeadSyncCommitteeDomain(ctx context.Context, slot types.Slot) ([]byte, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return s.domainWithHeadState(ctx, slot, params.BeaconConfig().DomainSyncCommittee)
}

// HeadSyncSelectionProofDomain returns the head sync committee domain using current head state advanced up to `slot`.
func (s *Service) HeadSyncSelectionProofDomain(ctx context.Context, slot types.Slot) ([]byte, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return s.domainWithHeadState(ctx, slot, params.BeaconConfig().DomainSyncCommitteeSelectionProof)
}

// HeadSyncContributionProofDomain returns the head sync committee domain using current head state advanced up to `slot`.
func (s *Service) HeadSyncContributionProofDomain(ctx context.Context, slot types.Slot) ([]byte, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	return s.domainWithHeadState(ctx, slot, params.BeaconConfig().DomainContributionAndProof)
}

// HeadCurrentSyncCommitteeIndices returns the input validator `index`'s position indices in the current sync committee with respect to `slot`.
// Head state advanced up to `slot` is used for calculation.
func (s *Service) HeadCurrentSyncCommitteeIndices(ctx context.Context, index types.ValidatorIndex, slot types.Slot) ([]types.CommitteeIndex, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	headState, err := s.getSyncCommitteeHeadState(ctx, slot)
	if err != nil {
		return nil, err
	}
	return helpers.CurrentPeriodSyncSubcommitteeIndices(headState, index)
}

// HeadNextSyncCommitteeIndices returns the input validator `index`'s position indices in the next sync committee with respect to `slot`.
// Head state advanced up to `slot` is used for calculation.
func (s *Service) HeadNextSyncCommitteeIndices(ctx context.Context, index types.ValidatorIndex, slot types.Slot) ([]types.CommitteeIndex, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	headState, err := s.getSyncCommitteeHeadState(ctx, slot)
	if err != nil {
		return nil, err
	}
	return helpers.NextPeriodSyncSubcommitteeIndices(headState, index)
}

// HeadSyncCommitteePubKeys returns the head sync committee public keys with respect to `slot` and subcommittee index `committeeIndex`.
// Head state advanced up to `slot` is used for calculation.
func (s *Service) HeadSyncCommitteePubKeys(ctx context.Context, slot types.Slot, committeeIndex types.CommitteeIndex) ([][]byte, error) {
	s.headLock.RLock()
	defer s.headLock.RUnlock()

	headState, err := s.getSyncCommitteeHeadState(ctx, slot)
	if err != nil {
		return nil, err
	}

	nextSlotEpoch := helpers.SlotToEpoch(headState.Slot() + 1)
	currEpoch := helpers.SlotToEpoch(headState.Slot())

	var syncCommittee *ethpb.SyncCommittee
	if helpers.SyncCommitteePeriod(currEpoch) == helpers.SyncCommitteePeriod(nextSlotEpoch) {
		syncCommittee, err = headState.CurrentSyncCommittee()
		if err != nil {
			return nil, err
		}
	} else {
		syncCommittee, err = headState.NextSyncCommittee()
		if err != nil {
			return nil, err
		}
	}

	return altair.SyncSubCommitteePubkeys(syncCommittee, committeeIndex)
}

// returns calculated domain using input `domain` and `slot`.
func (s *Service) domainWithHeadState(ctx context.Context, slot types.Slot, domain [4]byte) ([]byte, error) {
	headState, err := s.getSyncCommitteeHeadState(ctx, slot)
	if err != nil {
		return nil, err
	}
	return helpers.Domain(headState.Fork(), helpers.SlotToEpoch(headState.Slot()), domain, headState.GenesisValidatorRoot())
}

// returns the head state that is advanced up to `slot`. It utilizes the cache `syncCommitteeHeadState` by retrieving using `slot` as key.
// For the cache miss, it processes head state up to slot and fill the cache with `slot` as key.
func (s *Service) getSyncCommitteeHeadState(ctx context.Context, slot types.Slot) (state.BeaconState, error) {
	var headState state.BeaconState
	var err error

	// If there's already a head state exists with the request slot, we don't need to process slots.
	cachedState, err := syncCommitteeHeadStateCache.Get(slot)
	if err != nil {
		return nil, err
	}
	if cachedState != nil && !cachedState.IsNil() {
		syncHeadStateHit.Inc()
		headState = cachedState
	} else {
		headState, err = s.HeadState(ctx)
		if err != nil {
			return nil, err
		}
		if slot > headState.Slot() {
			headState, err = core.ProcessSlots(ctx, headState, slot)
			if err != nil {
				return nil, err
			}
		}
		syncHeadStateMiss.Inc()
		syncCommitteeHeadStateCache.Put(slot, headState)
	}

	return headState, nil
}
