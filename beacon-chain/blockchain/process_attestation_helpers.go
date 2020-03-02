package blockchain

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// getAttPreState retrieves the att pre state by either from the cache or the DB.
func (s *Service) getAttPreState(ctx context.Context, c *ethpb.Checkpoint) (*stateTrie.BeaconState, error) {
	s.checkpointStateLock.Lock()
	defer s.checkpointStateLock.Unlock()
	cachedState, err := s.checkpointState.StateByCheckpoint(c)
	if err != nil {
		return nil, errors.Wrap(err, "could not get cached checkpoint state")
	}
	if cachedState != nil {
		return cachedState, nil
	}

	baseState, err := s.beaconDB.State(ctx, bytesutil.ToBytes32(c.Root))
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pre state for slot %d", helpers.StartSlot(c.Epoch))
	}
	if baseState == nil {
		return nil, fmt.Errorf("pre state of target block %d does not exist", helpers.StartSlot(c.Epoch))
	}

	if helpers.StartSlot(c.Epoch) > baseState.Slot() {
		baseState, err = state.ProcessSlots(ctx, baseState, helpers.StartSlot(c.Epoch))
		if err != nil {
			return nil, errors.Wrapf(err, "could not process slots up to %d", helpers.StartSlot(c.Epoch))
		}
	}

	if err := s.checkpointState.AddCheckpointState(&cache.CheckpointState{
		Checkpoint: c,
		State:      baseState.Copy(),
	}); err != nil {
		return nil, errors.Wrap(err, "could not saved checkpoint state to cache")
	}

	return baseState, nil
}

// verifyAttTargetEpoch validates attestation is from the current or previous epoch.
func (s *Service) verifyAttTargetEpoch(ctx context.Context, genesisTime uint64, nowTime uint64, c *ethpb.Checkpoint) error {
	currentSlot := (nowTime - genesisTime) / params.BeaconConfig().SecondsPerSlot
	currentEpoch := helpers.SlotToEpoch(currentSlot)
	var prevEpoch uint64
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
	b, err := s.beaconDB.Block(ctx, bytesutil.ToBytes32(data.BeaconBlockRoot))
	if err != nil {
		return err
	}
	if b == nil || b.Block == nil {
		return fmt.Errorf("beacon block %#x does not exist", bytesutil.Trunc(data.BeaconBlockRoot))
	}
	if b.Block.Slot > data.Slot {
		return fmt.Errorf("could not process attestation for future block, %d > %d", b.Block.Slot, data.Slot)
	}
	return nil
}

// verifyAttestation validates input attestation is valid.
func (s *Service) verifyAttestation(ctx context.Context, baseState *stateTrie.BeaconState, a *ethpb.Attestation) (*ethpb.IndexedAttestation, error) {
	committee, err := helpers.BeaconCommitteeFromState(baseState, a.Data.Slot, a.Data.CommitteeIndex)
	if err != nil {
		return nil, err
	}
	indexedAtt, err := attestationutil.ConvertToIndexed(ctx, a, committee)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert attestation to indexed attestation")
	}

	if err := blocks.VerifyIndexedAttestation(ctx, baseState, indexedAtt); err != nil {
		if err == blocks.ErrSigFailedToVerify {
			// When sig fails to verify, check if there's a differences in committees due to
			// different seeds.
			aState, err := s.beaconDB.State(ctx, bytesutil.ToBytes32(a.Data.BeaconBlockRoot))
			if err != nil {
				return nil, err
			}
			epoch := helpers.SlotToEpoch(a.Data.Slot)
			origSeed, err := helpers.Seed(baseState, epoch, params.BeaconConfig().DomainBeaconAttester)
			if err != nil {
				return nil, errors.Wrap(err, "could not get original seed")
			}

			aSeed, err := helpers.Seed(aState, epoch, params.BeaconConfig().DomainBeaconAttester)
			if err != nil {
				return nil, errors.Wrap(err, "could not get attester's seed")
			}
			if origSeed != aSeed {
				return nil, fmt.Errorf("could not verify indexed attestation due to differences in seeds: %v != %v",
					hex.EncodeToString(bytesutil.Trunc(origSeed[:])), hex.EncodeToString(bytesutil.Trunc(aSeed[:])))
			}
		}
		return nil, errors.Wrap(err, "could not verify indexed attestation")
	}

	return indexedAtt, nil
}
