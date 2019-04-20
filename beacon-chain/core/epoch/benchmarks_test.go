package epoch_test

import (
	"context"
	"fmt"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"testing"
	"time"

	bal "github.com/prysmaticlabs/prysm/beacon-chain/core/balances"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/mathutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var RunAmount = 1000
var SmallRunAmount = 50

// var conditions = "MAX"

var beaconState16K = createFullState(16000)
var beaconState300K = createFullState(300000)

var cfg = &state.TransitionConfig{
	Logging:          false,
	VerifySignatures: false,
}

func setBenchmarkConfig() {
	c := params.DemoBeaconConfig()
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
	c.MaxAttestations = 2
	// 	c.MaxDeposits = 2
	// 	c.MaxVoluntaryExits = 2
	// }
	params.OverrideBeaconConfig(c)

	featureCfg := &featureconfig.FeatureFlagConfig{
		EnableCrosslinks: true,
	}
	featureconfig.InitFeatureConfig(featureCfg)
}

func BenchmarkProcessEth1Data(b *testing.B) {
	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = epoch.ProcessEth1Data(beaconState16K)
		}
	})

	b.Run("300K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = epoch.ProcessEth1Data(beaconState300K)
		}
	})
}

func BenchmarkProcessJustification(b *testing.B) {
	currentEpoch := helpers.CurrentEpoch(beaconState16K)
	prevEpoch := helpers.PrevEpoch(beaconState16K)

	activeValidatorIndices := helpers.ActiveValidatorIndices(beaconState16K.ValidatorRegistry, currentEpoch)
	totalBalance := e.TotalBalance(beaconState16K, activeValidatorIndices)

	currentEpochAttestations := e.CurrentAttestations(beaconState16K)
	currentEpochBoundaryAttestations, err := e.CurrentEpochBoundaryAttestations(beaconState16K, currentEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}

	currentBoundaryAttesterIndices, err := v.ValidatorIndices(beaconState16K, currentEpochBoundaryAttestations)
	if err != nil {
		b.Fatal(err)
	}
	currentBoundaryAttestingBalances := e.TotalBalance(beaconState16K, currentBoundaryAttesterIndices)

	previousActiveValidatorIndices := helpers.ActiveValidatorIndices(beaconState16K.ValidatorRegistry, prevEpoch)
	prevTotalBalance := e.TotalBalance(beaconState16K, previousActiveValidatorIndices)

	prevEpochAttestations := e.PrevAttestations(beaconState16K)
	prevEpochAttesterIndices, err := v.ValidatorIndices(beaconState16K, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochAttestingBalance := e.TotalBalance(beaconState16K, prevEpochAttesterIndices)

	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := e.ProcessJustificationAndFinalization(
				beaconState16K,
				currentBoundaryAttestingBalances,
				prevEpochAttestingBalance,
				prevTotalBalance,
				totalBalance,
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkProcessCrosslinks(b *testing.B) {
	currentEpochAttestations := e.CurrentAttestations(beaconState16K)
	prevEpochAttestations := e.PrevAttestations(beaconState16K)

	b.Run("16K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := epoch.ProcessCrosslinks(beaconState16K, currentEpochAttestations, prevEpochAttestations)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	currentEpochAttestations = e.CurrentAttestations(beaconState300K)
	prevEpochAttestations = e.PrevAttestations(beaconState300K)

	b.Run("300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := epoch.ProcessCrosslinks(beaconState300K, currentEpochAttestations, prevEpochAttestations)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkProcessRewards(b *testing.B) {
	currentEpoch := helpers.CurrentEpoch(beaconState16K)

	activeValidatorIndices := helpers.ActiveValidatorIndices(beaconState16K.ValidatorRegistry, currentEpoch)
	totalBalance := e.TotalBalance(beaconState16K, activeValidatorIndices)

	prevEpochAttestations := e.PrevAttestations(beaconState16K)
	prevEpochAttesterIndices, err := v.ValidatorIndices(beaconState16K, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochAttestingBalance := e.TotalBalance(beaconState16K, prevEpochAttesterIndices)

	prevEpochBoundaryAttestations, err := e.PrevEpochBoundaryAttestations(beaconState16K, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}

	prevEpochBoundaryAttesterIndices, err := v.ValidatorIndices(beaconState16K, prevEpochBoundaryAttestations)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochBoundaryAttestingBalances := e.TotalBalance(beaconState16K, prevEpochBoundaryAttesterIndices)

	prevEpochHeadAttestations, err := e.PrevHeadAttestations(beaconState16K, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochHeadAttesterIndices, err := v.ValidatorIndices(beaconState16K, prevEpochHeadAttestations)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochHeadAttestingBalances := e.TotalBalance(beaconState16K, prevEpochHeadAttesterIndices)

	b.N = SmallRunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bal.ExpectedFFGSource(
			beaconState16K,
			prevEpochAttesterIndices,
			prevEpochAttestingBalance,
			totalBalance)

		_ = bal.ExpectedFFGTarget(
			beaconState16K,
			prevEpochBoundaryAttesterIndices,
			prevEpochBoundaryAttestingBalances,
			totalBalance)

		_ = bal.ExpectedBeaconChainHead(
			beaconState16K,
			prevEpochHeadAttesterIndices,
			prevEpochHeadAttestingBalances,
			totalBalance)

		_, err = bal.InclusionDistance(
			beaconState16K,
			prevEpochAttesterIndices,
			totalBalance)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessLeak(b *testing.B) {
	currentEpoch := helpers.CurrentEpoch(beaconState16K)

	activeValidatorIndices := helpers.ActiveValidatorIndices(beaconState16K.ValidatorRegistry, currentEpoch)
	totalBalance := e.TotalBalance(beaconState16K, activeValidatorIndices)

	prevEpochAttestations := e.PrevAttestations(beaconState16K)
	prevEpochAttesterIndices, err := v.ValidatorIndices(beaconState16K, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochBoundaryAttestations, err := e.PrevEpochBoundaryAttestations(beaconState16K, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}

	prevEpochBoundaryAttesterIndices, err := v.ValidatorIndices(beaconState16K, prevEpochBoundaryAttestations)
	if err != nil {
		b.Fatal(err)
	}

	prevEpochHeadAttestations, err := e.PrevHeadAttestations(beaconState16K, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochHeadAttesterIndices, err := v.ValidatorIndices(beaconState16K, prevEpochHeadAttestations)
	if err != nil {
		b.Fatal(err)
	}

	var epochsSinceFinality uint64 = 4
	b.N = SmallRunAmount
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bal.InactivityFFGSource(
			beaconState16K,
			prevEpochAttesterIndices,
			totalBalance,
			epochsSinceFinality)

		_ = bal.InactivityFFGTarget(
			beaconState16K,
			prevEpochBoundaryAttesterIndices,
			totalBalance,
			epochsSinceFinality)

		_ = bal.InactivityChainHead(
			beaconState16K,
			prevEpochHeadAttesterIndices,
			totalBalance)

		_ = bal.InactivityExitedPenalties(
			beaconState16K,
			totalBalance,
			epochsSinceFinality)

		_, err = bal.InactivityInclusionDistance(
			beaconState16K,
			prevEpochAttesterIndices,
			totalBalance)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessPenaltiesAndExits(b *testing.B) {
	b.Run("16K", func(b *testing.B) {
		b.N = SmallRunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = v.ProcessPenaltiesAndExits(beaconState16K)
		}
	})

	b.Run("300K", func(b *testing.B) {
		b.N = SmallRunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = v.ProcessPenaltiesAndExits(beaconState300K)
		}
	})
}

func BenchmarkProcessEjections(b *testing.B) {
	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := epoch.ProcessEjections(beaconState16K, false /* disable logging */)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("300K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := epoch.ProcessEjections(beaconState300K, false /* disable logging */)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkAttestationInclusion(b *testing.B) {
	currentEpoch := helpers.CurrentEpoch(beaconState16K)
	activeValidatorIndices := helpers.ActiveValidatorIndices(beaconState16K.ValidatorRegistry, currentEpoch)

	totalBalance := e.TotalBalance(beaconState16K, activeValidatorIndices)
	prevEpochAttestations := e.PrevAttestations(beaconState16K)

	prevEpochAttesterIndices, err := v.ValidatorIndices(beaconState16K, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := bal.AttestationInclusion(beaconState16K, totalBalance, prevEpochAttesterIndices)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	currentEpoch = helpers.CurrentEpoch(beaconState300K)
	activeValidatorIndices = helpers.ActiveValidatorIndices(beaconState300K.ValidatorRegistry, currentEpoch)

	totalBalance = e.TotalBalance(beaconState300K, activeValidatorIndices)
	prevEpochAttestations = e.PrevAttestations(beaconState300K)

	prevEpochAttesterIndices, err = v.ValidatorIndices(beaconState300K, prevEpochAttestations)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("300K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := bal.AttestationInclusion(beaconState300K, totalBalance, prevEpochAttesterIndices)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkCleanupAttestations(b *testing.B) {
	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = epoch.CleanupAttestations(beaconState16K)
		}
	})

	b.Run("300K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = epoch.CleanupAttestations(beaconState300K)
		}
	})
}

func BenchmarkUpdateRegistry(b *testing.B) {
	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := v.UpdateRegistry(beaconState16K)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("300K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := v.UpdateRegistry(beaconState300K)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkUpdateLatestActiveIndexRoots(b *testing.B) {
	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := epoch.UpdateLatestActiveIndexRoots(beaconState16K)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("300K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := epoch.UpdateLatestActiveIndexRoots(beaconState300K)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkUpdateLatestSlashedBalances(b *testing.B) {
	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = epoch.UpdateLatestSlashedBalances(beaconState16K)
		}
	})

	b.Run("300K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = epoch.UpdateLatestSlashedBalances(beaconState300K)
		}
	})
}

func BenchmarkProcessEpoch(b *testing.B) {
	b.Run("16K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := state.ProcessEpoch(context.Background(), beaconState16K, cfg)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("300K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := state.ProcessEpoch(context.Background(), beaconState300K, cfg)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

}

func BenchmarkActiveValidatorIndices(b *testing.B) {
	currentEpoch := helpers.CurrentEpoch(beaconState16K)

	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = helpers.ActiveValidatorIndices(beaconState16K.ValidatorRegistry, currentEpoch)
		}
	})

	b.Run("300K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = helpers.ActiveValidatorIndices(beaconState300K.ValidatorRegistry, currentEpoch)
		}
	})
}

