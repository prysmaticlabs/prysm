package forkchoice

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// ErrTargetRootNotInDB returns when the target block root of an attestation cannot be found in the
// beacon database.
var ErrTargetRootNotInDB = errors.New("target root does not exist in db")

// OnAttestation is called whenever an attestation is received, it updates validators latest vote,
// as well as the fork choice store struct.
//
// Spec pseudocode definition:
//   def on_attestation(store: Store, attestation: Attestation) -> None:
//    """
//    Run ``on_attestation`` upon receiving a new ``attestation`` from either within a block or directly on the wire.
//
//    An ``attestation`` that is asserted as invalid may be valid at a later time,
//    consider scheduling it for later processing in such case.
//    """
//    target = attestation.data.target
//
//    # Attestations must be from the current or previous epoch
//    current_epoch = compute_epoch_at_slot(get_current_slot(store))
//    # Use GENESIS_EPOCH for previous when genesis to avoid underflow
//    previous_epoch = current_epoch - 1 if current_epoch > GENESIS_EPOCH else GENESIS_EPOCH
//    assert target.epoch in [current_epoch, previous_epoch]
//    # Cannot calculate the current shuffling if have not seen the target
//    assert target.root in store.blocks
//
//    # Attestations target be for a known block. If target block is unknown, delay consideration until the block is found
//    assert target.root in store.blocks
//    # Attestations cannot be from future epochs. If they are, delay consideration until the epoch arrives
//    base_state = store.block_states[target.root].copy()
//    assert store.time >= base_state.genesis_time + compute_start_slot_at_epoch(target.epoch) * SECONDS_PER_SLOT
//
//    # Attestations must be for a known block. If block is unknown, delay consideration until the block is found
//    assert attestation.data.beacon_block_root in store.blocks
//    # Attestations must not be for blocks in the future. If not, the attestation should not be considered
//    assert store.blocks[attestation.data.beacon_block_root].slot <= attestation.data.slot
//
//    # Store target checkpoint state if not yet seen
//    if target not in store.checkpoint_states:
//        process_slots(base_state, compute_start_slot_at_epoch(target.epoch))
//        store.checkpoint_states[target] = base_state
//    target_state = store.checkpoint_states[target]
//
//    # Attestations can only affect the fork choice of subsequent slots.
//    # Delay consideration in the fork choice until their slot is in the past.
//    assert store.time >= (attestation.data.slot + 1) * SECONDS_PER_SLOT
//
//    # Get state at the `target` to validate attestation and calculate the committees
//    indexed_attestation = get_indexed_attestation(target_state, attestation)
//    assert is_valid_indexed_attestation(target_state, indexed_attestation)
//
//    # Update latest messages
//    for i in indexed_attestation.attesting_indices:
//        if i not in store.latest_messages or target.epoch > store.latest_messages[i].epoch:
//            store.latest_messages[i] = LatestMessage(epoch=target.epoch, root=attestation.data.beacon_block_root)
func (s *Store) OnAttestation(ctx context.Context, a *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "forkchoice.onAttestation")
	defer span.End()

	tgt := proto.Clone(a.Data.Target).(*ethpb.Checkpoint)
	tgtSlot := helpers.StartSlot(tgt.Epoch)

	// Verify beacon node has seen the target block before.
	if !s.db.HasBlock(ctx, bytesutil.ToBytes32(tgt.Root)) {
		return ErrTargetRootNotInDB
	}

	// Verify attestation target has had a valid pre state produced by the target block.
	baseState, err := s.verifyAttPreState(ctx, tgt)
	if err != nil {
		return err
	}

	// Verify attestation target is from current epoch or previous epoch.
	if err := s.verifyAttTargetEpoch(ctx, baseState.GenesisTime, uint64(time.Now().Unix()), tgt); err != nil {
		return err
	}

	// Verify Attestations cannot be from future epochs.
	if err := helpers.VerifySlotTime(baseState.GenesisTime, tgtSlot); err != nil {
		return errors.Wrap(err, "could not verify attestation target slot")
	}

	// Verify attestation beacon block is known and not from the future.
	if err := s.verifyBeaconBlock(ctx, a.Data); err != nil {
		return errors.Wrap(err, "could not verify attestation beacon block")
	}

	// Store target checkpoint state if not yet seen.
	baseState, err = s.saveCheckpointState(ctx, baseState, tgt)
	if err != nil {
		return err
	}

	// Verify attestations can only affect the fork choice of subsequent slots.
	if err := helpers.VerifySlotTime(baseState.GenesisTime, a.Data.Slot+1); err != nil {
		return err
	}

	// Use the target state to to validate attestation and calculate the committees.
	indexedAtt, err := s.verifyAttestation(ctx, baseState, a)
	if err != nil {
		return err
	}

	// Update every validator's latest vote.
	if err := s.updateAttVotes(ctx, indexedAtt, tgt.Root, tgt.Epoch); err != nil {
		return err
	}

	// Mark attestation as seen we don't update votes when it appears in block.
	if err := s.setSeenAtt(a); err != nil {
		return err
	}

	if err := s.db.SaveAttestation(ctx, a); err != nil {
		return err
	}

	log := log.WithFields(logrus.Fields{
		"Slot":               a.Data.Slot,
		"Index":              a.Data.CommitteeIndex,
		"AggregatedBitfield": fmt.Sprintf("%08b", a.AggregationBits),
		"BeaconBlockRoot":    fmt.Sprintf("%#x", bytesutil.Trunc(a.Data.BeaconBlockRoot)),
	})
	log.Debug("Updated latest votes")

	return nil
}

