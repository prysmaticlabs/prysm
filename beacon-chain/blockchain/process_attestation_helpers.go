package blockchain

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/async"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/blocks"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
)

// getAttPreState retrieves the att pre state by either from the cache or the DB.
func (s *Service) getAttPreState(ctx context.Context, c *ethpb.Checkpoint) (state.BeaconState, error) {
	// Use a multilock to allow scoped holding of a mutex by a checkpoint root + epoch
	// allowing us to behave smarter in terms of how this function is used concurrently.
	epochKey := strconv.FormatUint(uint64(c.Epoch), 10 /* base 10 */)
	lock := async.NewMultilock(string(c.Root) + epochKey)
	lock.Lock()
	defer lock.Unlock()
	cachedState, err := s.checkpointStateCache.StateByCheckpoint(c)
	if err != nil {
		return nil, errors.Wrap(err, "could not get cached checkpoint state")
	}
	if cachedState != nil && !cachedState.IsNil() {
		return cachedState, nil
	}

	baseState, err := s.cfg.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(c.Root))
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pre state for epoch %d", c.Epoch)
	}

	epochStartSlot, err := slots.EpochStart(c.Epoch)
	if err != nil {
		return nil, err
	}
	baseState, err = transition.ProcessSlotsIfPossible(ctx, baseState, epochStartSlot)
	if err != nil {
		return nil, errors.Wrapf(err, "could not process slots up to epoch %d", c.Epoch)
	}

	// Sharing the same state across caches is perfectly fine here, the fetching
	// of attestation prestate is by far the most accessed state fetching pattern in
	// the beacon node. An extra state instance cached isn't an issue in the bigger
	// picture.
	if err := s.checkpointStateCache.AddCheckpointState(c, baseState); err != nil {
		return nil, errors.Wrap(err, "could not save checkpoint state to cache")
	}
	return baseState, nil

}

// verifyAttTargetEpoch validates attestation is from the current or previous epoch.
func verifyAttTargetEpoch(_ context.Context, genesisTime, nowTime uint64, c *ethpb.Checkpoint) error {
	currentSlot := types.Slot((nowTime - genesisTime) / params.BeaconConfig().SecondsPerSlot)
	currentEpoch := slots.ToEpoch(currentSlot)
	var prevEpoch types.Epoch
	// Prevents previous epoch under flow
	if currentEpoch > 1 {
		prevEpoch = currentEpoch - 1
	}
	if c.Epoch != prevEpoch && c.Epoch != currentEpoch {
		return fmt.Errorf("target epoch %d does not match current epoch %d or prev epoch %d", c.Epoch, currentEpoch, prevEpoch)
	}
	return nil
}

// verifyBeaconBlock verifies beacon head block is known and not from the future.
func (s *Service) verifyBeaconBlock(ctx context.Context, data *ethpb.AttestationData) error {
	r := bytesutil.ToBytes32(data.BeaconBlockRoot)
	b, err := s.getBlock(ctx, r)
	if err != nil {
		return err
	}
	if err := blocks.BeaconBlockIsNil(b); err != nil {
		return err
	}
	if b.Block().Slot() > data.Slot {
		return fmt.Errorf("could not process attestation for future block, block.Slot=%d > attestation.Data.Slot=%d", b.Block().Slot(), data.Slot)
	}
	return nil
}
