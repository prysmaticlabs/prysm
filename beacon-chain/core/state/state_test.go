package state_test

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		CacheTreeHash: false,
	})
}

func TestGenesisBeaconState_OK(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	genesisEpochNumber := uint64(0)

	if !bytes.Equal(params.BeaconConfig().GenesisForkVersion, []byte{0, 0, 0, 0}) {
		t.Error("GenesisSlot( should be {0,0,0,0} for these tests to pass")
	}
	genesisForkVersion := params.BeaconConfig().GenesisForkVersion

	if params.BeaconConfig().ZeroHash != [32]byte{} {
		t.Error("ZeroHash should be all 0s for these tests to pass")
	}

	if params.BeaconConfig().LatestRandaoMixesLength != 8192 {
		t.Error("LatestRandaoMixesLength should be 8192 for these tests to pass")
	}
	latestRandaoMixesLength := int(params.BeaconConfig().LatestRandaoMixesLength)

	if params.BeaconConfig().ShardCount != 1024 {
		t.Error("ShardCount should be 1024 for these tests to pass")
	}
	shardCount := int(params.BeaconConfig().ShardCount)

	if params.BeaconConfig().LatestBlockRootsLength != 8192 {
		t.Error("LatestBlockRootsLength should be 8192 for these tests to pass")
	}

	if params.BeaconConfig().DepositsForChainStart != 16384 {
		t.Error("DepositsForChainStart should be 16384 for these tests to pass")
	}
	depositsForChainStart := int(params.BeaconConfig().DepositsForChainStart)

	if params.BeaconConfig().LatestSlashedExitLength != 8192 {
		t.Error("LatestSlashedExitLength should be 8192 for these tests to pass")
	}

	genesisTime := uint64(99999)
	processedPowReceiptRoot := []byte{'A', 'B', 'C'}
	maxDeposit := params.BeaconConfig().MaxDepositAmount
	var deposits []*pb.Deposit
	for i := 0; i < depositsForChainStart; i++ {
		depositData, err := helpers.EncodeDepositData(
			&pb.DepositInput{
				Pubkey:                      []byte(strconv.Itoa(i)),
				ProofOfPossession:           []byte{'B'},
				WithdrawalCredentialsHash32: []byte{'C'},
			},
			maxDeposit,
			time.Now().Unix(),
		)
		if err != nil {
			t.Fatalf("Could not encode deposit data: %v", err)
		}
		deposits = append(deposits, &pb.Deposit{
			Proof:       [][]byte{{1}, {2}, {3}},
			Index:       0,
			DepositData: depositData,
		})
	}

	newState, err := state.GenesisBeaconState(
		deposits,
		genesisTime,
		&pb.Eth1Data{
			DepositRoot: processedPowReceiptRoot,
			BlockRoot:   []byte{},
		})
	if err != nil {
		t.Fatalf("could not execute GenesisBeaconState: %v", err)
	}

	// Misc fields checks.
	if newState.Slot != 0 {
		t.Error("Slot was not correctly initialized")
	}
	if newState.GenesisTime != genesisTime {
		t.Error("GenesisTime was not correctly initialized")
	}
	if !reflect.DeepEqual(*newState.Fork, pb.Fork{
		PreviousVersion: genesisForkVersion,
		CurrentVersion:  genesisForkVersion,
		Epoch:           genesisEpochNumber,
	}) {
		t.Error("Fork was not correctly initialized")
	}

	// Validator registry fields checks.
	if len(newState.ValidatorRegistry) != depositsForChainStart {
		t.Error("ValidatorRegistry was not correctly initialized")
	}
	if len(newState.Balances) != depositsForChainStart {
		t.Error("Balances was not correctly initialized")
	}

	// Randomness and committees fields checks.
	if len(newState.LatestRandaoMixes) != latestRandaoMixesLength {
		t.Error("Length of LatestRandaoMixes was not correctly initialized")
	}

	// Finality fields checks.
	if newState.PreviousJustifiedEpoch != genesisEpochNumber {
		t.Error("PreviousJustifiedEpoch was not correctly initialized")
	}
	if newState.CurrentJustifiedEpoch != genesisEpochNumber {
		t.Error("JustifiedEpoch was not correctly initialized")
	}
	if newState.FinalizedEpoch != genesisEpochNumber {
		t.Error("FinalizedSlot was not correctly initialized")
	}
	if newState.JustificationBitfield != 0 {
		t.Error("JustificationBitfield was not correctly initialized")
	}

	// Recent state checks.
	if len(newState.CurrentCrosslinks) != shardCount {
		t.Error("Length of CurrentCrosslinks was not correctly initialized")
	}
	if len(newState.PreviousCrosslinks) != shardCount {
		t.Error("Length of PreviousCrosslinks was not correctly initialized")
	}
	if !reflect.DeepEqual(newState.LatestSlashedBalances, make([]uint64, params.BeaconConfig().LatestSlashedExitLength)) {
		t.Error("LatestSlashedBalances was not correctly initialized")
	}
	if !reflect.DeepEqual(newState.CurrentEpochAttestations, []*pb.PendingAttestation{}) {
		t.Error("CurrentEpochAttestations was not correctly initialized")
	}
	if !reflect.DeepEqual(newState.PreviousEpochAttestations, []*pb.PendingAttestation{}) {
		t.Error("PreviousEpochAttestations was not correctly initialized")
	}

	activeValidators := helpers.ActiveValidatorIndices(newState, 0)
	indicesBytes := []byte{}
	for _, val := range activeValidators {
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf, val)
		indicesBytes = append(indicesBytes, buf...)
	}
	genesisActiveIndexRoot := hashutil.Hash(indicesBytes)
	if !bytes.Equal(newState.LatestActiveIndexRoots[0], genesisActiveIndexRoot[:]) {
		t.Errorf(
			"Expected index roots to be the tree hash root of active validator indices, received %#x",
			newState.LatestActiveIndexRoots[0],
		)
	}
	if !bytes.Equal(newState.LatestActiveIndexRoots[0], genesisActiveIndexRoot[:]) {
		t.Errorf(
			"Expected index roots to be the tree hash root of active validator indices, received %#x",
			newState.LatestActiveIndexRoots[0],
		)
	}
	// deposit root checks.
	if !bytes.Equal(newState.LatestEth1Data.DepositRoot, processedPowReceiptRoot) {
		t.Error("LatestEth1Data DepositRoot was not correctly initialized")
	}
	if !reflect.DeepEqual(newState.Eth1DataVotes, []*pb.Eth1Data{}) {
		t.Error("Eth1DataVotes was not correctly initialized")
	}
}

