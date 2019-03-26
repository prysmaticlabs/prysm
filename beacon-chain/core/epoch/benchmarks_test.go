package epoch_test

import (
	"context"
	"fmt"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"testing"
	"time"

	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var ValidatorCount = 8192
var RunAmount = 67108864 / ValidatorCount

func BenchmarkProcessEth1Data(b *testing.B) {
	deposits := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}
	requiredVoteCount := params.BeaconConfig().EpochsPerEth1VotingPeriod *
		params.BeaconConfig().SlotsPerEpoch
	beaconState.Slot = 15 * params.BeaconConfig().SlotsPerEpoch
	beaconState.LatestEth1Data = &pb.Eth1Data{
		DepositRootHash32: nil,
		BlockHash32:       nil,
	}
	beaconState.Eth1DataVotes = []*pb.Eth1DataVote{
		{
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{'A'},
				BlockHash32:       []byte{'B'},
			},
			VoteCount: 0,
		},
		{
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{'C'},
				BlockHash32:       []byte{'D'},
			},
			VoteCount: requiredVoteCount/2 + 1,
		},
		{
			Eth1Data: &pb.Eth1Data{
				DepositRootHash32: []byte{'E'},
				BlockHash32:       []byte{'F'},
			},
			VoteCount: requiredVoteCount / 2,
		},
	}

	b.ResetTimer()
	b.N = RunAmount
	for i := 0; i < b.N; i++ {
		_ = epoch.ProcessEth1Data(context.Background(), beaconState)
	}
}

func BenchmarkProcessJustification(b *testing.B) {
	deposits := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		b.Errorf("SlotsPerEpoch should be 64 for this benchmark to run")
	}

	beaconState.Slot = 300 + params.BeaconConfig().GenesisSlot
	beaconState.JustifiedEpoch = 3
	beaconState.JustificationBitfield = 4

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = epoch.ProcessJustification(context.Background(), beaconState, 1, 1, 1, 1, false)
	}
}

func BenchmarkProcessCrosslinks(b *testing.B) {
	deposits := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}
	beaconState.Slot = params.BeaconConfig().GenesisSlot + 5*params.BeaconConfig().SlotsPerEpoch

	byteLength := int(uint64(ValidatorCount) / params.BeaconConfig().TargetCommitteeSize / 8 * 2)
	var participationBitfield []byte
	for i := 0; i < byteLength; i++ {
		participationBitfield = append(participationBitfield, byte(0xff))
	}

	var attestations []*pb.PendingAttestation
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                    beaconState.Slot,
				CrosslinkDataRootHash32: []byte{'A'},
			},
			// All validators attested to the above roots.
			AggregationBitfield: participationBitfield,
		}
		attestations = append(attestations, attestation)
	}

	b.N = 20
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := epoch.ProcessCrosslinks(
			context.Background(),
			beaconState,
			attestations,
			nil,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessPenaltiesAndExits(b *testing.B) {
	deposits := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}

	latestSlashedExits := make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
	for i := 0; i < len(latestSlashedExits); i++ {
		latestSlashedExits[i] = uint64(i) * params.BeaconConfig().MaxDepositAmount
	}

	beaconState.Slot = params.BeaconConfig().LatestSlashedExitLength / 2 * params.BeaconConfig().SlotsPerEpoch
	beaconState.LatestSlashedBalances = latestSlashedExits

	b.N = 50
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.ProcessPenaltiesAndExits(context.Background(), beaconState)
	}
}

func BenchmarkProcessEjections(b *testing.B) {
	deposits := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}

	beaconState.Slot = 1
	for index := range beaconState.ValidatorBalances {
		if index%2^5 == 0 {
			beaconState.ValidatorBalances[index] = params.BeaconConfig().EjectionBalance - 1
		}
	}

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = epoch.ProcessEjections(context.Background(), beaconState, false /* disable logging */)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdateRegistry(b *testing.B) {
	deposits := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}

	currentEpoch := uint64(5)
	beaconState.Slot = currentEpoch * params.BeaconConfig().SlotsPerEpoch

	exitEpoch := helpers.EntryExitEffectEpoch(currentEpoch)
	for index := range beaconState.ValidatorRegistry {
		if index%2^6 == 0 {
			beaconState.ValidatorRegistry[index].ExitEpoch = exitEpoch
			beaconState.ValidatorRegistry[index].StatusFlags = pb.Validator_INITIATED_EXIT
		} else if index%2^5 == 0 {
			beaconState.ValidatorRegistry[index].ExitEpoch = params.BeaconConfig().ActivationExitDelay
			beaconState.ValidatorRegistry[index].ActivationEpoch = 5 + params.BeaconConfig().ActivationExitDelay + 1
		}
	}

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := v.UpdateRegistry(context.Background(), beaconState)
		if err != nil {
			b.Fatal(err)
		}
	}
}

