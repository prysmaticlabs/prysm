package blockchain

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
	"go.opencensus.io/trace"
)

// ErrTargetRootNotInDB returns when the target block root of an attestation cannot be found in the
// beacon database.
var ErrTargetRootNotInDB = errors.New("target root does not exist in db")

// onAttestation is called whenever an attestation is received, verifies the attestation is valid and saves
// it to the DB. As a stateless function, this does not hold nor delay attestation based on the spec descriptions.
// The delay is handled by the caller in `processAttestation`.
//
// Spec pseudocode definition:
//   def on_attestation(store: Store, attestation: Attestation) -> None:
//    """
//    Run ``on_attestation`` upon receiving a new ``attestation`` from either within a block or directly on the wire.
//
//    An ``attestation`` that is asserted as invalid may be valid at a later time,
//    consider scheduling it for later processing in such case.
//    """
//    validate_on_attestation(store, attestation)
//    store_target_checkpoint_state(store, attestation.data.target)
//
//    # Get state at the `target` to fully validate attestation
//    target_state = store.checkpoint_states[attestation.data.target]
//    indexed_attestation = get_indexed_attestation(target_state, attestation)
//    assert is_valid_indexed_attestation(target_state, indexed_attestation)
//
//    # Update latest messages for attesting indices
//    update_latest_messages(store, indexed_attestation.attesting_indices, attestation)
// TODO(#6072): This code path is highly untested. Requires comprehensive tests and simpler refactoring.
func (s *Service) onAttestation(ctx context.Context, a *ethpb.Attestation) ([]uint64, error) {
	ctx, span := trace.StartSpan(ctx, "blockChain.onAttestation")
	defer span.End()

	if a == nil {
		return nil, errors.New("nil attestation")
	}
	if a.Data == nil {
		return nil, errors.New("nil attestation.Data field")
	}
	if a.Data.Target == nil {
		return nil, errors.New("nil attestation.Data.Target field")
	}

	tgt := stateTrie.CopyCheckpoint(a.Data.Target)

	if helpers.SlotToEpoch(a.Data.Slot) != a.Data.Target.Epoch {
		return nil, fmt.Errorf("data slot is not in the same epoch as target %d != %d", helpers.SlotToEpoch(a.Data.Slot), a.Data.Target.Epoch)
	}

	// Verify beacon node has seen the target block before.
	if !s.hasBlock(ctx, bytesutil.ToBytes32(tgt.Root)) {
		return nil, ErrTargetRootNotInDB
	}

	// Retrieve attestation's data beacon block pre state. Advance pre state to latest epoch if necessary and
	// save it to the cache.
	baseState, err := s.getAttPreState(ctx, tgt)
	if err != nil {
		return nil, err
	}

	genesisTime := baseState.GenesisTime()

	// Verify attestation target is from current epoch or previous epoch.
	if err := s.verifyAttTargetEpoch(ctx, genesisTime, uint64(roughtime.Now().Unix()), tgt); err != nil {
		return nil, err
	}

	// Verify attestation beacon block is known and not from the future.
	if err := s.verifyBeaconBlock(ctx, a.Data); err != nil {
		return nil, errors.Wrap(err, "could not verify attestation beacon block")
	}

	// Verify LMG GHOST and FFG votes are consistent with each other.
	if err := s.verifyLMDFFGConsistent(ctx, a.Data.Target.Epoch, a.Data.Target.Root, a.Data.BeaconBlockRoot); err != nil {
		return nil, errors.Wrap(err, "could not verify attestation beacon block")
	}

	// Verify attestations can only affect the fork choice of subsequent slots.
	if err := helpers.VerifySlotTime(genesisTime, a.Data.Slot+1, params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
		return nil, err
	}

	// Use the target state to validate attestation and calculate the committees.
	indexedAtt, err := s.verifyAttestation(ctx, baseState, a)
	if err != nil {
		return nil, err
	}

	if indexedAtt.AttestingIndices == nil {
		return nil, errors.New("nil attesting indices")
	}

	// Update forkchoice store with the new attestation for updating weight.
	s.forkChoiceStore.ProcessAttestation(ctx, indexedAtt.AttestingIndices, bytesutil.ToBytes32(a.Data.BeaconBlockRoot), a.Data.Target.Epoch)

	return indexedAtt.AttestingIndices, nil
}
