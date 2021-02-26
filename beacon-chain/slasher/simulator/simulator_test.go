package simulator

import (
	"bytes"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/slashutil"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestGenerateAttestationsForSlot_Slashing(t *testing.T) {
	simParams := &Parameters{
		NumValidators:          64,
		AttesterSlashingProbab: 1,
	}
	epoch3Atts := generateAttestationsForSlot(simParams, params.BeaconConfig().SlotsPerEpoch*3)
	epoch4Atts := generateAttestationsForSlot(simParams, params.BeaconConfig().SlotsPerEpoch*4)
	for i := 0; i < len(epoch3Atts); i += 2 {
		goodAtt := epoch3Atts[i]
		surroundAtt := epoch4Atts[i+1]
		require.Equal(t, true, slashutil.IsSurround(surroundAtt, goodAtt))
	}
}

func TestGenerateAttestationsForSlot_CorrectIndices(t *testing.T) {
	simParams := &Parameters{
		NumValidators:          16384,
		AttesterSlashingProbab: 0,
	}
	slot0Atts := generateAttestationsForSlot(simParams, 0)
	slot1Atts := generateAttestationsForSlot(simParams, 1)
	slot2Atts := generateAttestationsForSlot(simParams, 2)
	var indices []uint64
	for _, att := range append(slot0Atts, slot1Atts...) {
		indices = append(indices, att.AttestingIndices...)
	}
	for _, att := range slot2Atts {
		indices = append(indices, att.AttestingIndices...)
	}

	// Making sure indices are one after the other for attestations.
	var indice uint64
	for _, ii := range indices {
		require.Equal(t, indice, ii)
		indice++
	}
}

func TestGenerateBlockHeadersForSlot_Slashing(t *testing.T) {
	simParams := &Parameters{
		NumValidators:          64,
		ProposerSlashingProbab: 1,
	}
	slot1Blocks := generateBlockHeadersForSlot(simParams, 1)
	require.Equal(t, 2, len(slot1Blocks))

	block1Root, err := slot1Blocks[0].HashTreeRoot()
	require.NoError(t, err)
	block2Root, err := slot1Blocks[1].HashTreeRoot()
	require.NoError(t, err)
	if slot1Blocks[0].ProposerIndex == slot1Blocks[1].ProposerIndex && bytes.Equal(block1Root[:], block2Root[:]) {
		t.Error("Blocks received were not slashable")
	}
}
