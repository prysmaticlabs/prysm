package client

import (
	"context"
	"time"

	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/config/params"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/primitives"
	prysmTime "github.com/prysmaticlabs/prysm/v5/time"
	"github.com/prysmaticlabs/prysm/v5/time/slots"
)

func (v *validator) SubmitBid(ctx context.Context, slot primitives.Slot, pubKey [fieldparams.BLSPubkeyLength]byte) {
	if params.BeaconConfig().EPBSForkEpoch > slots.ToEpoch(slot) {
		return
	}

	v.waitToSubmitBid(ctx, slot)

	if err := v.SubmitHeader(ctx, slot+1, pubKey); err != nil {
		log.WithError(err).Error("Failed to submit header")
		return
	}

	v.waitToSubmitPayload(ctx, slot+1)

	if err := v.SubmitExecutionPayloadEnvelope(ctx, slot+1, pubKey); err != nil {
		log.WithError(err).Error("Failed to submit execution payload")
		return
	}
}

func (v *validator) waitToSubmitBid(ctx context.Context, slot primitives.Slot) {
	startTime := slots.StartTime(v.genesisTime, slot)
	dutyTime := startTime.Add(11 * time.Second)
	wait := prysmTime.Until(dutyTime)
	if wait <= 0 {
		return
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return
	case <-t.C:
		return
	}
}

func (v *validator) waitToSubmitPayload(ctx context.Context, slot primitives.Slot) {
	startTime := slots.StartTime(v.genesisTime, slot)
	dutyTime := startTime.Add(6 * time.Second)
	wait := prysmTime.Until(dutyTime)
	if wait <= 0 {
		return
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return
	case <-t.C:
		return
	}
}
