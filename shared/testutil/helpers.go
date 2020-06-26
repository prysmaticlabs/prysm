package testutil

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	stateTrie "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/rand"
)

// RandaoReveal returns a signature of the requested epoch using the beacon proposer private key.
func RandaoReveal(beaconState *stateTrie.BeaconState, epoch uint64, privKeys []bls.SecretKey) ([]byte, error) {
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not get beacon proposer index")
	}
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain, err := helpers.Domain(beaconState.Fork(), epoch, params.BeaconConfig().DomainRandao, beaconState.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	root, err := helpers.ComputeSigningRoot(epoch, domain)
	if err != nil {
		return nil, errors.Wrap(err, "could not compute signing root of epoch")
	}
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx].Sign(root[:])
	return epochSignature.Marshal(), nil
}

// BlockSignature calculates the post-state root of the block and returns the signature.
func BlockSignature(
	bState *stateTrie.BeaconState,
	block *ethpb.BeaconBlock,
	privKeys []bls.SecretKey,
) (bls.Signature, error) {
	var err error
	s, err := state.CalculateStateRoot(context.Background(), bState, &ethpb.SignedBeaconBlock{Block: block})
	if err != nil {
		return nil, err
	}
	block.StateRoot = s[:]
	domain, err := helpers.Domain(bState.Fork(), helpers.CurrentEpoch(bState), params.BeaconConfig().DomainBeaconProposer, bState.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	blockRoot, err := helpers.ComputeSigningRoot(block, domain)
	if err != nil {
		return nil, err
	}
	// Temporarily increasing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	currentSlot := bState.Slot()
	if err := bState.SetSlot(block.Slot); err != nil {
		return nil, err
	}
	proposerIdx, err := helpers.BeaconProposerIndex(bState)
	if err != nil {
		return nil, err
	}
	if err := bState.SetSlot(currentSlot); err != nil {
		return nil, err
	}
	return privKeys[proposerIdx].Sign(blockRoot[:]), nil
}

// Random32Bytes generates a random 32 byte slice.
func Random32Bytes(t *testing.T) []byte {
	b := make([]byte, 32)
	_, err := rand.NewDeterministicRandomGenerator().Read(b)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
