package altair

import (
	"bytes"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/attestationutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

type matchedSource bool
type matchedTarget bool
type matchedHead bool

// HasValidatorFlag returns true if the flag at position has set.
func HasValidatorFlag(flag, flagPosition uint8) bool {
	return ((flag >> flagPosition) & 1) == 1
}

// AddValidatorFlag adds new validator flag to existing one.
func AddValidatorFlag(flag, flagPosition uint8) uint8 {
	return flag | (1 << flagPosition)
}

// This retrieves a map of attestation scoring based on Altair's participation flag indices.
// This is used to facilitate process attestation during state transition and during upgrade to altair state.
func attestationParticipationFlagIndices(beaconState state.BeaconStateAltair, data *ethpb.AttestationData, delay types.Slot) (map[uint8]bool, error) {
	currEpoch := helpers.CurrentEpoch(beaconState)
	var justifiedCheckpt *ethpb.Checkpoint
	if data.Target.Epoch == currEpoch {
		justifiedCheckpt = beaconState.CurrentJustifiedCheckpoint()
	} else {
		justifiedCheckpt = beaconState.PreviousJustifiedCheckpoint()
	}

	matchingSource, matchingTarget, matchingHead, err := matchingStatus(beaconState, data, justifiedCheckpt)
	if err != nil {
		return nil, err
	}
	if !matchingSource {
		return nil, errors.New("source epoch does not match")
	}

	participatedFlags := make(map[uint8]bool)
	cfg := params.BeaconConfig()
	sourceFlagIndex := cfg.TimelySourceFlagIndex
	targetFlagIndex := cfg.TimelyTargetFlagIndex
	headFlagIndex := cfg.TimelyHeadFlagIndex
	slotsPerEpoch := cfg.SlotsPerEpoch
	sqtRootSlots := mathutil.IntegerSquareRoot(uint64(slotsPerEpoch))
	if matchingSource && delay <= types.Slot(sqtRootSlots) {
		participatedFlags[sourceFlagIndex] = true
	}
	if matchingTarget && delay <= slotsPerEpoch {
		participatedFlags[targetFlagIndex] = true
	}
	matchingHeadTarget := bool(matchingHead) && bool(matchingTarget)
	if matchingHeadTarget && delay == cfg.MinAttestationInclusionDelay {
		participatedFlags[headFlagIndex] = true
	}
	return participatedFlags, nil
}

// This returns the matching statues for attestation data's source target and head.
func matchingStatus(beaconState state.BeaconState, data *ethpb.AttestationData, cp *ethpb.Checkpoint) (s matchedSource, t matchedTarget, h matchedHead, err error) {
	s = matchedSource(attestationutil.CheckPointIsEqual(data.Source, cp))

	r, err := helpers.BlockRoot(beaconState, data.Target.Epoch)
	if err != nil {
		return false, false, false, err
	}
	t = matchedTarget(bytes.Equal(r, data.Target.Root))

	r, err = helpers.BlockRootAtSlot(beaconState, data.Slot)
	if err != nil {
		return false, false, false, err
	}
	h = matchedHead(bytes.Equal(r, data.BeaconBlockRoot))
	return
}
