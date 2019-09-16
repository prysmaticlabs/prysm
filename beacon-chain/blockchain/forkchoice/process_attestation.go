package forkchoice

import (
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/cache"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/roughtime"
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
func (s *Store) OnAttestation(ctx context.Context, a *ethpb.Attestation) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "forkchoice.onAttestation")
	defer span.End()

	tgt := a.Data.Target
	tgtSlot := helpers.StartSlot(tgt.Epoch)

	// Verify beacon node has seen the target block before.
	if !s.db.HasBlock(ctx, bytesutil.ToBytes32(tgt.Root)) {
		return 0, fmt.Errorf("target root %#x does not exist in db", bytesutil.Trunc(tgt.Root))
	}

	// Verify attestation target has had a valid pre state produced by the target block.
	baseState, err := s.verifyAttPreState(ctx, tgt)
	if err != nil {
		return 0, err
	}

	// Verify Attestations cannot be from future epochs.
	slotTime := baseState.GenesisTime + tgtSlot*params.BeaconConfig().SecondsPerSlot
	currentTime := uint64(roughtime.Now().Unix())
	if slotTime > currentTime {
		return 0, fmt.Errorf("could not process attestation from the future epoch, time %d > time %d", slotTime, currentTime)
	}

	// Store target checkpoint state if not yet seen.
	baseState, err = s.saveCheckpointState(ctx, baseState, tgt)
	if err != nil {
		return 0, err
	}

	// Delay attestation processing until the subsequent slot.
	if err := s.waitForAttInclDelay(ctx, a, baseState); err != nil {
		return 0, err
	}

	// Verify attestations can only affect the fork choice of subsequent slots.
	if err := s.verifyAttSlotTime(ctx, baseState, a.Data); err != nil {
		return 0, err
	}

	// Process aggregated attestation in the queue every `slot/2` duration,
	// this allows the incoming attestation to aggregate and avoid
	// excessively processing individual attestations.
	if halfSlot(baseState.GenesisTime) {
		s.attsQueueLock.Lock()
		for root, a := range s.attsQueue {
			log.WithFields(logrus.Fields{
				"AggregatedBitfield": fmt.Sprintf("%b", a.AggregationBits),
				"Root":               fmt.Sprintf("%#x", root),
			}).Debug("Updating latest votes")

			// Use the target state to to validate attestation and calculate the committees.
			indexedAtt, err := s.verifyAttestation(ctx, baseState, a)
			if err != nil {
				return 0, err
			}

			// Update every validator's latest vote.
			if err := s.updateAttVotes(ctx, indexedAtt, tgt.Root, tgt.Epoch); err != nil {
				return 0, err
			}
			delete(s.attsQueue, root)
		}
		s.attsQueueLock.Unlock()
	}

	return tgtSlot, nil
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

// waitForAttInclDelay waits until the next slot because attestation can only affect
// fork choice of subsequent slot. This is to delay attestation inclusion for fork choice
// until the attested slot is in the past.
func (s *Store) waitForAttInclDelay(ctx context.Context, a *ethpb.Attestation, targetState *pb.BeaconState) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.forkchoice.waitForAttInclDelay")
	defer span.End()

	slot, err := helpers.AttestationDataSlot(targetState, a.Data)
	if err != nil {
		return errors.Wrap(err, "could not get attestation slot")
	}

	nextSlot := slot + 1
	duration := time.Duration(nextSlot*params.BeaconConfig().SecondsPerSlot)*time.Second + time.Millisecond
	timeToInclude := time.Unix(int64(targetState.GenesisTime), 0).Add(duration)

	if err := s.aggregateAttestation(ctx, a); err != nil {
		return errors.Wrap(err, "could not aggregate attestation")
	}

	time.Sleep(time.Until(timeToInclude))
	return nil
}

// aggregateAttestation aggregates the attestations in the pending queue.
func (s *Store) aggregateAttestation(ctx context.Context, att *ethpb.Attestation) error {
	s.attsQueueLock.Lock()
	defer s.attsQueueLock.Unlock()

	root, err := ssz.HashTreeRoot(att.Data)
	if err != nil {
		return err
	}

	incomingAttBits := att.AggregationBits
	if a, ok := s.attsQueue[root]; ok {
		if !a.AggregationBits.Contains(incomingAttBits) {
			newBits := a.AggregationBits.Or(incomingAttBits)
			incomingSig, err := bls.SignatureFromBytes(att.Signature)
			if err != nil {
				return err
			}
			currentSig, err := bls.SignatureFromBytes(a.Signature)
			if err != nil {
				return err
			}
			aggregatedSig := bls.AggregateSignatures([]*bls.Signature{currentSig, incomingSig})
			a.Signature = aggregatedSig.Marshal()
			a.AggregationBits = newBits
			s.attsQueue[root] = a
		}
		return nil
	}

	s.attsQueue[root] = att
	return nil
}

// verifyAttSlotTime validates input attestation is not from the future.
func (s *Store) verifyAttSlotTime(ctx context.Context, baseState *pb.BeaconState, d *ethpb.AttestationData) error {
	aSlot, err := helpers.AttestationDataSlot(baseState, d)
	if err != nil {
		return errors.Wrap(err, "could not get attestation slot")
	}
	slotTime := baseState.GenesisTime + (aSlot+1)*params.BeaconConfig().SecondsPerSlot
	currentTime := uint64(time.Now().Unix())
	if slotTime > currentTime+timeShiftTolerance {
		return fmt.Errorf("could not process attestation for fork choice until inclusion delay, time %d > time %d", slotTime, currentTime)
	}
	return nil
}

// verifyAttestation validates input attestation is valid.
func (s *Store) verifyAttestation(ctx context.Context, baseState *pb.BeaconState, a *ethpb.Attestation) (*ethpb.IndexedAttestation, error) {
	indexedAtt, err := blocks.ConvertToIndexed(baseState, a)
	if err != nil {
		return nil, errors.Wrap(err, "could not convert attestation to indexed attestation")
	}
	if err := blocks.VerifyIndexedAttestation(baseState, indexedAtt); err != nil {
		return nil, errors.New("could not verify indexed attestation")
	}
	return indexedAtt, nil
}

// updateAttVotes updates validator's latest votes based on the incoming attestation.
func (s *Store) updateAttVotes(
	ctx context.Context,
	indexedAtt *ethpb.IndexedAttestation,
	tgtRoot []byte,
	tgtEpoch uint64) error {

	for _, i := range append(indexedAtt.CustodyBit_0Indices, indexedAtt.CustodyBit_1Indices...) {
		vote, err := s.db.ValidatorLatestVote(ctx, i)
		if err != nil {
			return errors.Wrapf(err, "could not get latest vote for validator %d", i)
		}
		if !s.db.HasValidatorLatestVote(ctx, i) || tgtEpoch > vote.Epoch {
			if err := s.db.SaveValidatorLatestVote(ctx, i, &pb.ValidatorLatestVote{
				Epoch: tgtEpoch,
				Root:  tgtRoot,
			}); err != nil {
				return errors.Wrapf(err, "could not save latest vote for validator %d", i)
			}
		}
	}
	return nil
}

// returns true when time is divisible with slot duration / 2.
func halfSlot(genesisTime uint64) bool {
	t := time.Unix(int64(genesisTime), 0)
	return uint64(roughtime.Since(t).Seconds())%params.BeaconConfig().SecondsPerSlot/2 == 0
}
