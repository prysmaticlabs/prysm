package blockchain

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

// AttestationReceiver interface defines the methods in the blockchain service which
// directly receives a new attestation from other services and applies the full processing pipeline.
type AttestationReceiver interface {
	ReceiveAttestation(ctx context.Context, att *ethpb.Attestation) error
}

// ReceiveAttestation is a function that defines the operations that are preformed on
// any attestation that is received from p2p layer or rpc.
func (c *ChainService) ReceiveAttestation(ctx context.Context, att *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.blockchain.ReceiveAttestation")
	defer span.End()

	root, err := ssz.SigningRoot(att)
	if err != nil {
		return errors.Wrap(err, "could not sign root attestation")
	}

	// broadcast the attestation to other peers.
	c.p2p.Broadcast(ctx, att)

	// Delay attestation inclusion until the attested slot is in the past.
	if err := c.waitForAttInclDelay(ctx, att); err != nil {
		return errors.Wrap(err, "could not delay attestation inclusion")
	}

	c.forkChoiceStore.OnTick(uint64(time.Now().Unix()))

	if err := c.forkChoiceStore.OnAttestation(ctx, att); err != nil {
		return errors.Wrap(err, "could not process block from fork choice service")
	}
	log.WithFields(logrus.Fields{
		"root": hex.EncodeToString(root[:]),
	}).Info("Finished update fork choice store for attestation")

	// Run fork choice for head block and head block.
	// The spec says to run fork choice on every aggregated attestation, we can
	// remove this if we don't feel it's necessary. But i think this will be good
	// for interopt to gain stability.
	headRoot, err := c.forkChoiceStore.Head(ctx)
	if err != nil {
		return errors.Wrap(err, "could not get head from fork choice service")
	}
	headBlk, err := c.deprecatedBeaconDB.Block(bytesutil.ToBytes32(headRoot))
	if err != nil {
		return errors.Wrap(err, "could not compute state from block head")

	}

	c.canonicalRootsLock.Lock()
	defer c.canonicalRootsLock.Unlock()
	c.headSlot = headBlk.Slot
	c.canonicalRoots[headBlk.Slot] = headRoot
	if err := c.db.SaveHeadBlockRoot(ctx, bytesutil.ToBytes32(headRoot)); err != nil {
		return errors.Wrap(err, "could not save head root in DB")
	}

	log.WithFields(logrus.Fields{
		"slots": headBlk.Slot,
		"root":  hex.EncodeToString(headRoot),
	}).Info("Finished fork choice for attestation")

	return nil
}

// waitForAttInclDelay waits until the next slot because attestation can only affect
// fork choice of subsequent slot. This is to delay attestation inclusion for fork choice
// until the attested slot is in the past.
func (c *ChainService) waitForAttInclDelay(ctx context.Context, a *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.forkchoice.waitForAttInclDelay")
	defer span.End()

	s, err := c.db.State(ctx, bytesutil.ToBytes32(a.Data.Target.Root))
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
