// The aim is to check for duplicate attestations at Validator Launch for the same keystore
// If it is detected , a doppelganger exists, so alert the user and exit.
// This is is done for N(two) epochs. That is better than starting a duplicate validator and getting slashed.
package client

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	iface "github.com/prysmaticlabs/prysm/beacon-chain/state/interface"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
	//"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

type DuplicateDetection struct {
	Slot         types.Slot
	DuplicateKey []byte
}

// The Public Keys and Indices of this Validator. Retrieve once.
var validatingPublicKeys [][48]byte
var valIndices []types.ValidatorIndex

// N epochs to check
var NoEpochsToCheck uint8

// Starts the Doppelganger detection
func (v *validator) StartDoppelgangerService(ctx context.Context) error {
	log.Info("Doppelganger service started")

	// N epochs to check
	NoEpochsToCheck = params.BeaconConfig().DuplicateValidatorEpochsCheck

	// Public Keys of this Validator. Retrieve once.
	var err error
	valIndices, validatingPublicKeys, err = v.retrieveValidatingPublicKeys(ctx)
	if err != nil {
		return err
	}

	// Get the currentEpoch and genesisEpoch
	slot := <-v.NextSlot()
	currentEpoch := helpers.SlotToEpoch(slot)
	genesisEpoch := params.BeaconConfig().GenesisEpoch

	// Counting N epochs from the starting Slot(substract 1 since we alwasy check slot-1 at the start).
	endingSlot := slot.Add(uint64(params.BeaconConfig().SlotsPerEpoch.Mul(uint64(NoEpochsToCheck)))).Sub(1)

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
		if slot >= endingSlot.Sub(1) {
			log.Info("Doppelganger service - finished the epoch checks for duplicates ")
			return nil
		}

		log.Infof("Doppelganger service - starts the check at previous slot %d", slot-1)
		// Detect a doppelganger in the previous Slot
		dupIndex, err := v.detectDoppelganger(ctx, slot-1)
		if err != nil {
			return err
		}
		if dupIndex != nil {
			log.Infof("Doppelganger detected! Validator key 0x%x seems to be running elsewhere."+
				"This process will exit, avoiding a proposer or attester slashing event."+
				"Please ensure you are not running your validator in two places simultaneously.", dupIndex)

			// Broadcast the findings at the slot. The Node process will listen and shutdown.
			ret := &DuplicateDetection{}
			ret.DuplicateKey = dupIndex
			ret.Slot = slot
			v.duplicateFeed.Send(ret)
			return nil
		}

		// Sleep time between now and start of next slot.
		nextSlotTime := v.SlotDeadline(slot)
		timeRemaining := time.Until(nextSlotTime)
		// Still time till next slot? sleep through and loop again
		if timeRemaining >= 0 {
			log.WithFields(logrus.Fields{
				"timeRemaining": timeRemaining,
			}).Info("Sleeping until the next slot - Doppelganger service")
			time.Sleep(timeRemaining)
			continue
		} // else condition should not happen.Clock is off?
		// Or maybe Detection process at previous slot took too long

		// Get next slot
		slot = <-v.NextSlot()
		log.Infof("Doppelganger service - new Slot %d", slot)
	}

}

// Doppelganger detection
// Retrieve all the beacon-chain blocks at the given slot
// and check for blocks with the same pubKey as this validator
func (v *validator) detectDoppelganger(ctx context.Context, slot types.Slot) ([]byte, error) {

	log.Infof("Doppelganger service - iterating through blocks ")
	// Get all this validator's attestation in the current slot so far requiring a duplicate detection check
	req := &ethpb.ListBlocksRequest{QueryFilter: &ethpb.ListBlocksRequest_Slot{Slot: slot}}
	blks, err := v.beaconClient.ListBlocks(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Doppelganger service - failed to get blocks from beacon-chain")
	}

	log.Infof("Doppelganger service - interating through signed Blocks at slot  %d", slot)
	for _, blk := range blks.BlockContainers {

		log.WithField("blk.BlockRoot", fmt.Sprintf("Doppelganger service - check Block for a duplicate proposer"+
			" at root %#x", blk.BlockRoot))
		dupProposalKey, err := v.checkBlockProposer(ctx, blk)
		if err != nil {
			log.Infof("Doppelganger service - failed on return from possible "+
				"proposer check at root %#x", blk.BlockRoot)
			return nil, errors.Wrapf(err,
				"Doppelganger service - failed on return from Prosposer check at root %#x", blk.BlockRoot)
		}
		if dupProposalKey != nil {
			log.Info("Doppelganger service - found a proposer duplicate")
			return dupProposalKey, nil
		}

		log.WithField("blk.BlockRoot", fmt.Sprintf("Doppelganger service - check Block for a duplicate attestor"+
			" at root %#x", blk.BlockRoot))
		dupAttestorKey, err := v.checkBlockAttestors(ctx, blk)
		if err != nil {
			log.Infof("Doppelganger service - failed on return from possible "+
				"attestor check at root %#x", blk.BlockRoot)
			return nil, err
		}
		if dupAttestorKey != nil {
			log.Info("Doppelganger service - found an attestor duplicate")
			return dupAttestorKey, nil
		}
	}
	return nil, nil
}

