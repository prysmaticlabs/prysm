package blockchain

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/feed"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slotutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// AttestationReceiver interface defines the methods of chain service receive and processing new attestations.
type AttestationReceiver interface {
	ReceiveAttestationNoPubsub(ctx context.Context, att *ethpb.Attestation) error
	IsValidAttestation(ctx context.Context, att *ethpb.Attestation) bool
}

// ReceiveAttestationNoPubsub is a function that defines the operations that are preformed on
// attestation that is received from regular sync. The operations consist of:
//  1. Validate attestation, update validator's latest vote
//  2. Apply fork choice to the processed attestation
//  3. Save latest head info
func (s *Service) ReceiveAttestationNoPubsub(ctx context.Context, att *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveAttestationNoPubsub")
	defer span.End()

	_, err := s.onAttestation(ctx, att)
	if err != nil {
		return errors.Wrap(err, "could not process attestation")
	}

	return nil
}

// IsValidAttestation returns true if the attestation can be verified against its pre-state.
func (s *Service) IsValidAttestation(ctx context.Context, att *ethpb.Attestation) bool {
	baseState, err := s.getAttPreState(ctx, att.Data.Target)
	if err != nil {
		log.WithError(err).Error("Failed to validate attestation")
		return false
	}

	if err := blocks.VerifyAttestation(ctx, baseState, att); err != nil {
		log.WithError(err).Error("Failed to validate attestation")
		return false
	}

	return true
}

// This processes attestations from the attestation pool to account for validator votes and fork choice.
func (s *Service) processAttestation() {
	// Wait for state to be initialized.
	stateChannel := make(chan *feed.Event, 1)
	stateSub := s.stateNotifier.StateFeed().Subscribe(stateChannel)
	<-stateChannel
	stateSub.Unsubscribe()

	st := slotutil.GetSlotTicker(s.genesisTime, params.BeaconConfig().SecondsPerSlot)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-st.C():
			ctx := context.Background()
			atts := s.attPool.ForkchoiceAttestations()
			for _, a := range atts {
				hasState := s.beaconDB.HasState(ctx, bytesutil.ToBytes32(a.Data.BeaconBlockRoot)) && s.beaconDB.HasState(ctx, bytesutil.ToBytes32(a.Data.Target.Root))
				hasBlock := s.hasBlock(ctx, bytesutil.ToBytes32(a.Data.BeaconBlockRoot))
				if !(hasState && hasBlock) {
					continue
				}

				if err := s.attPool.DeleteForkchoiceAttestation(a); err != nil {
					log.WithError(err).Error("Could not delete fork choice attestation in pool")
				}

				if !s.verifyCheckpointEpoch(a.Data.Target) {
					continue
				}

				if err := s.ReceiveAttestationNoPubsub(ctx, a); err != nil {
					log.WithFields(logrus.Fields{
						"slot":             a.Data.Slot,
						"committeeIndex":   a.Data.CommitteeIndex,
						"beaconBlockRoot":  fmt.Sprintf("%#x", bytesutil.Trunc(a.Data.BeaconBlockRoot)),
						"targetRoot":       fmt.Sprintf("%#x", bytesutil.Trunc(a.Data.Target.Root)),
						"aggregationCount": a.AggregationBits.Count(),
					}).WithError(err).Warn("Could not receive attestation in chain service")
				}
			}
		}
	}
}

// This verifies the epoch of input checkpoint is within current epoch and previous epoch
// with respect to current time. Returns true if it's within, false if it's not.
func (s *Service) verifyCheckpointEpoch(c *ethpb.Checkpoint) bool {
	now := uint64(time.Now().Unix())
	genesisTime := uint64(s.genesisTime.Unix())
	currentSlot := (now - genesisTime) / params.BeaconConfig().SecondsPerSlot
	currentEpoch := helpers.SlotToEpoch(currentSlot)

	var prevEpoch uint64
	if currentEpoch > 1 {
		prevEpoch = currentEpoch - 1
	}

	if c.Epoch != prevEpoch && c.Epoch != currentEpoch {
		return false
	}

	return true
}
