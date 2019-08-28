package blockchain

import (
	"bytes"
	"context"
	"encoding/hex"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// AttestationReceiver interface defines the methods of chain service receive and processing new attestations.
type AttestationReceiver interface {
	ReceiveAttestation(ctx context.Context, att *ethpb.Attestation) error
	ReceiveAttestationNoPubsub(ctx context.Context, att *ethpb.Attestation) error
}

// ReceiveAttestation is a function that defines the operations that are preformed on
// attestation that is received from regular sync. The operations consist of:
//  1. Gossip attestation to other peers
//  2. Validate attestation, update validator's latest vote
//  3. Apply fork choice to the processed attestation
//  4. Save latest head info
func (c *ChainService) ReceiveAttestation(ctx context.Context, att *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveAttestation")
	defer span.End()

	// Broadcast the new attestation to the network.
	if err := c.p2p.Broadcast(ctx, att); err != nil {
		return errors.Wrap(err, "could not broadcast attestation")
	}

	attRoot, err := ssz.HashTreeRoot(att)
	if err != nil {
		log.WithError(err).Error("Failed to hash attestation")
	}

	log.WithFields(logrus.Fields{
		"attRoot": hex.EncodeToString(attRoot[:]),
		"attDataRoot": hex.EncodeToString(att.Data.BeaconBlockRoot),
	}).Debug("Broadcasting attestation")

	if err := c.ReceiveAttestationNoPubsub(ctx, att); err != nil {
		return err
	}

	processedAtt.Inc()
	return nil
}

// ReceiveAttestationNoPubsub is a function that defines the operations that are preformed on
// attestation that is received from regular sync. The operations consist of:
//  1. Validate attestation, update validator's latest vote
//  2. Apply fork choice to the processed attestation
//  3. Save latest head info
func (c *ChainService) ReceiveAttestationNoPubsub(ctx context.Context, att *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveAttestationNoPubsub")
	defer span.End()

	// Delay attestation inclusion until the attested slot is in the past.
	slot, err := c.waitForAttInclDelay(ctx, att)
	if err != nil {
		return errors.Wrap(err, "could not delay attestation inclusion")
	}

	// Update forkchoice store for the new attestation
	if err := c.forkChoiceStore.OnAttestation(ctx, att); err != nil {
		return errors.Wrap(err, "could not process block from fork choice service")
	}
	log.WithFields(logrus.Fields{
		"attSlot":     slot,
		"attDataRoot": hex.EncodeToString(att.Data.BeaconBlockRoot),
	}).Debug("Finished updating fork choice store for attestation")

	// Run fork choice for head block after updating fork choice store.
	headRoot, err := c.forkChoiceStore.Head(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head from fork choice service")
	}
	headBlk, err := c.beaconDB.Block(ctx, bytesutil.ToBytes32(headRoot))
	if err != nil {
		return errors.Wrap(err, "could not compute state from block head")
	}
	log.WithFields(logrus.Fields{
		"headSlot": headBlk.Slot,
		"headRoot": hex.EncodeToString(headRoot),
	}).Debug("Finished applying fork choice for attestation")

	// Skip checking for competing attestation's target roots at epoch boundary.
	if helpers.IsEpochEnd(slot) {
		targetRoot, err := helpers.BlockRoot(c.headState, att.Data.Target.Epoch)
		if err != nil {
			return errors.Wrapf(err, "could not get target root for epoch %d", att.Data.Target.Epoch)
		}
		isCompetingAtts(targetRoot, att.Data.Target.Root[:])
	}

	// Save head info after running fork choice.
	if err := c.saveHead(ctx, headBlk, bytesutil.ToBytes32(headRoot)); err != nil {
		return errors.Wrap(err, "could not save head")
	}

	processedAttNoPubsub.Inc()
	return nil
}

// waitForAttInclDelay waits until the next slot because attestation can only affect
// fork choice of subsequent slot. This is to delay attestation inclusion for fork choice
// until the attested slot is in the past.
func (c *ChainService) waitForAttInclDelay(ctx context.Context, a *ethpb.Attestation) (uint64, error) {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.forkchoice.waitForAttInclDelay")
	defer span.End()

	s, err := c.beaconDB.State(ctx, bytesutil.ToBytes32(a.Data.Target.Root))
	if err != nil {
		return 0, errors.Wrap(err, "could not get state")
	}
	slot, err := helpers.AttestationDataSlot(s, a.Data)
	if err != nil {
		return 0, errors.Wrap(err, "could not get attestation slot")
	}

	nextSlot := slot + 1
	duration := time.Duration(nextSlot*params.BeaconConfig().SecondsPerSlot) * time.Second
	timeToInclude := time.Unix(int64(s.GenesisTime), 0).Add(duration)

	time.Sleep(time.Until(timeToInclude))
	return slot, nil
}

// This checks if the attestation is from a competing chain, emits warning and updates metrics.
func isCompetingAtts(headTargetRoot []byte, attTargetRoot []byte) {
	if !bytes.Equal(attTargetRoot, headTargetRoot) {
		log.WithFields(logrus.Fields{
			"attTargetRoot":  hex.EncodeToString(attTargetRoot),
			"headTargetRoot": hex.EncodeToString(headTargetRoot),
		}).Warn("target heads different from new attestation")
		competingAtts.Inc()
	}
}
