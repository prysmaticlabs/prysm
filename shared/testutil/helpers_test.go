package testutil

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestSignBlock(t *testing.T) {
	deposits, privKeys := SetupInitialDeposits(t, 100)
	validators := make([]*ethpb.Validator, len(deposits))
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey: privKeys[i].PublicKey().Marshal()[:],
		}
	}

	beaconState := &pb.BeaconState{
		Slot: 0,
		Fork: &pb.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	block := &ethpb.BeaconBlock{
		Slot:       0,
		ParentRoot: []byte{0xC0},
	}
	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Error(err)
	}
	signingRoot, err := ssz.SigningRoot(block)
	if err != nil {
		t.Error(err)
	}
	epoch := helpers.SlotToEpoch(block.Slot)
	domain := helpers.Domain(beaconState, epoch, params.BeaconConfig().DomainBeaconProposer)
	blockSig := privKeys[proposerIdx].Sign(signingRoot[:], domain).Marshal()

	signedBlock, err := SignBlock(beaconState, block, privKeys)
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(blockSig[:], signedBlock.Signature) {
		t.Errorf("Expected block signatures to be equal, received %#x != %#x", blockSig[:], signedBlock.Signature)
	}
}

func TestCreateRandaoReveal(t *testing.T) {
	deposits, privKeys := SetupInitialDeposits(t, 100)
	validators := make([]*ethpb.Validator, len(deposits))
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
			PublicKey: privKeys[i].PublicKey().Marshal()[:],
		}
	}

	beaconState := &pb.BeaconState{
		Slot: 0,
		Fork: &pb.Fork{
			CurrentVersion:  params.BeaconConfig().GenesisForkVersion,
			PreviousVersion: params.BeaconConfig().GenesisForkVersion,
		},
		Validators:       validators,
		RandaoMixes:      make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
	}

	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := CreateRandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Error(err)
	}

	proposerIdx, err := helpers.BeaconProposerIndex(beaconState)
	if err != nil {
		t.Error(err)
	}
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, epoch)
	domain := helpers.Domain(beaconState, epoch, params.BeaconConfig().DomainRandao)
	// We make the previous validator's index sign the message instead of the proposer.
	epochSignature := privKeys[proposerIdx].Sign(buf, domain).Marshal()

	if !bytes.Equal(randaoReveal[:], epochSignature[:]) {
		t.Errorf("Expected randao reveals to be equal, received %#x != %#x", randaoReveal[:], epochSignature[:])
	}
}
