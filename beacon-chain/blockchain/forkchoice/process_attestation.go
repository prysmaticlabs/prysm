package forkchoice

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// OnAttestation is called whenever an attestation is received, it updates validators latest vote,
// as well as the fork choice store struct.
//
// Spec pseudocode definition:
//   def on_attestation(store: Store, attestation: Attestation) -> None:
//    target = attestation.data.target
//
//    # Cannot calculate the current shuffling if have not seen the target
//    assert target.root in store.blocks
//
//    # Attestations cannot be from future epochs. If they are, delay consideration until the epoch arrives
//    base_state = store.block_states[target.root].copy()
//    assert store.time >= base_state.genesis_time + compute_start_slot_of_epoch(target.epoch) * SECONDS_PER_SLOT
//
//    # Store target checkpoint state if not yet seen
//    if target not in store.checkpoint_states:
//        process_slots(base_state, compute_start_slot_of_epoch(target.epoch))
//        store.checkpoint_states[target] = base_state
//    target_state = store.checkpoint_states[target]
//
//    # Attestations can only affect the fork choice of subsequent slots.
//    # Delay consideration in the fork choice until their slot is in the past.
//    attestation_slot = get_attestation_data_slot(target_state, attestation.data)
//    assert store.time >= (attestation_slot + 1) * SECONDS_PER_SLOT
//
//    # Get state at the `target` to validate attestation and calculate the committees
//    indexed_attestation = get_indexed_attestation(target_state, attestation)
//    assert is_valid_indexed_attestation(target_state, indexed_attestation)
//
//    # Update latest messages
//    for i in indexed_attestation.custody_bit_0_indices + indexed_attestation.custody_bit_1_indices:
//        if i not in store.latest_messages or target.epoch > store.latest_messages[i].epoch:
//            store.latest_messages[i] = LatestMessage(epoch=target.epoch, root=attestation.data.beacon_block_root)
func (s *Store) OnAttestation(ctx context.Context, a *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "forkchoice.onAttestation")
	defer span.End()

	tgt := proto.Clone(a.Data.Target).(*ethpb.Checkpoint)
	tgtSlot := helpers.StartSlot(tgt.Epoch)

	// Verify beacon node has seen the target block before.
	if !s.db.HasBlock(ctx, bytesutil.ToBytes32(tgt.Root)) {
		return fmt.Errorf("target root %#x does not exist in db", bytesutil.Trunc(tgt.Root))
	}

	// Verify attestation target has had a valid pre state produced by the target block.
	baseState, err := s.verifyAttPreState(ctx, tgt)
	if err != nil {
		return err
	}

	// Verify Attestations cannot be from future epochs.
	if err := helpers.VerifySlotTime(baseState.GenesisTime, tgtSlot); err != nil {
		return errors.Wrap(err, "could not verify attestation target slot")
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
		"Index":              a.Data.Index,
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

// saveCheckpointState saves and returns the processed state with the associated check point.
func (s *Store) saveCheckpointState(ctx context.Context, baseState *pb.BeaconState, c *ethpb.Checkpoint) (*pb.BeaconState, error) {
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
		baseState, err = state.ProcessSlots(ctx, stateCopy, helpers.StartSlot(c.Epoch))
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

// verifyAttestation validates input attestation is valid.
func (s *Store) verifyAttestation(ctx context.Context, baseState *pb.BeaconState, a *ethpb.Attestation) (*ethpb.IndexedAttestation, error) {
	indexedAtt, err := blocks.ConvertToIndexed(ctx, baseState, a)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert attestation to indexed attestation")
	}
	if err := blocks.VerifyIndexedAttestation(ctx, baseState, indexedAtt); err != nil {
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

	indices := append(indexedAtt.CustodyBit_0Indices, indexedAtt.CustodyBit_1Indices...)
	newVoteIndices := make([]uint64, 0, len(indices))
	newVotes := make([]*pb.ValidatorLatestVote, 0, len(indices))
	for _, i := range indices {
		vote, err := s.db.ValidatorLatestVote(ctx, i)
		if err != nil {
			return errors.Wrapf(err, "could not get latest vote for validator %d", i)
		}
		if vote == nil || tgtEpoch > vote.Epoch {
			newVotes = append(newVotes, &pb.ValidatorLatestVote{
				Epoch: tgtEpoch,
				Root:  tgtRoot,
			})
			newVoteIndices = append(newVoteIndices, i)
		}
	}
	return s.db.SaveValidatorLatestVotes(ctx, newVoteIndices, newVotes)
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
