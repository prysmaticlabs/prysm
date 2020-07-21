package testutil

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil/assert"
	"github.com/prysmaticlabs/prysm/shared/testutil/require"
)

func TestBlockSignature(t *testing.T) {
	beaconState, privKeys := DeterministicGenesisState(t, 100)
	block, err := GenerateFullBlock(beaconState, privKeys, nil, 0)
	require.NoError(t, err)

	require.NoError(t, beaconState.SetSlot(beaconState.Slot()+1))
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	assert.NoError(t, err)

	if err := beaconState.SetSlot(beaconState.Slot() - 1); err != nil {
		t.Error(err)
	}
	epoch := helpers.SlotToEpoch(block.Block.Slot)
	domain, err := helpers.Domain(beaconState.Fork(), epoch, params.BeaconConfig().DomainBeaconProposer, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	signingRoot, err := helpers.ComputeSigningRoot(block.Block, domain)
	assert.NoError(t, err)

	blockSig := privKeys[proposerIdx].Sign(signingRoot[:]).Marshal()

	signature, err := BlockSignature(beaconState, block.Block, privKeys)
	assert.NoError(t, err)

	if !bytes.Equal(blockSig[:], signature.Marshal()) {
		t.Errorf("Expected block signatures to be equal, received %#x != %#x", blockSig[:], signature.Marshal())
	}
}

func TestRandaoReveal(t *testing.T) {
	beaconState, privKeys := DeterministicGenesisState(t, 100)

	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := RandaoReveal(beaconState, epoch, privKeys)
	assert.NoError(t, err)

	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	assert.NoError(t, err)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain, err := helpers.Domain(beaconState.Fork(), epoch, params.BeaconConfig().DomainRandao, beaconState.GenesisValidatorRoot())
	require.NoError(t, err)
	root, err := helpers.ComputeSigningRoot(epoch, domain)
	require.NoError(t, err)
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx].Sign(root[:]).Marshal()

	if !bytes.Equal(randaoReveal[:], epochSignature[:]) {
		t.Errorf("Expected randao reveals to be equal, received %#x != %#x", randaoReveal[:], epochSignature[:])
	}
}
