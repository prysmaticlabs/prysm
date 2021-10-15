package util

import (
	"context"
	"encoding/binary"
	"testing"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/signing"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/time"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/transition"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/crypto/bls"
	"github.com/prysmaticlabs/prysm/crypto/rand"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1/wrapper"
)

// RandaoReveal returns a signature of the requested epoch using the beacon proposer private key.
func RandaoReveal(beaconState state.ReadOnlyBeaconState, epoch types.Epoch, privKeys []bls.SecretKey) ([]byte, error) {
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := helpers.BeaconProposerIndex(context.Background(), beaconState)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not get beacon proposer index")
	}
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, uint64(epoch))

	// We make the previous validator's index sign the message instead of the proposer.
	sszEpoch := types.SSZUint64(epoch)
	return signing.ComputeDomainAndSign(beaconState, epoch, &sszEpoch, params.BeaconConfig().DomainRandao, privKeys[proposerIdx])
}

// BlockSignature calculates the post-state root of the block and returns the signature.
func BlockSignature(
	bState state.BeaconState,
	block *ethpb.BeaconBlock,
	privKeys []bls.SecretKey,
) (bls.Signature, error) {
	var err error
	s, err := transition.CalculateStateRoot(context.Background(), bState, wrapper.WrappedPhase0SignedBeaconBlock(&ethpb.SignedBeaconBlock{Block: block}))
	if err != nil {
		return nil, err
	}
	block.StateRoot = s[:]
	domain, err := signing.Domain(bState.Fork(), time.CurrentEpoch(bState), params.BeaconConfig().DomainBeaconProposer, bState.GenesisValidatorRoot())
	if err != nil {
		return nil, err
	}
	blockRoot, err := signing.ComputeSigningRoot(block, domain)
	if err != nil {
		return nil, err
	}
	// Temporarily increasing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	currentSlot := bState.Slot()
	if err := bState.SetSlot(block.Slot); err != nil {
		return nil, err
	}
	proposerIdx, err := helpers.BeaconProposerIndex(context.Background(), bState)
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
	_, err := rand.NewDeterministicGenerator().Read(b)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
