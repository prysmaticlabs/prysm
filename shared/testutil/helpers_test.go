package testutil

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestBlockSignature_CorrectSignature(t *testing.T) {
	helpers.ClearAllCaches()

	beaconState, privKeys, err := DeterministicGenesisState(100)
	if err != nil {
		t.Fatal(err)
	}
	block, err := GenerateFullBlock(beaconState, privKeys, nil, 0)
	if err != nil {
		t.Fatal(err)
	}

	beaconState.Slot++
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Error(err)
	}
	beaconState.Slot--
	signingRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Error(err)
	}
	epoch := helpers.SlotToEpoch(block.Slot)
	domain := helpers.Domain(beaconState.Fork, epoch, params.BeaconConfig().DomainBeaconProposer)
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:], domain).Marshal()

	signature, err := BlockSignature(beaconState, block, privKeys)
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(blockSig[:], signature.Marshal()) {
		t.Errorf("Expected block signatures to be equal, received %#x != %#x", blockSig[:], signature.Marshal())
	}
}

func TestRandaoReveal_CorrectReveal(t *testing.T) {
	beaconState, privKeys, err := DeterministicGenesisState(100)
	if err != nil {
		t.Fatal(err)
	}

	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := RandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Error(err)
	}

	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Error(err)
	}
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := helpers.Domain(beaconState.Fork, epoch, params.BeaconConfig().DomainRandao)
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx].Sign(buf, domain).Marshal()

	if !bytes.Equal(randaoReveal[:], epochSignature[:]) {
		t.Errorf("Expected randao reveals to be equal, received %#x != %#x", randaoReveal[:], epochSignature[:])
	}
}
