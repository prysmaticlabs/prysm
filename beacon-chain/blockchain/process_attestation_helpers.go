package blockchain

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/mputil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// getAttPreState retrieves the att pre state by either from the cache or the DB.
func (s *Service) getAttPreState(ctx context.Context, c *ethpb.Checkpoint) (iface.BeaconState, error) {
	// Use a multilock to allow scoped holding of a mutex by a checkpoint root + epoch
	// allowing us to behave smarter in terms of how this function is used concurrently.
	epochKey := strconv.FormatUint(uint64(c.Epoch), 10 /* base 10 */)
	lock := mputil.NewMultilock(string(c.Root) + epochKey)
	lock.Lock()
	defer lock.Unlock()
	cachedState, err := s.checkpointStateCache.StateByCheckpoint(c)
	if err != nil {
		return nil, errors.Wrap(err, "could not get cached checkpoint state")
	}
	if cachedState != nil {
		return cachedState, nil
	}

	baseState, err := s.cfg.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(c.Root))
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pre state for epoch %d", c.Epoch)
	}

	epochStartSlot, err := helpers.StartSlot(c.Epoch)
	if err != nil {
		return nil, err
	}
	if epochStartSlot > baseState.Slot() {
		if featureconfig.Get().EnableNextSlotStateCache {
			baseState, err = state.ProcessSlotsUsingNextSlotCache(ctx, baseState, c.Root, epochStartSlot)
			if err != nil {
				return nil, errors.Wrapf(err, "could not process slots up to epoch %d", c.Epoch)
			}
		} else {
			baseState, err = state.ProcessSlots(ctx, baseState, epochStartSlot)
			if err != nil {
				return nil, errors.Wrapf(err, "could not process slots up to epoch %d", c.Epoch)
			}
		}
		if err := s.checkpointStateCache.AddCheckpointState(c, baseState); err != nil {
			return nil, errors.Wrap(err, "could not saved checkpoint state to cache")
		}
		return baseState, nil
	}

	// To avoid sharing the same state across checkpoint state cache and hot state cache,
	// we don't add the state to check point cache.
	has, err := s.cfg.StateGen.HasStateInCache(ctx, bytesutil.ToBytes32(c.Root))
	if err != nil {
		return nil, err
	}
	if !has {
		if err := s.checkpointStateCache.AddCheckpointState(c, baseState); err != nil {
			return nil, errors.Wrap(err, "could not saved checkpoint state to cache")
		}
	}
	return baseState, nil

}

// verifyAttTargetEpoch validates attestation is from the current or previous epoch.
func (s *Service) verifyAttTargetEpoch(_ context.Context, genesisTime, nowTime uint64, c *ethpb.Checkpoint) error {
	currentSlot := types.Slot((nowTime - genesisTime) / params.BeaconConfig().SecondsPerSlot)
	currentEpoch := helpers.SlotToEpoch(currentSlot)
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
	b, err := s.cfg.BeaconDB.Block(ctx, r)
	if err != nil {
		return err
	}
	// If the block does not exist in db, check again if block exists in initial sync block cache.
	// This could happen as the node first syncs to head.
	if b == nil && s.hasInitSyncBlock(r) {
		b = s.getInitSyncBlock(r)
	}
	if err := helpers.VerifyNilBeaconBlock(b); err != nil {
		return err
	}
	if b.Block.Slot > data.Slot {
		return fmt.Errorf("could not process attestation for future block, block.Slot=%d > attestation.Data.Slot=%d", b.Block.Slot, data.Slot)
	}
	return nil
}