//

// state = v.ProcessPenaltiesAndExits(ctx, state)

func BenchmarkUpdateLatestActiveIndexRoots(b *testing.B) {
	deposits := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}

	currentEpoch := uint64(1234)
	latestActiveIndexRoots := make([][]byte,
		params.BeaconConfig().LatestActiveIndexRootsLength)

	beaconState.Slot = currentEpoch * params.BeaconConfig().SlotsPerEpoch
	beaconState.LatestIndexRootHash32S = latestActiveIndexRoots

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := epoch.UpdateLatestActiveIndexRoots(context.Background(), beaconState)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdateLatestSlashedBalances(b *testing.B) {
	deposits := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}

	slashedExitLength := params.BeaconConfig().LatestSlashedExitLength
	currentEpoch := (slashedExitLength + 1) % slashedExitLength
	beaconState.Slot = currentEpoch * params.BeaconConfig().SlotsPerEpoch

	latestSlashedExitBalances := make([]uint64, slashedExitLength)
	latestSlashedExitBalances[currentEpoch] = 234324
	beaconState.LatestSlashedBalances = latestSlashedExitBalances

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = epoch.UpdateLatestSlashedBalances(context.Background(), beaconState)
	}
}

func BenchmarkCleanupAttestations(b *testing.B) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		b.Error("SlotsPerEpoch should be 64 for these tests to pass")
	}

	deposits := setupBenchmarkInitialDeposits(ValidatorCount)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		b.Fatal(err)
	}

	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	beaconState.Slot = slotsPerEpoch
	beaconState.LatestAttestations = []*pb.PendingAttestation{
		{Data: &pb.AttestationData{Slot: 1}},
		{Data: &pb.AttestationData{Slot: slotsPerEpoch - 10}},
		{Data: &pb.AttestationData{Slot: slotsPerEpoch}},
		{Data: &pb.AttestationData{Slot: slotsPerEpoch + 1}},
		{Data: &pb.AttestationData{Slot: slotsPerEpoch + 20}},
		{Data: &pb.AttestationData{Slot: 32}},
		{Data: &pb.AttestationData{Slot: 33}},
		{Data: &pb.AttestationData{Slot: 2 * slotsPerEpoch}},
	}

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = epoch.CleanupAttestations(context.Background(), beaconState)
	}
}

func setupBenchmarkInitialDeposits(numDeposits int) []*pb.Deposit {
	setBenchmarkConfig()
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		depositInput := &pb.DepositInput{
			Pubkey:                      []byte(fmt.Sprintf("%d", i)),
			WithdrawalCredentialsHash32: []byte{1, 2, 3},
		}
		balance := params.BeaconConfig().MaxDepositAmount
		depositData, err := helpers.EncodeDepositData(depositInput, balance, time.Now().Unix())
		if err != nil {
			panic(err)
		}
		deposits[i] = &pb.Deposit{
			DepositData: depositData,
		}
	}
	return deposits
}

func setBenchmarkConfig() {
	c := params.BeaconConfig()
	// From Danny Ryan's "Minimal Config"
	// c.SlotsPerEpoch = 8
	// c.MinAttestationInclusionDelay = 2
	// c.TargetCommitteeSize = 4
	// c.GenesisEpoch = c.GenesisSlot / 8
	// c.LatestRandaoMixesLength = 64
	// c.LatestActiveIndexRootsLength = 64
	// c.LatestSlashedExitLength = 64
	params.OverrideBeaconConfig(c)
}
