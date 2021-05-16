// The aim is to check for duplicate attestations at Validator Launch for the same keystore
// If it is detected , a doppelganger exists, so alert the user and exit.
// This is is done for two epochs. That is better than starting a duplicate validator and causing slashing.
package client

import (
	"context"
	"fmt"
	"time"
	"bytes"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/sirupsen/logrus"
	//"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

type DuplicateDetection struct {
	slot            uint64
	index           types.ValidatorIndex
	remainingEpochs types.Epoch
}

// The Public Keys of this Validator. Retrieve once.
var validatingPublicKeys [][48]byte

// N epochs to check
var NoEpochsToCheck  uint8

// Starts the Doppelganger detection
func (v *validator) startDoppelgangerService(ctx context.Context) error {
	log.Info("Doppelganger Service started")

	// N epochs to check
	NoEpochsToCheck = params.BeaconConfig().DuplicateValidatorEpochsCheck

	// The Public Keys of this Validator. Retrieve once.
	var err error
	validatingPublicKeys, err = v.retrieveValidatingPublicKeys(ctx)
	if err != nil {
		return err
	}

	// Get the currentEpoch and genesisEpoch
	slot := <-v.NextSlot()
	currentEpoch := helpers.SlotToEpoch(slot)
	genesisEpoch := params.BeaconConfig().GenesisEpoch
		//types.Epoch(v.genesisTime / uint64(params.BeaconConfig().SlotsPerEpoch.Mul(params.BeaconConfig().SecondsPerSlot)))


	// Returns number of slots since the start of the epoch.
	sinceStart := slot % params.BeaconConfig().SlotsPerEpoch

	endingSlot := slot.Add(uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(NoEpochsToCheck))))
	endingSlot += sinceStart

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

		// Are we done?
		if slot  >=  endingSlot{
			log.Info("Doppelganger Service - finished the epoch checks for duplicates ")
			return nil
		}

		// Detect a doppelganger in the previous Epoch
		foundDuplicate, sig, err := v.detectDoppelganger( ctx,helpers.SlotToEpoch(slot)-1)
		if err != nil {
			return err
		}
		if foundDuplicate {
			log.WithFields(logrus.Fields{
				"pubKey": sig,
			}).Info("Doppelganger detected! Validator key 0x%x seems to be running elsewhere."+
				"This process will exit, avoiding a proposer or attester slashing event."+
				"Please ensure you are not running your validator in two places simultaneously.", sig)
			return errors.New("Doppelganger detected")
		}

		// Sleep time between now and start of next slot.
		nextSlotTime := v.SlotDeadline(slot)
		timeRemaining := time.Until(nextSlotTime)
		// Still time till next slot? sleep through and loop again
		if timeRemaining >= 0 {
			log.WithFields(logrus.Fields{
				"timeRemaining": timeRemaining,
			}).Info("Sleeping until the next slot - Doppelganger detection")
			time.Sleep(timeRemaining)

			continue
		} else {
			// this should not happen. Clock is off? Sleep for 1 slot
			log.WithFields(logrus.Fields{
				"timeRemaining": timeRemaining,
			}).Info("Time remaining till next slot is negative! Sleep a slot - Doppelganger detection")
			time.Sleep(time.Second * time.Duration(params.BeaconConfig().SecondsPerSlot))
		}
		// Get next slot
		slot = <-v.NextSlot()
		log.WithField("slot", fmt.Sprintf("Doppelganger Service - new Slot %d", slot))

	}

}

// Doppelganger detection
// At the start of every epoch, retrieve all the beacon-chain blocks
// and check for blocks with the same pubKey as this validator
func (v *validator) detectDoppelganger(ctx context.Context,epoch types.Epoch) (bool, []byte, error) {
	result := make([]byte, 1)

	// Get all this validator 's attestation in the current slot so far requiring a duplicate detection check
	req := &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Epoch{Epoch: epoch}}
	blks, err := v.beaconClient.ListBlocks(ctx, req)
	if err != nil {
		return false, result,errors.Wrap(err, "Doppelganger service - failed to get blocks from beacon-chain")
	}
	for _, blk := range blks.BlockContainers {
		dupProposalKey ,err :=v.checkBlockProposer(ctx,blk)
		if err != nil {
			return false,nil, err
		}
		if dupProposalKey != nil {
			return true,dupProposalKey ,nil
		}
		//dupAttestorKey, err := v.checkBlockAttestors(ctx,blk)
		//if err != nil {
		//	return false,nil, err
		//}
		//if dupAttestorKey != nil {
		//	return true,dupAttestorKey ,nil
		//}
	}
	return false,nil ,nil

	/*
	parentState, err := s.cfg.StateGen.StateByRoot(ctx, bytesutil.ToBytes32(blk.Block.ParentRoot))*/
}


// Given a SignedBlock determine if the proposer's idx is that of one this validator's pubKeys
func (v *validator) checkBlockProposer(ctx context.Context,blk *ethpb.BeaconBlockContainer) ([]byte,error){
	for _,key := range validatingPublicKeys{
		valID,err := v.validatorClient.ValidatorIndex(ctx, &ethpb.ValidatorIndexRequest{PublicKey: key[:]})
		if err != nil {
			return nil, err
		}
		vID,err :=valID.Index.MarshalSSZ()
		if err != nil {
			return nil, err
		}
		pID,err := blk.Block.Block.ProposerIndex.MarshalSSZ()
		if err != nil {
			return nil, err
		}
		if bytes.Equal(vID,pID) {
			// Found a doppelganger. This validator proposed a block in somewhere else
			return key[:], nil
		}
	}
	return nil,nil
}

// Given a SignedBlock determine if the any of the attestors' idx is that of one this validator's pubKeys
/*func (v *validator) checkBlockAttestors(ctx context.Context,blk *ethpb.BeaconBlockContainer) ([]byte,error){
	for _,key := range validatingPublicKeys{
		valID,err := v.validatorClient.ValidatorIndex(ctx, &ethpb.ValidatorIndexRequest{PublicKey: key[:]})
		if err != nil {
			return nil, err
		}
		vID,err :=valID.Index.MarshalSSZ()
		if err != nil {
			return nil, err
		}
		blk.Block.Block.Body.Attestations
		// Get the attestations committe and ensure non is our validator
		//if blk.Block.Block.Body.Attestations == valIdx.Index{
			// Found a doppelganger. This validator proposed a block in somewhere else
		//	return key[:], nil
		//}
	}
	return nil,nil

		//	// Use the target state to verify attesting indices are valid.
		//	committee, err := helpers.BeaconCommitteeFromState(baseState, a.Data.Slot, a.Data.CommitteeIndex)
		//	if err != nil {
		//		return err
		//	}
		//	indexedAtt, err := attestationutil.ConvertToIndexed(ctx, a, committee)
		//	if err != nil {
		//		return err
		//	}
		//	if err := attestationutil.IsValidAttestationIndices(ctx, indexedAtt); err != nil {
		//		return err
		//	}
}*/

func (v *validator)retrieveValidatingPublicKeys(ctx context.Context)([][48]byte,error){
	validatingPublicKeys, err := v.keyManager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return nil, err
	}
	return validatingPublicKeys,nil

}