// The aim is to check for duplicate attestations at Validator Launch for the same keystore
// If it is detected , a doppelganger exists, so alert the user and exit.
// This is is done for two epochs. That is better than starting a duplicate validator and causing slashing.
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/timeutils"
	"github.com/sirupsen/logrus"
)

type DuplicateDetection struct {
	slot            uint64
	index           types.ValidatorIndex
	remainingEpochs types.Epoch
}

var EpochsToCheck = params.BeaconConfig().DuplicateValidatorEpochsCheck

// Starts the Doppelganger detection
func (v *validator) startDoppelgangerService(ctx context.Context) error {
	log.Info("Doppelganger Service started")

	//get the currentEpoch and genesisEpoch
	slot := <-v.NextSlot()
	currentEpoch := helpers.SlotToEpoch(slot)
	genesisEpoch := params.BeaconConfig().GenesisEpoch
		//types.Epoch(v.genesisTime / uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot)))

	// Ensure currentEpoch > genesisEpoch
	// If the current is equal to the genesis or prior to genesis then no duplicate check is performed.
	if genesisEpoch >= currentEpoch {
		return nil
	}

	// Either a proposal or attestation duplicate is detected at one of the slots in a 2 epoch period which results
	// in a forced validator stop, or none is found and flow continues in the validator runner.
	// Steps:
	// 1. Detect doppelganger.
	// 2. if not found sleep till next slot; Go to 4
	// 3. If found exit
	// 4. repeat for 2 epochs, Go to 1.
	for {

		// Check if current slot is the end of the epoch. If so subtract from epochs to check counter.
		if helpers.IsEpochEnd(slot) {
			EpochsToCheck -= 1
			log.WithField("EpochsToCheck", fmt.Sprintf("Doppelganger Service - end of current Epoch,"+
				" 1 less Epoch to check %d Epochs", EpochsToCheck))
		}
		if EpochsToCheck == 0 {
			log.Info("Doppelganger Service - finished the epoch checks for duplicates ")
			return nil
		}

		// return time between now and start of next slot
		// sleep until the next slot
		currentTime := timeutils.Now()
		nextSlotTime := v.SlotDeadline(slot)
		// Detect a doppelganger
		foundDuplicate, pubKey, err := v.detectDoppelganger(slot)
		if err != nil {
			return err
		}
		if foundDuplicate {
			log.WithFields(logrus.Fields{
				"pubKey": pubKey,
			}).Info("Doppelganger detected! Validator key 0x%x seems to be running elsewhere."+
				"This process will exit, avoiding a proposer or attester slashing event."+
				"Please ensure you are not running your validator in two places simultaneously.", pubKey)
			return errors.New("Doppelganger detected")
		}

		timeRemaining := nextSlotTime.Sub(currentTime)
		// Still time till next slot? sleep through and loop again
		if timeRemaining > 0 {
			log.WithFields(logrus.Fields{
				"timeRemaining": timeRemaining,
			}).Info("Sleeping until the next slot - Doppelganger detection")
			time.Sleep(timeRemaining)
			// Get next slot
			slot = <-v.NextSlot()
			continue
		} else {
			// this should not happen. Slot in the future? Clock is off?
			log.WithFields(logrus.Fields{
				"timeRemaining": timeRemaining,
			}).Fatal("Time remaining till next slot is negative!")
			return errors.New(fmt.Sprintf("Time remaining till next slot is negative %d milliseconds!",
				int64(timeRemaining.Truncate(time.Millisecond))))
		}

	}

}

// Starts the Doppelganger detection
func (v *validator) detectDoppelganger(slot types.Slot) (bool, [][48]byte, error) {
	result := make([][48]byte, 1)

	// Get all this validator 's attestation in the current slot so far requiring a duplicate detection check

	// Check if any match this validator
	return false, result, nil

}
