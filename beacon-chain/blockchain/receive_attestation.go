package blockchain

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/async/event"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/time/slots"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// AttestationStateFetcher allows for retrieving a beacon state corresponding to the block
// root of an attestation's target checkpoint.
type AttestationStateFetcher interface {
	AttestationTargetState(ctx context.Context, target *ethpb.Checkpoint) (state.BeaconState, error)
}

// AttestationReceiver interface defines the methods of chain service receive and processing new attestations.
type AttestationReceiver interface {
	AttestationStateFetcher
	VerifyLmdFfgConsistency(ctx context.Context, att *ethpb.Attestation) error
	VerifyFinalizedConsistency(ctx context.Context, root []byte) error
}

// AttestationTargetState returns the pre state of attestation.
func (s *Service) AttestationTargetState(ctx context.Context, target *ethpb.Checkpoint) (state.BeaconState, error) {
	ss, err := slots.EpochStart(target.Epoch)
	if err != nil {
		return nil, err
	}
	if err := slots.ValidateClock(ss, uint64(s.genesisTime.Unix())); err != nil {
		return nil, err
	}
	return s.getAttPreState(ctx, target)
}

// VerifyLmdFfgConsistency verifies that attestation's LMD and FFG votes are consistency to each other.
func (s *Service) VerifyLmdFfgConsistency(ctx context.Context, a *ethpb.Attestation) error {
	targetSlot, err := slots.EpochStart(a.Data.Target.Epoch)
	if err != nil {
		return err
	}
	r, err := s.ancestor(ctx, a.Data.BeaconBlockRoot, targetSlot)
	if err != nil {
		return err
	}
	if !bytes.Equal(a.Data.Target.Root, r) {
		return errors.New("FFG and LMD votes are not consistent")
	}
	return nil
}

// VerifyFinalizedConsistency verifies input root is consistent with finalized store.
// When the input root is not be consistent with finalized store then we know it is not
// on the finalized check point that leads to current canonical chain and should be rejected accordingly.
func (s *Service) VerifyFinalizedConsistency(ctx context.Context, root []byte) error {
	// A canonical root implies the root to has an ancestor that aligns with finalized check point.
	// In this case, we could exit early to save on additional computation.
	blockRoot := bytesutil.ToBytes32(root)
	if s.cfg.ForkChoiceStore.HasNode(blockRoot) && s.cfg.ForkChoiceStore.IsCanonical(blockRoot) {
		return nil
	}

	f := s.FinalizedCheckpt()
	ss, err := slots.EpochStart(f.Epoch)
	if err != nil {
		return err
	}
	r, err := s.ancestor(ctx, root, ss)
	if err != nil {
		return err
	}
	if !bytes.Equal(f.Root, r) {
		return errors.New("Root and finalized store are not consistent")
	}

	return nil
}

// This routine processes fork choice attestations from the pool to account for validator votes and fork choice.
func (s *Service) spawnProcessAttestationsRoutine(stateFeed *event.Feed) {
	// Wait for state to be initialized.
	stateChannel := make(chan *feed.Event, 1)
	stateSub := stateFeed.Subscribe(stateChannel)
	go func() {
		select {
		case <-s.ctx.Done():
			stateSub.Unsubscribe()
			return
		case <-stateChannel:
			stateSub.Unsubscribe()
			break
		}

		if s.genesisTime.IsZero() {
			log.Warn("ProcessAttestations routine waiting for genesis time")
			for s.genesisTime.IsZero() {
				if err := s.ctx.Err(); err != nil {
					log.WithError(err).Error("Giving up waiting for genesis time")
					return
				}
				time.Sleep(1 * time.Second)
			}
			log.Warn("Genesis time received, now available to process attestations")
		}

		st := slots.NewSlotTicker(s.genesisTime, params.BeaconConfig().SecondsPerSlot)
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-st.C():
				if err := s.ForkChoicer().NewSlot(s.ctx, s.CurrentSlot()); err != nil {
					log.WithError(err).Error("Could not process new slot")
				}

				if err := s.UpdateHead(s.ctx); err != nil {
					log.WithError(err).Error("Could not process attestations and update head")
				}
			}
		}
	}()
}

