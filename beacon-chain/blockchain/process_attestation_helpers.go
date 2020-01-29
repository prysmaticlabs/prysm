package blockchain

import (
	"context"
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
	"go.opencensus.io/trace"
)

// verifyAttPreState validates input attested check point has a valid pre-state.
func (s *Service) verifyAttPreState(ctx context.Context, c *ethpb.Checkpoint) (*stateTrie.BeaconState, error) {
	baseState, err := s.beaconDB.State(ctx, bytesutil.ToBytes32(c.Root))
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pre state for slot %d", helpers.StartSlot(c.Epoch))
	}
	if baseState == nil {
		return nil, fmt.Errorf("pre state of target block %d does not exist", helpers.StartSlot(c.Epoch))
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

// saveCheckpointState saves and returns the processed state with the associated check point.
func (s *Service) saveCheckpointState(ctx context.Context, baseState *stateTrie.BeaconState, c *ethpb.Checkpoint) (*stateTrie.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "blockchain.saveCheckpointState")
	defer span.End()

	s.checkpointStateLock.Lock()
	defer s.checkpointStateLock.Unlock()
	cachedState, err := s.checkpointState.StateByCheckpoint(c)
	if err != nil {
		return nil, errors.Wrap(err, "could not get cached checkpoint state")
	}
	if cachedState != nil {
		return cachedState, nil
	}

	// Advance slots only when it's higher than current state slot.
	if helpers.StartSlot(c.Epoch) > baseState.Slot() {
		stateCopy := baseState.Copy()
		stateCopy, err = state.ProcessSlots(ctx, stateCopy, helpers.StartSlot(c.Epoch))
		if err != nil {
			return nil, errors.Wrapf(err, "could not process slots up to %d", helpers.StartSlot(c.Epoch))
		}

		if err := s.checkpointState.AddCheckpointState(&cache.CheckpointState{
			Checkpoint: c,
			State:      stateCopy,
		}); err != nil {
			return nil, errors.Wrap(err, "could not saved checkpoint state to cache")
		}

		return stateCopy, nil
	}

	return baseState, nil
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

		// TODO(3603): Delete the following signature verify fallback when issue 3603 closes.
		// When signature fails to verify with committee cache enabled at run time,
		// the following re-runs the same signature verify routine without cache in play.
		// This provides extra assurance that committee cache can't break run time.
		if err == blocks.ErrSigFailedToVerify {
			committee, err = helpers.BeaconCommitteeWithoutCache(baseState, a.Data.Slot, a.Data.CommitteeIndex)
			if err != nil {
				return nil, errors.Wrap(err, "could not convert attestation to indexed attestation without cache")
			}
			indexedAtt, err = attestationutil.ConvertToIndexed(ctx, a, committee)
			if err != nil {
				return nil, errors.Wrap(err, "could not convert attestation to indexed attestation")
			}
			if err := blocks.VerifyIndexedAttestation(ctx, baseState, indexedAtt); err != nil {
				return nil, errors.Wrap(err, "could not verify indexed attestation without cache")
			}
			sigFailsToVerify.Inc()
			return indexedAtt, nil
		}

		return nil, errors.Wrap(err, "could not verify indexed attestation")
	}

	return indexedAtt, nil
}