// Given a SignedBlock determine if the proposer's idx is that of one this validator's pubKeys
func (v *validator) checkBlockProposer(ctx context.Context, blk *ethpb.BeaconBlockContainer) ([]byte, error) {
	log.Info("Doppelganger service - inside  BeaconBlock looking for a duplicate proposer ")
	for _, key := range validatingPublicKeys {
		valID, err := v.validatorClient.ValidatorIndex(ctx, &ethpb.ValidatorIndexRequest{PublicKey: key[:]})
		if err != nil {
			return nil, errors.Wrapf(err,
				"Doppelganger service - cannot retrieve Validator index %#x for  Prosposer check", key[:])
		}
		vID, err := valID.Index.MarshalSSZ()
		if err != nil {
			return nil, errors.Wrapf(err,
				"Doppelganger service - cannot serialize Validator index %#x for  Prosposer check", key[:])
		}
		pID, err := blk.Block.Block.ProposerIndex.MarshalSSZ()
		if err != nil {
			return nil, errors.Wrapf(err,
				"Doppelganger service - cannot serialize proposer index %#x for  Prosposer check", blk.Block.Block.ProposerIndex)
		}
		if bytes.Equal(vID, pID) {
			// Found a doppelganger. This validator proposed a block somewhere else
			return key[:], nil
		}
	}
	return nil, nil
}

// Given a SignedBlock determine if any of the attestors' idx is that of one this validator's
func (v *validator) checkBlockAttestors(ctx context.Context, blk *ethpb.BeaconBlockContainer) ([]byte, error) {

	log.Info("Doppelganger service - inside  BeaconBlock looking for a duplicate attestor ")
	targetStates := make(map[[32]byte]iface.ReadOnlyBeaconState)

	for _, att := range blk.Block.Block.Body.Attestations {
		tr := bytesutil.ToBytes32(att.Data.Target.Root)
		s := targetStates[tr]

		committee, err := helpers.BeaconCommitteeFromState(s, att.Data.Slot, att.Data.CommitteeIndex)
		if err != nil {
			return nil, errors.Wrap(err, "Doppelganger service - could not get committee")
		}
		indices, err := attestationutil.AttestingIndices(att.AggregationBits, committee)
		if err != nil {
			return nil, errors.Wrap(err, "Doppelganger service - could not get attesting indices")
		}
		for _, i := range indices {
			for _, vID := range valIndices {
				if i == uint64(vID) {
					v, err := vID.MarshalSSZ()
					if err != nil {
						return nil, errors.Wrap(err, "Doppelganger service - could not get "+
							"marshal index into []bytes")
					}
					return v, nil
				}
			}
		}
	}

	return nil, nil
}

// Load the PublicKeys and the corresponding Indices of the Validator. Do it once.
func (v *validator) retrieveValidatingPublicKeys(ctx context.Context) ([]types.ValidatorIndex, [][48]byte, error) {
	validatingPublicKeys, err := v.keyManager.FetchValidatingPublicKeys(ctx)
	if err != nil {
		return nil, nil, err
	}

	valIndices = make([]types.ValidatorIndex, len(validatingPublicKeys))

	// Convert the ValidatingKeys to an array of Indices to be used by Committee retrieval.
	for _, key := range validatingPublicKeys {
		valID, err := v.validatorClient.ValidatorIndex(ctx, &ethpb.ValidatorIndexRequest{PublicKey: key[:]})
		if err != nil {
			return nil, nil, err
		}
		valIndices = append(valIndices, valID.Index)
	}
	return valIndices, validatingPublicKeys, nil

}
