package blockchain

import (
	"bytes"
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

	baseState, err := s.stateGen.StateByRoot(ctx, bytesutil.ToBytes32(c.Root))
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pre state for slot %d", helpers.StartSlot(c.Epoch))
	}

	if helpers.StartSlot(c.Epoch) > baseState.Slot() {
		baseState = baseState.Copy()
		baseState, err = state.ProcessSlots(ctx, baseState, helpers.StartSlot(c.Epoch))
		if err != nil {
			return nil, errors.Wrapf(err, "could not process slots up to %d", helpers.StartSlot(c.Epoch))
		}
	}

	if err := s.checkpointState.AddCheckpointState(&cache.CheckpointState{
		Checkpoint: c,
		State:      baseState,
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
	r := bytesutil.ToBytes32(data.BeaconBlockRoot)
	b, err := s.beaconDB.Block(ctx, r)
	if err != nil {
		return err
	}
	// If the block does not exist in db, check again if block exists in initial sync block cache.
	// This could happen as the node first syncs to head.
	if b == nil && s.hasInitSyncBlock(r) {
		b = s.getInitSyncBlock(r)
	}
	if b == nil || b.Block == nil {
		return fmt.Errorf("beacon block %#x does not exist", bytesutil.Trunc(data.BeaconBlockRoot))
	}
	if b.Block.Slot > data.Slot {
		return fmt.Errorf("could not process attestation for future block, block.Slot=%d > attestation.Data.Slot=%d", b.Block.Slot, data.Slot)
	}
	return nil
}

// verifyLMDFFGConsistent verifies LMD GHOST and FFG votes are consistent with each other.
func (s *Service) verifyLMDFFGConsistent(ctx context.Context, ffgEpoch uint64, ffgRoot []byte, lmdRoot []byte) error {
	ffgSlot := helpers.StartSlot(ffgEpoch)
	r, err := s.ancestor(ctx, lmdRoot, ffgSlot)
	if err != nil {
		return err
	}
	if !bytes.Equal(ffgRoot, r) {
		return errors.New("FFG and LMD votes are not consistent")
	}

	return nil
}

// verifyAttestation validates input attestation is valid.
func (s *Service) verifyAttestation(ctx context.Context, baseState *stateTrie.BeaconState, a *ethpb.Attestation) (*ethpb.IndexedAttestation, error) {
	committee, err := helpers.BeaconCommitteeFromState(baseState, a.Data.Slot, a.Data.CommitteeIndex)
	if err != nil {
		return nil, err
	}
	indexedAtt := attestationutil.ConvertToIndexed(ctx, a, committee)
	if err := blocks.VerifyIndexedAttestation(ctx, baseState, indexedAtt); err != nil {
		return nil, errors.Wrap(err, "could not verify indexed attestation")
	}
	return indexedAtt, nil
}
