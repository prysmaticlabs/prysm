package blockchain

import (
	"bytes"
	"context"
	"time"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// AttestationReceiver interface defines the methods of chain service receive and processing new attestations.
type AttestationReceiver interface {
	ReceiveAttestationNoPubsub(ctx context.Context, att *ethpb.Attestation) error
}

// ReceiveAttestationNoPubsub is a function that defines the operations that are preformed on
// attestation that is received from regular sync. The operations consist of:
//  1. Validate attestation, update validator's latest vote
//  2. Apply fork choice to the processed attestation
//  3. Save latest head info
func (s *Service) ReceiveAttestationNoPubsub(ctx context.Context, att *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveAttestationNoPubsub")
	defer span.End()

	// Update forkchoice store for the new attestation
	if err := s.forkChoiceStore.OnAttestation(ctx, att); err != nil {
		return errors.Wrap(err, "could not process attestation from fork choice service")
	}

	// Run fork choice for head block after updating fork choice store.
	headRoot, err := s.forkChoiceStore.Head(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head from fork choice service")
	}
	// Only save head if it's different than the current head.
	if !bytes.Equal(headRoot, s.HeadRoot()) {
		headBlk, err := s.beaconDB.Block(ctx, bytesutil.ToBytes32(headRoot))
		if err != nil {
			return errors.Wrap(err, "could not compute state from block head")
		}
		if err := s.saveHead(ctx, headBlk, bytesutil.ToBytes32(headRoot)); err != nil {
			return errors.Wrap(err, "could not save head")
		}
	}

	processedAttNoPubsub.Inc()
	return nil
}

// This processes attestations from the attestation pool to account for validator votes and fork choice.
func (s *Service) processAttestation() {
	period := time.Duration(params.BeaconConfig().SecondsPerSlot) * time.Second
	ticker := time.NewTicker(period)
	for {
		ctx := context.Background()
		select {
		case <-ticker.C:
			atts, err := s.opsPoolService.AttestationPoolForForkchoice(ctx)
			if err != nil {
				log.WithError(err).Error("Could not retrieve attestation from pool")
			}

			for _, a := range atts {
				if err := s.ReceiveAttestationNoPubsub(ctx, a); err != nil {
					log.WithError(err).Error("Could not receive attestation in chain service")
				}
			}
		case <-s.ctx.Done():
			log.Debug("Context closed, exiting routine")
			return
		}
	}
}