// verifyAttPreState validates input attested check point has a valid pre-state.
func (s *Store) verifyAttPreState(ctx context.Context, c *ethpb.Checkpoint) (*pb.BeaconState, error) {
	baseState, err := s.db.State(ctx, bytesutil.ToBytes32(c.Root))
	if err != nil {
		return nil, errors.Wrapf(err, "could not get pre state for slot %d", helpers.StartSlot(c.Epoch))
	}
	if baseState == nil {
		return nil, fmt.Errorf("pre state of target block %d does not exist", helpers.StartSlot(c.Epoch))
	}
	return baseState, nil
}

// verifyAttTargetEpoch validates attestation is from the current or previous epoch.
func (s *Store) verifyAttTargetEpoch(ctx context.Context, genesisTime uint64, nowTime uint64, c *ethpb.Checkpoint) error {
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
func (s *Store) verifyBeaconBlock(ctx context.Context, data *ethpb.AttestationData) error {
	b, err := s.db.Block(ctx, bytesutil.ToBytes32(data.BeaconBlockRoot))
	if err != nil {
		return err
	}
	if b == nil {
		return fmt.Errorf("beacon block %#x does not exist", bytesutil.Trunc(data.BeaconBlockRoot))
	}
	if b.Slot > data.Slot {
		return fmt.Errorf("could not process attestation for future block, %d > %d", b.Slot, data.Slot)
	}
	return nil
}

// saveCheckpointState saves and returns the processed state with the associated check point.
func (s *Store) saveCheckpointState(ctx context.Context, baseState *pb.BeaconState, c *ethpb.Checkpoint) (*pb.BeaconState, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.saveCheckpointState")
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
	if helpers.StartSlot(c.Epoch) > baseState.Slot {
		stateCopy := proto.Clone(baseState).(*pb.BeaconState)
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
func (s *Store) verifyAttestation(ctx context.Context, baseState *pb.BeaconState, a *ethpb.Attestation) (*ethpb.IndexedAttestation, error) {
	committee, err := helpers.BeaconCommittee(baseState, a.Data.Slot, a.Data.CommitteeIndex)
	if err != nil {
		return nil, err
	}
	indexedAtt, err := blocks.ConvertToIndexed(ctx, a, committee)
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
			indexedAtt, err = blocks.ConvertToIndexed(ctx, a, committee)
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

// updateAttVotes updates validator's latest votes based on the incoming attestation.
func (s *Store) updateAttVotes(
	ctx context.Context,
	indexedAtt *ethpb.IndexedAttestation,
	tgtRoot []byte,
	tgtEpoch uint64) error {

	indices := indexedAtt.AttestingIndices
	s.voteLock.Lock()
	defer s.voteLock.Unlock()
	for _, i := range indices {
		vote, ok := s.latestVoteMap[i]
		if !ok || tgtEpoch > vote.Epoch {
			s.latestVoteMap[i] = &pb.ValidatorLatestVote{
				Epoch: tgtEpoch,
				Root:  tgtRoot,
			}
		}
	}
	return nil
}

// setSeenAtt sets the attestation hash in seen attestation map to true.
func (s *Store) setSeenAtt(a *ethpb.Attestation) error {
	s.seenAttsLock.Lock()
	defer s.seenAttsLock.Unlock()

	r, err := hashutil.HashProto(a)
	if err != nil {
		return err
	}
	s.seenAtts[r] = true

	return nil
}

// aggregatedAttestation returns the aggregated attestation after checking saved one in db.
func (s *Store) aggregatedAttestations(ctx context.Context, att *ethpb.Attestation) ([]*ethpb.Attestation, error) {
	r, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		return nil, err
	}
	saved, err := s.db.AttestationsByDataRoot(ctx, r)
	if err != nil {
		return nil, err
	}

	if saved == nil {
		return []*ethpb.Attestation{att}, nil
	}

	aggregated, err := helpers.AggregateAttestations(append(saved, att))
	if err != nil {
		return nil, err
	}

	return aggregated, nil
}
