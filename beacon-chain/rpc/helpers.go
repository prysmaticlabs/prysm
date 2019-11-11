package rpc

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// Computes validator assignments for an epoch and validator index using archived committee
// information, archived balances, and a set of active validators.
func archivedValidatorCommittee(
	epoch uint64,
	validatorIndex uint64,
	archivedInfo *ethpb.ArchivedCommitteeInfo,
	activeIndices []uint64,
	archivedBalances []uint64,
) ([]uint64, uint64, uint64, uint64, error) {
	committeeCount := archivedInfo.CommitteeCount
	proposerSeed := bytesutil.ToBytes32(archivedInfo.ProposerSeed)
	attesterSeed := bytesutil.ToBytes32(archivedInfo.AttesterSeed)

	startSlot := helpers.StartSlot(epoch)
	proposerIndexToSlot := make(map[uint64]uint64)
	for slot := startSlot; slot < startSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {
		seedWithSlot := append(proposerSeed[:], bytesutil.Bytes8(slot)...)
		seedWithSlotHash := hashutil.Hash(seedWithSlot)
		i, err := archivedProposerIndex(activeIndices, archivedBalances, seedWithSlotHash)
		if err != nil {
			return nil, 0, 0, 0, errors.Wrapf(err, "could not check proposer at slot %d", slot)
		}
		proposerIndexToSlot[i] = slot
	}
	for slot := startSlot; slot < startSlot+params.BeaconConfig().SlotsPerEpoch; slot++ {
		var countAtSlot = uint64(len(activeIndices)) / params.BeaconConfig().SlotsPerEpoch / params.BeaconConfig().TargetCommitteeSize
		if countAtSlot > params.BeaconConfig().MaxCommitteesPerSlot {
			countAtSlot = params.BeaconConfig().MaxCommitteesPerSlot
		}
		if countAtSlot == 0 {
			countAtSlot = 1
		}
		for i := uint64(0); i < countAtSlot; i++ {
			epochOffset := i + (slot%params.BeaconConfig().SlotsPerEpoch)*countAtSlot
			committee, err := helpers.ComputeCommittee(activeIndices, attesterSeed, epochOffset, committeeCount)
			if err != nil {
				return nil, 0, 0, 0, errors.Wrap(err, "could not compute committee")
			}
			for _, index := range committee {
				if validatorIndex == index {
					proposerSlot, _ := proposerIndexToSlot[validatorIndex]
					return committee, i, slot, proposerSlot, nil
				}
			}
		}
	}
	return nil, 0, 0, 0, fmt.Errorf("could not find committee for validator index %d", validatorIndex)
}

func archivedProposerIndex(activeIndices []uint64, activeBalances []uint64, seed [32]byte) (uint64, error) {
	length := uint64(len(activeIndices))
	if length == 0 {
		return 0, errors.New("empty indices list")
	}
	maxRandomByte := uint64(1<<8 - 1)
	for i := uint64(0); ; i++ {
		candidateIndex, err := helpers.ComputeShuffledIndex(i%length, length, seed, true)
		if err != nil {
			return 0, err
		}
		b := append(seed[:], bytesutil.Bytes8(i/32)...)
		randomByte := hashutil.Hash(b)[i%32]
		effectiveBalance := activeBalances[candidateIndex]
		if effectiveBalance >= params.BeaconConfig().MaxEffectiveBalance {
			// if the actual balance is greater than or equal to the max effective balance,
			// we just determine the proposer index using config.MaxEffectiveBalance.
			effectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		}
		if effectiveBalance*maxRandomByte >= params.BeaconConfig().MaxEffectiveBalance*uint64(randomByte) {
			return candidateIndex, nil
		}
	}
}
