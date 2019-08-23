package blockchain

import (
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
	log.WithFields(logrus.Fields{
		"attDataRoot": hex.EncodeToString(att.Data.BeaconBlockRoot),
	}).Info("Broadcasting attestation")

	return c.ReceiveAttestationNoPubsub(ctx, att)
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
	if err := c.waitForAttInclDelay(ctx, att); err != nil {
		return errors.Wrap(err, "could not delay attestation inclusion")
	}

	// Update forkchoice store for the new attestation
	if err := c.forkChoiceStore.OnAttestation(ctx, att); err != nil {
		return errors.Wrap(err, "could not process block from fork choice service")
	}
	root, err := ssz.SigningRoot(att)
	if err != nil {
		return errors.Wrap(err, "could not sign root attestation")
	}
	log.WithFields(logrus.Fields{
		"attRoot":     hex.EncodeToString(root[:]),
		"attDataRoot": hex.EncodeToString(att.Data.BeaconBlockRoot),
	}).Info("Finished updating fork choice store for attestation")

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
	}).Info("Finished applying fork choice")

	// Save head info after running fork choice.
	if err := c.saveHead(ctx, headBlk, bytesutil.ToBytes32(headRoot)); err != nil {
		return errors.Wrap(err, "could not save head")
	}

	return nil
}

// waitForAttInclDelay waits until the next slot because attestation can only affect
// fork choice of subsequent slot. This is to delay attestation inclusion for fork choice
// until the attested slot is in the past.
func (c *ChainService) waitForAttInclDelay(ctx context.Context, a *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.forkchoice.waitForAttInclDelay")
	defer span.End()

	s, err := c.beaconDB.State(ctx, bytesutil.ToBytes32(a.Data.Target.Root))
	if err != nil {
		return errors.Wrap(err, "could not get state")
	}
	slot, err := helpers.AttestationDataSlot(s, a.Data)
	if err != nil {
		return errors.Wrap(err, "could not get attestation slot")
	}

	nextSlot := slot + 1
	duration := time.Duration(nextSlot*params.BeaconConfig().SecondsPerSlot) * time.Second
	timeToInclude := time.Unix(int64(s.GenesisTime), 0).Add(duration)

	time.Sleep(time.Until(timeToInclude))
	return nil
}
