package epoch_test

import (
	"context"
	"fmt"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	bal "github.com/prysmaticlabs/prysm/beacon-chain/core/balances"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var ValidatorCount = 300000
var RunAmount = 134217728 / ValidatorCount

// var conditions = "MAX"

var genesisState = createGenesisState(ValidatorCount)

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
	// if conditions == "MAX" {
	// 	c.MaxAttestations = 128
	// 	c.MaxDeposits = 16
	// 	c.MaxVoluntaryExits = 16
	// } else if conditions == "MIN" {
	// 	c.MaxAttestations = 16
	// 	c.MaxDeposits = 2
	// 	c.MaxVoluntaryExits = 2
	// }
	params.OverrideBeaconConfig(c)
}

func BenchmarkProcessEth1Data(b *testing.B) {
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

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
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

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
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

	beaconState.Slot = params.BeaconConfig().GenesisSlot + 5*params.BeaconConfig().SlotsPerEpoch

	// 4 Mil 31230
	byteLength := mathutil.CeilDiv8(31230)

	var attestations []*pb.PendingAttestation
	for i := 0; i < 10; i++ {
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                    beaconState.Slot,
				CrosslinkDataRootHash32: []byte{'A'},
			},
			// All validators attested to the above roots.
			AggregationBitfield: bitutil.FillBitfield(byteLength),
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

func BenchmarkProcessRewards(b *testing.B) {
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

	var attestations []*pb.PendingAttestation
	for i := uint64(0); i < params.BeaconConfig().MaxAttestations; i++ {
		attestations = append(attestations, &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                     i + params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot,
				Shard:                    1,
				JustifiedEpoch:           params.BeaconConfig().GenesisEpoch + 1,
				JustifiedBlockRootHash32: []byte{0},
			},
			InclusionSlot: i + params.BeaconConfig().SlotsPerEpoch + 1 + params.BeaconConfig().GenesisSlot,
		})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	beaconState.Slot = params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot + 1
	beaconState.LatestAttestations = attestations
	beaconState.LatestRandaoMixes = randaoHashes

	currentEpoch := helpers.CurrentEpoch(beaconState)

	activeValidatorIndices := helpers.ActiveValidatorIndices(beaconState.ValidatorRegistry, currentEpoch)
	totalBalance := e.TotalBalance(context.Background(), beaconState, activeValidatorIndices)

	prevEpochAttestations := e.PrevAttestations(context.Background(), beaconState)
	prevEpochAttesterIndices, err := v.ValidatorIndices(context.Background(), beaconState, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochAttestingBalance := e.TotalBalance(context.Background(), beaconState, prevEpochAttesterIndices)

	prevEpochBoundaryAttestations, err := e.PrevEpochBoundaryAttestations(context.Background(), beaconState, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}

	prevEpochBoundaryAttesterIndices, err := v.ValidatorIndices(context.Background(), beaconState, prevEpochBoundaryAttestations)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochBoundaryAttestingBalances := e.TotalBalance(context.Background(), beaconState, prevEpochBoundaryAttesterIndices)

	prevEpochHeadAttestations, err := e.PrevHeadAttestations(context.Background(), beaconState, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochHeadAttesterIndices, err := v.ValidatorIndices(context.Background(), beaconState, prevEpochHeadAttestations)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochHeadAttestingBalances := e.TotalBalance(context.Background(), beaconState, prevEpochHeadAttesterIndices)

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bal.ExpectedFFGSource(
			context.Background(),
			beaconState,
			prevEpochAttesterIndices,
			prevEpochAttestingBalance,
			totalBalance)

		_ = bal.ExpectedFFGTarget(
			context.Background(),
			beaconState,
			prevEpochBoundaryAttesterIndices,
			prevEpochBoundaryAttestingBalances,
			totalBalance)

		_ = bal.ExpectedBeaconChainHead(
			context.Background(),
			beaconState,
			prevEpochHeadAttesterIndices,
			prevEpochHeadAttestingBalances,
			totalBalance)

		_, err = bal.InclusionDistance(
			context.Background(),
			beaconState,
			prevEpochAttesterIndices,
			totalBalance)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessLeak(b *testing.B) {
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

	var attestations []*pb.PendingAttestation
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch*2; i++ {
		attestations = append(attestations, &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                     i + params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot,
				Shard:                    1,
				JustifiedEpoch:           params.BeaconConfig().GenesisEpoch + 1,
				JustifiedBlockRootHash32: []byte{0},
			},
			InclusionSlot: i + params.BeaconConfig().SlotsPerEpoch + 1 + params.BeaconConfig().GenesisSlot,
		})
	}

	var randaoHashes [][]byte
	for i := uint64(0); i < params.BeaconConfig().SlotsPerEpoch; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}

	beaconState.Slot = params.BeaconConfig().SlotsPerEpoch + params.BeaconConfig().GenesisSlot + 1
	beaconState.LatestAttestations = attestations
	beaconState.LatestRandaoMixes = randaoHashes

	currentEpoch := helpers.CurrentEpoch(beaconState)

	activeValidatorIndices := helpers.ActiveValidatorIndices(beaconState.ValidatorRegistry, currentEpoch)
	totalBalance := e.TotalBalance(context.Background(), beaconState, activeValidatorIndices)

	prevEpochAttestations := e.PrevAttestations(context.Background(), beaconState)
	prevEpochAttesterIndices, err := v.ValidatorIndices(context.Background(), beaconState, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochBoundaryAttestations, err := e.PrevEpochBoundaryAttestations(context.Background(), beaconState, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}

	prevEpochBoundaryAttesterIndices, err := v.ValidatorIndices(context.Background(), beaconState, prevEpochBoundaryAttestations)
	if err != nil {
		b.Fatal(err)
	}

	prevEpochHeadAttestations, err := e.PrevHeadAttestations(context.Background(), beaconState, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochHeadAttesterIndices, err := v.ValidatorIndices(context.Background(), beaconState, prevEpochHeadAttestations)
	if err != nil {
		b.Fatal(err)
	}

	var epochsSinceFinality uint64 = 4
	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bal.InactivityFFGSource(
			context.Background(),
			beaconState,
			prevEpochAttesterIndices,
			totalBalance,
			epochsSinceFinality)

		_ = bal.InactivityFFGTarget(
			context.Background(),
			beaconState,
			prevEpochBoundaryAttesterIndices,
			totalBalance,
			epochsSinceFinality)

		_ = bal.InactivityChainHead(
			context.Background(),
			beaconState,
			prevEpochHeadAttesterIndices,
			totalBalance)

		_ = bal.InactivityExitedPenalties(
			context.Background(),
			beaconState,
			totalBalance,
			epochsSinceFinality)

		_, err = bal.InactivityInclusionDistance(
			context.Background(),
			beaconState,
			prevEpochAttesterIndices,
			totalBalance)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessPenaltiesAndExits(b *testing.B) {
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

	latestSlashedExits := make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
	for i := 0; i < len(latestSlashedExits); i++ {
		latestSlashedExits[i] = uint64(i) * params.BeaconConfig().MaxDepositAmount
	}

	beaconState.Slot = params.BeaconConfig().LatestSlashedExitLength / 2 * params.BeaconConfig().SlotsPerEpoch
	beaconState.LatestSlashedBalances = latestSlashedExits

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = v.ProcessPenaltiesAndExits(context.Background(), beaconState)
	}
}

func BenchmarkProcessEjections(b *testing.B) {
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

	beaconState.Slot = 1
	for index := range beaconState.ValidatorBalances {
		if index%2^5 == 0 {
			beaconState.ValidatorBalances[index] = params.BeaconConfig().EjectionBalance - 1
		}
	}

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := epoch.ProcessEjections(context.Background(), beaconState, false /* disable logging */)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCleanupAttestations(b *testing.B) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		b.Error("SlotsPerEpoch should be 64 for these tests to pass")
	}

	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

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

func BenchmarkUpdateRegistry(b *testing.B) {
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

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

func BenchmarkUpdateLatestActiveIndexRoots(b *testing.B) {
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

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
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

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

func BenchmarkActiveValidatorIndices(b *testing.B) {
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

	currentEpoch := uint64(5)
	beaconState.Slot = currentEpoch * params.BeaconConfig().SlotsPerEpoch

	for index := range beaconState.ValidatorRegistry {
		if index%2^6 == 0 {
			beaconState.ValidatorRegistry[index].ExitEpoch = 4
			beaconState.ValidatorRegistry[index].StatusFlags = pb.Validator_INITIATED_EXIT
		} else if index%2^5 == 0 {
			beaconState.ValidatorRegistry[index].ExitEpoch = params.BeaconConfig().ActivationExitDelay
			beaconState.ValidatorRegistry[index].ActivationEpoch = 5 + params.BeaconConfig().ActivationExitDelay + 1
		}
	}

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = helpers.ActiveValidatorIndices(beaconState.ValidatorRegistry, 5)
	}
}

func BenchmarkValidatorIndexMap(b *testing.B) {
	beaconState := proto.Clone(genesisState).(*pb.BeaconState)

	currentEpoch := uint64(5)
	beaconState.Slot = currentEpoch * params.BeaconConfig().SlotsPerEpoch

	b.N = RunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stateutils.ValidatorIndexMap(beaconState)
	}
}

func createGenesisState(numDeposits int) *pb.BeaconState {
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
	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), &pb.Eth1Data{})
	if err != nil {
		panic(err)
	}

	return genesisState
}
