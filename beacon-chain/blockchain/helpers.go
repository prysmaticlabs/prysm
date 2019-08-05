package blockchain

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/db"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"go.opencensus.io/trace"
)

// waitForAttInclDelay waits until the next slot because attestation can only affect
// fork choice of subsequent slot. This is to delay attestation inclusion for fork choice
// until the attested slot is in the past.
func waitForAttInclDelay(ctx context.Context, db *db.BeaconDB, a *ethpb.Attestation) error {
	ctx, span := trace.StartSpan(ctx, "beacon-chain.forkchoice.waitForAttInclDelay")
	defer span.End()

	s, err := db.ForkChoiceState(ctx, a.Data.Target.Root)
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