func TestGenesisState_HashEquality(t *testing.T) {
	state1, _ := state.GenesisBeaconState(nil, 0, &pb.Eth1Data{})
	state2, _ := state.GenesisBeaconState(nil, 0, &pb.Eth1Data{})

	root1, err1 := hashutil.HashProto(state1)
	root2, err2 := hashutil.HashProto(state2)

	if err1 != nil || err2 != nil {
		t.Fatalf("Failed to marshal state to bytes: %v %v", err1, err2)
	}

	if root1 != root2 {
		t.Fatalf("Tree hash of two genesis states should be equal, received %#x == %#x", root1, root2)
	}
}

func TestGenesisState_InitializesLatestBlockHashes(t *testing.T) {
	s, _ := state.GenesisBeaconState(nil, 0, nil)
	want, got := len(s.LatestBlockRoots), int(params.BeaconConfig().LatestBlockRootsLength)
	if want != got {
		t.Errorf("Wrong number of recent block hashes. Got: %d Want: %d", got, want)
	}

	want = cap(s.LatestBlockRoots)
	if want != got {
		t.Errorf("The slice underlying array capacity is wrong. Got: %d Want: %d", got, want)
	}

	for _, h := range s.LatestBlockRoots {
		if !bytes.Equal(h, params.BeaconConfig().ZeroHash[:]) {
			t.Errorf("Unexpected non-zero hash data: %v", h)
		}
	}
}