// UpdateHead updates the canonical head of the chain based on information from fork-choice attestations and votes.
// It requires no external inputs.
func (s *Service) UpdateHead(ctx context.Context) error {
	// Only one process can process attestations and update head at a time.
	s.processAttestationsLock.Lock()
	defer s.processAttestationsLock.Unlock()

	start := time.Now()
	s.processAttestations(ctx)
	processAttsElapsedTime.Observe(float64(time.Since(start).Milliseconds()))

	justified := s.ForkChoicer().JustifiedCheckpoint()
	balances, err := s.justifiedBalances.get(ctx, justified.Root)
	if err != nil {
		return err
	}
	start = time.Now()
	newHeadRoot, err := s.cfg.ForkChoiceStore.Head(ctx, balances)
	if err != nil {
		log.WithError(err).Error("Could not compute head from new attestations")
	}
	newAttHeadElapsedTime.Observe(float64(time.Since(start).Milliseconds()))

	s.headLock.RLock()
	if s.headRoot() != newHeadRoot {
		log.WithFields(logrus.Fields{
			"oldHeadRoot": fmt.Sprintf("%#x", s.headRoot()),
			"newHeadRoot": fmt.Sprintf("%#x", newHeadRoot),
		}).Debug("Head changed due to attestations")
	}
	s.headLock.RUnlock()
	if err := s.notifyEngineIfChangedHead(ctx, newHeadRoot); err != nil {
		return err
	}
	return nil
}

// This calls notify Forkchoice Update in the event that the head has changed
func (s *Service) notifyEngineIfChangedHead(ctx context.Context, newHeadRoot [32]byte) error {
	s.headLock.RLock()
	if newHeadRoot == [32]byte{} || s.headRoot() == newHeadRoot {
		s.headLock.RUnlock()
		return nil
	}
	s.headLock.RUnlock()

	if !s.hasBlockInInitSyncOrDB(ctx, newHeadRoot) {
		log.Debug("New head does not exist in DB. Do nothing")
		return nil // We don't have the block, don't notify the engine and update head.
	}

	newHeadBlock, err := s.getBlock(ctx, newHeadRoot)
	if err != nil {
		log.WithError(err).Error("Could not get new head block")
		return nil
	}
	headState, err := s.cfg.StateGen.StateByRoot(ctx, newHeadRoot)
	if err != nil {
		log.WithError(err).Error("Could not get state from db")
		return nil
	}
	arg := &notifyForkchoiceUpdateArg{
		headState: headState,
		headRoot:  newHeadRoot,
		headBlock: newHeadBlock.Block(),
	}
	_, err = s.notifyForkchoiceUpdate(s.ctx, arg)
	if err != nil {
		return err
	}
	if err := s.saveHead(ctx, newHeadRoot, newHeadBlock, headState); err != nil {
		log.WithError(err).Error("could not save head")
	}
	return nil
}

// This processes fork choice attestations from the pool to account for validator votes and fork choice.
func (s *Service) processAttestations(ctx context.Context) {
	atts := s.cfg.AttPool.ForkchoiceAttestations()
	for _, a := range atts {
		// Based on the spec, don't process the attestation until the subsequent slot.
		// This delays consideration in the fork choice until their slot is in the past.
		// https://github.com/ethereum/consensus-specs/blob/dev/specs/phase0/fork-choice.md#validate_on_attestation
		nextSlot := a.Data.Slot + 1
		if err := slots.VerifyTime(uint64(s.genesisTime.Unix()), nextSlot, params.BeaconNetworkConfig().MaximumGossipClockDisparity); err != nil {
			continue
		}

		hasState := s.cfg.BeaconDB.HasStateSummary(ctx, bytesutil.ToBytes32(a.Data.BeaconBlockRoot))
		hasBlock := s.hasBlock(ctx, bytesutil.ToBytes32(a.Data.BeaconBlockRoot))
		if !(hasState && hasBlock) {
			continue
		}

		if err := s.cfg.AttPool.DeleteForkchoiceAttestation(a); err != nil {
			log.WithError(err).Error("Could not delete fork choice attestation in pool")
		}

		if !helpers.VerifyCheckpointEpoch(a.Data.Target, s.genesisTime) {
			continue
		}

		if err := s.receiveAttestationNoPubsub(ctx, a); err != nil {
			log.WithFields(logrus.Fields{
				"slot":             a.Data.Slot,
				"committeeIndex":   a.Data.CommitteeIndex,
				"beaconBlockRoot":  fmt.Sprintf("%#x", bytesutil.Trunc(a.Data.BeaconBlockRoot)),
				"targetRoot":       fmt.Sprintf("%#x", bytesutil.Trunc(a.Data.Target.Root)),
				"aggregationCount": a.AggregationBits.Count(),
			}).WithError(err).Warn("Could not process attestation for fork choice")
		}
	}
}

// receiveAttestationNoPubsub is a function that defines the operations that are performed on
// attestation that is received from regular sync. The operations consist of:
//  1. Validate attestation, update validator's latest vote
//  2. Apply fork choice to the processed attestation
//  3. Save latest head info
func (s *Service) receiveAttestationNoPubsub(ctx context.Context, att *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.receiveAttestationNoPubsub")
	defer span.End()

	if err := s.OnAttestation(ctx, att); err != nil {
		return errors.Wrap(err, "could not process attestation")
	}

	return nil
}
