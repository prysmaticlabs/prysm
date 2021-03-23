package simulator

import (
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGenerateAttestationsForSlot_Slashing(t *testing.T) {
	simParams := &Parameters{
		AggregationPercent:     1,
		NumValidators:          64,
		AttesterSlashingProbab: 1,
	}
	epoch3Atts, _ := generateAttestationsForSlot(simParams, params.BeaconConfig().SlotsPerEpoch*3)
	epoch4Atts, _ := generateAttestationsForSlot(simParams, params.BeaconConfig().SlotsPerEpoch*4)
	for i := 0; i < len(epoch3Atts); i += 2 {
		goodAtt := epoch3Atts[i]
		surroundAtt := epoch4Atts[i+1]
		require.Equal(t, true, slashutil.IsSurround(surroundAtt, goodAtt))
	}
}

func TestGenerateAttestationsForSlot_CorrectIndices(t *testing.T) {
	simParams := &Parameters{
		AggregationPercent:     1,
		NumValidators:          16384,
		AttesterSlashingProbab: 0,
	}
	slot0Atts, _ := generateAttestationsForSlot(simParams, 0)
	slot1Atts, _ := generateAttestationsForSlot(simParams, 1)
	slot2Atts, _ := generateAttestationsForSlot(simParams, 2)
	var validatorIndices []uint64
	for _, att := range append(slot0Atts, slot1Atts...) {
		validatorIndices = append(validatorIndices, att.AttestingIndices...)
	}
	for _, att := range slot2Atts {
		validatorIndices = append(validatorIndices, att.AttestingIndices...)
	}

	// Making sure indices are one after the other for attestations.
	var validatorIndex uint64
	for _, ii := range validatorIndices {
		require.Equal(t, validatorIndex, ii)
		validatorIndex++
	}
}
