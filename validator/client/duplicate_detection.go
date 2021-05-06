// The aim is to check for duplicate attestations at Validator Launch for the same keystore
// If it is detected , a doppleganger exists, so alert the user and exit.
// This is is done for two epochs. That is better than starting and causing slashing.
package client

import (
	"context"
	"time"

	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
	"github.com/sirupsen/logrus"
)

type DuplicateDetection struct {
	slotClock	uint64
	index     types.ValidatorIndex
}

// Starts the Doppelganger detection
func (v *validator) startDoppelgangerService(ctx context.Context) error {
	log.Info("Doppleganger Service started")

	//get the current_epoch and genesis_epoch
	slot := <-v.NextSlot()
	//currentEpoch := types.Epoch(slot / params.BeaconConfig().SlotsPerEpoch)
	//genesisEpoch := types.Epoch(v.genesisTime / uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot)))

	// sleep to next slot, detect doppelganger, repeat until 2 epochs from now
	for {

		// return time between now and start of next slot
		// sleep until the next slot
		currentTime := timeutils.Now()
		nextSlotTime := v.SlotDeadline(slot)
		timeRemaining := nextSlotTime.Sub(currentTime)
		if timeRemaining > 0{
			log.WithFields(logrus.Fields{
				"timeRemaining": timeRemaining,
			}).Info("Sleeping until the next slot - Doppelganger detection")
			time.Sleep(timeRemaining)
			return nil;
		}

	}

	/*

		select {
		case <-ticker.C:
			if timeRemaining >= time.Second {
				log.WithFields(logFields).Infof(
					"%s until chain genesis",
					timeRemaining.Truncate(time.Second),
				)
			}
		case <-ctx.Done():
			log.Debug("Context closed, exiting routine")
			return
		}

	ctx, span := trace.StartSpan(ctx, "validator.waitToSlotTwoThirds")
	defer span.End()

	twoThird := oneThird + oneThird
	delay := twoThird

	startTime := slotutil.SlotStartTime(v.genesisTime, slot)
	finalTime := startTime.Add(delay)
	wait := timeutils.Until(finalTime)
	if wait <= 0 {
		return
	}
	t := time.NewTimer(wait)
	defer t.Stop()
	select {
	case <-ctx.Done():
		traceutil.AnnotateError(span, ctx.Err())
		return
	case <-t.C:
		return
	}

	/*genesisTime time.Time,
	secondsPerSlot uint64,
	since, until func(time.Time) time.Duration,
	after func(time.Duration) <-chan time.Time) {

	d := time.Duration(secondsPerSlot) * time.Second

	go func() {
		sinceGenesis := since(genesisTime)

		var nextTickTime time.Time
		var slot types.Slot
		if sinceGenesis < d {
			// Handle when the current time is before the genesis time.
			nextTickTime = genesisTime
			slot = 0
		} else {
			nextTick := sinceGenesis.Truncate(d) + d
			nextTickTime = genesisTime.Add(nextTick)
			slot = types.Slot(nextTick / d)
		}

		for {
			waitTime := until(nextTickTime)
			select {
			case <-after(waitTime):
				s.c <- slot
				slot++
				nextTickTime = nextTickTime.Add(d)
			case <-s.done:
				return
			}
		}
	}()*/

}