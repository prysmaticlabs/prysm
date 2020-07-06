package blocks_test

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestProcessRandao_IncorrectProposerFailsVerification(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)
	// We fetch the proposer's index as that is whom the RANDAO will be verified against.
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Fatal(err)
	}
	epoch := uint64(0)
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain, err := helpers.Domain(beaconState.Fork(), epoch, params.BeaconConfig().DomainRandao, beaconState.GenesisValidatorRoot())
	if err != nil {
		t.Fatal(err)
	}
	root, err := ssz.HashTreeRoot(&pb.SigningData{ObjectRoot: buf, Domain: domain})
	if err != nil {
		t.Fatal(err)
	}
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx-1].Sign(root[:])
	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: epochSignature.Marshal(),
		},
	}

	want := "block randao: signature did not verify"
	if _, err := blocks.ProcessRandao(
		beaconState,
		block.Body,
	); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %v, received %v", want, err)
	}
}

func TestProcessRandao_SignatureVerifiesAndUpdatesLatestStateMixes(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	epoch := helpers.CurrentEpoch(beaconState)
	epochSignature, err := testutil.RandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: epochSignature,
		},
	}

	newState, err := blocks.ProcessRandao(
		beaconState,
		block.Body,
	)
	if err != nil {
		t.Errorf("Unexpected error processing block randao: %v", err)
	}
	currentEpoch := helpers.CurrentEpoch(beaconState)
	mix := newState.RandaoMixes()[currentEpoch%params.BeaconConfig().EpochsPerHistoricalVector]

	if bytes.Equal(mix, params.BeaconConfig().ZeroHash[:]) {
		t.Errorf(
			"Expected empty signature to be overwritten by randao reveal, received %v",
			params.BeaconConfig().EmptySignature,
		)
	}
}

func TestRandaoSignatureSet_OK(t *testing.T) {
	beaconState, privKeys := testutil.DeterministicGenesisState(t, 100)

	epoch := helpers.CurrentEpoch(beaconState)
	epochSignature, err := testutil.RandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}

	block := &ethpb.BeaconBlock{
		Body: &ethpb.BeaconBlockBody{
			RandaoReveal: epochSignature,
		},
	}

	set, _, err := blocks.RandaoSignatureSet(beaconState, block.Body)
	if err != nil {
		t.Fatal(err)
	}
	verified, err := set.Verify()
	if err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Error("Unable to verify randao signature set")
	}
}
