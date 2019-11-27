package testutil

import (
	"context"
	"encoding/binary"
	"math/rand"
	"testing"

	"github.com/pkg/errors"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// RandaoReveal returns a signature of the requested epoch using the beacon proposer private key.
func RandaoReveal(beaconState *pb.BeaconState, epoch uint64, privKeys []*bls.SecretKey) ([]byte, error) {
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		return []byte{}, errors.Wrap(err, "could not get beacon proposer index")
	}
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := helpers.Domain(beaconState.Fork, epoch, params.BeaconConfig().DomainRandao)
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx].Sign(buf, domain)
	return epochSignature.Marshal(), nil
}

// BlockSignature calculates the post-state root of the block and returns the signature.
func BlockSignature(
	bState *pb.BeaconState,
	block *ethpb.BeaconBlock,
	privKeys []*bls.SecretKey,
) (*bls.Signature, error) {
	s, err := state.CalculateStateRoot(context.Background(), bState, block)
	if err != nil {
		return nil, err
	}
	block.StateRoot = s[:]

	blockRoot, err := ssz.SigningRoot(block)
	if err != nil {
		return nil, err
	}
	// Temporarily increasing the beacon state slot here since BeaconProposerIndex is a
	// function deterministic on beacon state slot.
	currentSlot := bState.Slot
	bState.Slot = block.Slot
	proposerIdx, err := helpers.BeaconProposerIndex(bState)
	if err != nil {
		return nil, err
	}
	domain := helpers.Domain(bState.Fork, helpers.CurrentEpoch(bState), params.BeaconConfig().DomainBeaconProposer)
	bState.Slot = currentSlot
	return privKeys[proposerIdx].Sign(blockRoot[:], domain), nil
}

// Random32Bytes generates a random 32 byte slice.
func Random32Bytes(t *testing.T) []byte {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
