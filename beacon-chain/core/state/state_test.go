package state_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/proto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
)

func TestGenesisBeaconState_OK(t *testing.T) {
	genesisEpochNumber := uint64(0)

	if !bytes.Equal(params.BeaconConfig().GenesisForkVersion, []byte{0, 0, 0, 1}) {
		t.Error("GenesisSlot( should be {0,0,0,0} for these tests to pass")
	}
	genesisForkVersion := params.BeaconConfig().GenesisForkVersion

	if params.BeaconConfig().ZeroHash != [32]byte{} {
		t.Error("ZeroHash should be all 0s for these tests to pass")
	}

	if params.BeaconConfig().EpochsPerHistoricalVector != 65536 {
		t.Error("EpochsPerHistoricalVector should be 8192 for these tests to pass")
	}
	latestRandaoMixesLength := params.BeaconConfig().EpochsPerHistoricalVector

	if params.BeaconConfig().HistoricalRootsLimit != 16777216 {
		t.Error("HistoricalRootsLimit should be 16777216 for these tests to pass")
	}

	depositsForChainStart := 100

	if params.BeaconConfig().EpochsPerSlashingsVector != 8192 {
		t.Error("EpochsPerSlashingsVector should be 8192 for these tests to pass")
	}

	genesisTime := uint64(99999)
	deposits, _, err := testutil.DeterministicDepositsAndKeys(uint64(depositsForChainStart))
	if err != nil {
		t.Fatal(err)
	}
	eth1Data, err := testutil.DeterministicEth1Data(len(deposits))
	if err != nil {
		t.Fatal(err)
	}
	newState, err := state.GenesisBeaconState(deposits, genesisTime, eth1Data)
	if err != nil {
		t.Fatalf("could not execute GenesisBeaconState: %v", err)
	}

	// Misc fields checks.
	if newState.Slot() != 0 {
		t.Error("Slot was not correctly initialized")
	}
	if !proto.Equal(newState.Fork(), &pb.Fork{
		PreviousVersion: genesisForkVersion,
		CurrentVersion:  genesisForkVersion,
		Epoch:           genesisEpochNumber,
	}) {
		t.Error("Fork was not correctly initialized")
	}

	// Validator registry fields checks.
	if len(newState.Validators()) != depositsForChainStart {
		t.Error("Validators was not correctly initialized")
	}
	v, err := newState.ValidatorAtIndex(0)
	if err != nil {
		t.Fatal(err)
	}
	if v.ActivationEpoch != 0 {
		t.Error("Validators was not correctly initialized")
	}
	v, err = newState.ValidatorAtIndex(0)
	if err != nil {
		t.Fatal(err)
	}
	if v.ActivationEligibilityEpoch != 0 {
		t.Error("Validators was not correctly initialized")
	}
	if len(newState.Balances()) != depositsForChainStart {
		t.Error("Balances was not correctly initialized")
	}

	// Randomness and committees fields checks.
	if uint64(len(newState.RandaoMixes())) != latestRandaoMixesLength {
		t.Error("Length of RandaoMixes was not correctly initialized")
	}
	mix, err := newState.RandaoMixAtIndex(0)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(mix, eth1Data.BlockHash) {
		t.Error("RandaoMixes was not correctly initialized")
	}

	// Finality fields checks.
	if newState.PreviousJustifiedCheckpoint().Epoch != genesisEpochNumber {
		t.Error("PreviousJustifiedCheckpoint.Epoch was not correctly initialized")
	}
	if newState.CurrentJustifiedCheckpoint().Epoch != genesisEpochNumber {
		t.Error("JustifiedEpoch was not correctly initialized")
	}
	if newState.FinalizedCheckpointEpoch() != genesisEpochNumber {
		t.Error("FinalizedSlot was not correctly initialized")
	}
	if newState.JustificationBits()[0] != 0x00 {
		t.Error("JustificationBits was not correctly initialized")
	}

	// Recent state checks.
	if !reflect.DeepEqual(newState.Slashings(), make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)) {
		t.Error("Slashings was not correctly initialized")
	}
	if !reflect.DeepEqual(newState.CurrentEpochAttestations(), []*pb.PendingAttestation{}) {
		t.Error("CurrentEpochAttestations was not correctly initialized")
	}
	if !reflect.DeepEqual(newState.PreviousEpochAttestations(), []*pb.PendingAttestation{}) {
		t.Error("PreviousEpochAttestations was not correctly initialized")
	}

	zeroHash := params.BeaconConfig().ZeroHash[:]
	// History root checks.
	if !bytes.Equal(newState.StateRoots()[0], zeroHash) {
		t.Error("StateRoots was not correctly initialized")
	}
	if !bytes.Equal(newState.BlockRoots()[0], zeroHash) {
		t.Error("BlockRoots was not correctly initialized")
	}

	// Deposit root checks.
	if !bytes.Equal(newState.Eth1Data().DepositRoot, eth1Data.DepositRoot) {
		t.Error("Eth1Data DepositRoot was not correctly initialized")
	}
	if !reflect.DeepEqual(newState.Eth1DataVotes(), []*ethpb.Eth1Data{}) {
		t.Error("Eth1DataVotes was not correctly initialized")
	}
}

func TestGenesisState_HashEquality(t *testing.T) {
	deposits, _, err := testutil.DeterministicDepositsAndKeys(100)
	if err != nil {
		t.Fatal(err)
	}
	state1, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Error(err)
	}
	state2, err := state.GenesisBeaconState(deposits, 0, &ethpb.Eth1Data{BlockHash: make([]byte, 32)})
	if err != nil {
		t.Error(err)
	}

	root1, err1 := hashutil.HashProto(state1.CloneInnerState())
	root2, err2 := hashutil.HashProto(state2.CloneInnerState())

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to marshal state to bytes: %v %v", err1, err2)
	}

	if root1 != root2 {
		t.Fatalf("Tree hash of two genesis states should be equal, received %#x == %#x", root1, root2)
	}
}

func TestGenesisState_InitializesLatestBlockHashes(t *testing.T) {
	s, err := state.GenesisBeaconState(nil, 0, &ethpb.Eth1Data{})
	if err != nil {
		t.Error(err)
	}
	got, want := uint64(len(s.BlockRoots())), params.BeaconConfig().SlotsPerHistoricalRoot
	if want != got {
		t.Errorf("Wrong number of recent block hashes. Got: %d Want: %d", got, want)
	}

	got = uint64(cap(s.BlockRoots()))
	if want != got {
		t.Errorf("The slice underlying array capacity is wrong. Got: %d Want: %d", got, want)
	}

	for _, h := range s.BlockRoots() {
		if !bytes.Equal(h, params.BeaconConfig().ZeroHash[:]) {
			t.Errorf("Unexpected non-zero hash data: %v", h)
		}
	}
}

func TestGenesisState_FailsWithoutEth1data(t *testing.T) {
	_, err := state.GenesisBeaconState(nil, 0, nil)
	if err == nil || err.Error() != "no eth1data provided for genesis state" {
		t.Errorf("Did not receive eth1data error with nil eth1data, got %v", err)
	}
}