func BenchmarkValidatorIndexMap(b *testing.B) {
	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = stateutils.ValidatorIndexMap(beaconState16K)
		}
	})

	b.Run("300K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = stateutils.ValidatorIndexMap(beaconState300K)
		}
	})
}

func createFullState(validatorCount uint64) *pb.BeaconState {
	bState := createGenesisState(validatorCount)

	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	requiredVoteCount := params.BeaconConfig().EpochsPerEth1VotingPeriod * slotsPerEpoch
	currentSlot := params.BeaconConfig().GenesisSlot +
		(params.BeaconConfig().EpochsPerEth1VotingPeriod*2)*slotsPerEpoch - 1

	bState.Slot = currentSlot
	bState.JustifiedEpoch = helpers.SlotToEpoch(currentSlot) - 1
	bState.JustificationBitfield = 4

	prevEpoch := helpers.PrevEpoch(bState)
	currentEpoch := helpers.CurrentEpoch(bState)

	committeeSize := validatorCount / helpers.EpochCommitteeCount(validatorCount)
	byteLength := mathutil.CeilDiv8(int(committeeSize))

	attestationsPerEpoch := slotsPerEpoch * params.BeaconConfig().MaxAttestations
	var attestations []*pb.PendingAttestation

	// Previous epoch attestations
	for i := uint64(0); i < attestationsPerEpoch; i++ {
		attestationSlot := (prevEpoch * slotsPerEpoch) + (i % slotsPerEpoch)
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                     attestationSlot,
				Shard:                    0,
				JustifiedEpoch:           prevEpoch - 1,
				CrosslinkDataRootHash32:  []byte{'A'},
				JustifiedBlockRootHash32: []byte{0},
			},
			InclusionSlot:       attestationSlot + 1,
			AggregationBitfield: bitutil.SetBitfield(int(i)%byteLength, byteLength),
		}
		attestations = append(attestations, attestation)
	}

	fmt.Println("prev attestations created")

	// Current epoch attestations
	for i := uint64(0); i < attestationsPerEpoch; i++ {
		attestationSlot := (currentEpoch * slotsPerEpoch) + (i % slotsPerEpoch)
		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Slot:                     attestationSlot,
				Shard:                    0,
				JustifiedEpoch:           currentEpoch - 1,
				CrosslinkDataRootHash32:  []byte{'A'},
				JustifiedBlockRootHash32: []byte{0},
			},
			InclusionSlot:       attestationSlot + 1,
			AggregationBitfield: bitutil.SetBitfield(int(i)%byteLength, byteLength),
		}
		attestations = append(attestations, attestation)
	}
	fmt.Println("cur attestations created")
	bState.LatestAttestations = attestations

	// Eth1DataVotes
	bState.Eth1DataVotes = []*pb.Eth1DataVote{
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

	// RANDAO
	var randaoHashes [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestRandaoMixesLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}
	bState.LatestRandaoMixes = randaoHashes

	// LatestSlashedBalances
	latestSlashedBalances := make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
	for i := 0; i < len(latestSlashedBalances); i++ {
		latestSlashedBalances[i] = uint64(i) * params.BeaconConfig().MaxDepositAmount
	}
	bState.LatestSlashedBalances = latestSlashedBalances

	// Ejections
	ejectionCount := uint64(30)
	for index := range bState.ValidatorBalances {
		if uint64(index)%(validatorCount/ejectionCount)-1 == 0 {
			bState.ValidatorBalances[index] = params.BeaconConfig().EjectionBalance - 1
		}
	}
	fmt.Println("ejections created")

	// Exits and Activations
	exitCount := uint64(30)
	activationCount := uint64(30)
	exitEpoch := helpers.EntryExitEffectEpoch(helpers.CurrentEpoch(bState))
	for index := range bState.ValidatorRegistry {
		if uint64(index)%(validatorCount/exitCount)-3 == 0 {
			bState.ValidatorRegistry[index].ExitEpoch = exitEpoch
			bState.ValidatorRegistry[index].StatusFlags = pb.Validator_INITIATED_EXIT
		} else if uint64(index)%(validatorCount/activationCount)-4 == 0 {
			bState.ValidatorRegistry[index].ExitEpoch = params.BeaconConfig().ActivationExitDelay
			bState.ValidatorRegistry[index].ActivationEpoch = 5 + params.BeaconConfig().ActivationExitDelay + 1
		}
	}
	return bState
}

func createGenesisState(numDeposits uint64) *pb.BeaconState {
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
