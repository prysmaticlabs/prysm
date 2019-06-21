package epoch_test

import (
	"context"
	"github.com/gogo/protobuf/proto"
	"strconv"
	"testing"

	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bitutil"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

var RunAmount = 100

// var conditions = "MAX"

var beaconState16K = createFullState(16384)

// var beaconState300K = createFullState(300000)
var beaconStates16K = createCleanStates16K(RunAmount)

// var beaconStates300K = createCleanStates300K(RunAmount)

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
	c.MaxAttestations = 4
	// 	c.MaxDeposits = 2
	// 	c.MaxVoluntaryExits = 2
	// }
	params.OverrideBeaconConfig(c)

	featureCfg := &featureconfig.FeatureFlagConfig{
		EnableCrosslinks: false,
	}
	featureconfig.InitFeatureConfig(featureCfg)
}

func BenchmarkProcessJustificationAndFinalization(b *testing.B) {
	var err error
	prevEpoch := helpers.PrevEpoch(beaconState16K)
	currentEpoch := helpers.CurrentEpoch(beaconState16K)

	prevEpochAtts, err := e.MatchAttestations(beaconState16K, prevEpoch)
	if err != nil {
		b.Fatal(err)
	}
	currentEpochAtts, err := e.MatchAttestations(beaconState16K, currentEpoch)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochAttestedBalance, err := e.AttestingBalance(beaconState16K, prevEpochAtts.Target)
	if err != nil {
		b.Fatal(err)
	}
	currentEpochAttestedBalance, err := e.AttestingBalance(beaconState16K, currentEpochAtts.Target)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = e.ProcessJustificationAndFinalization(
				beaconStates16K[i],
				prevEpochAttestedBalance,
				currentEpochAttestedBalance,
			)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkProcessCrosslinks(b *testing.B) {
	var err error

	b.Run("16K", func(b *testing.B) {
		b.N = 5
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = epoch.ProcessCrosslinks(beaconStates16K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// currentEpochAttestations = e.CurrentAttestations(beaconState300K)
	// prevEpochAttestations = e.PrevAttestations(beaconState300K)

	// b.Run("300K", func(b *testing.B) {
	// 	b.N = 10
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_, err := epoch.ProcessCrosslinks(beaconState300K, currentEpochAttestations, prevEpochAttestations)
	// 		if err != nil {
	// 			b.Fatal(err)
	// 		}
	// 	}
	// })
}

func BenchmarkProcessRewardsAndPenalties(b *testing.B) {
	var err error

	b.Run("16K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = epoch.ProcessRewardsAndPenalties(beaconStates16K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// b.Run("300K", func(b *testing.B) {
	// 	b.N = 10
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_, err = epoch.ProcessRewardsAndPenalties(beaconStates300K[i])
	// 		if err != nil {
	// 			b.Fatal(err)
	// 		}
	// 	}
	// })
}

func BenchmarkProcessRegistryUpdates(b *testing.B) {
	var err error

	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = epoch.ProcessRegistryUpdates(beaconStates16K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// b.Run("300K", func(b *testing.B) {
	// 	b.N = RunAmount
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_, err = epoch.ProcessRegistryUpdates(beaconStates300K[i])
	// 		if err != nil {
	// 			b.Fatal(err)
	// 		}
	// 	}
	// })
}

func BenchmarkProcessSlashings(b *testing.B) {
	var err error

	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = epoch.ProcessSlashings(beaconStates16K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// b.Run("300K", func(b *testing.B) {
	// 	b.N = RunAmount
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_, err = epoch.ProcessSlashings(beaconStates300K[i])
	// 		if err != nil {
	// 			b.Fatal(err)
	// 		}
	// 	}
	// })
}

func BenchmarkProcessFinalUpdates(b *testing.B) {
	var err error

	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = epoch.ProcessFinalUpdates(beaconStates16K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// b.Run("300K", func(b *testing.B) {
	// 	b.N = RunAmount
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_, err = epoch.ProcessFinalUpdates(beaconStates300K[i])
	// 		if err != nil {
	// 			b.Fatal(err)
	// 		}
	// 	}
	// })
}

func BenchmarkProcessEpoch(b *testing.B) {
	b.Run("16K", func(b *testing.B) {
		b.N = 5
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := state.ProcessEpoch(context.Background(), beaconStates16K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// b.Run("300K", func(b *testing.B) {
	// 	b.N = 10
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_, err := state.ProcessEpoch(context.Background(), beaconState300K, nil, cfg)
	// 		if err != nil {
	// 			b.Fatal(err)
	// 		}
	// 	}
	// })
}

func BenchmarkActiveValidatorIndices(b *testing.B) {
	currentEpoch := helpers.CurrentEpoch(beaconState16K)

	var err error
	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = helpers.ActiveValidatorIndices(beaconStates16K[i], currentEpoch)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	// b.Run("300K", func(b *testing.B) {
	// 	b.N = RunAmount
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_, err = helpers.ActiveValidatorIndices(beaconStates300K[i], currentEpoch)
	// 		if err != nil {
	// 			b.Fatal(err)
	// 		}
	// 	}
	// })
}

func BenchmarkValidatorIndexMap(b *testing.B) {
	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = stateutils.ValidatorIndexMap(beaconState16K)
		}
	})

	// b.Run("300K", func(b *testing.B) {
	// 	b.N = RunAmount
	// 	b.ResetTimer()
	// 	for i := 0; i < b.N; i++ {
	// 		_ = stateutils.ValidatorIndexMap(beaconState300K)
	// 	}
	// })
}

func createFullState(validatorCount uint64) *pb.BeaconState {
	bState := createGenesisState(validatorCount)

	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	currentSlot := params.BeaconConfig().SlotsPerEpoch*2048 - 1
	bState.Slot = currentSlot
	bState.FinalizedEpoch = helpers.SlotToEpoch(currentSlot) - 1
	bState.JustificationBitfield = 4

	prevEpoch := helpers.PrevEpoch(bState)
	currentEpoch := helpers.CurrentEpoch(bState)

	committeeCount, err := helpers.EpochCommitteeCount(bState, currentEpoch)
	if err != nil {
		panic(err)
	}
	committeeSize := int(validatorCount / committeeCount)

	attestationsPerEpoch := slotsPerEpoch * params.BeaconConfig().MaxAttestations

	var prevAttestations []*pb.PendingAttestation
	for i := uint64(0); i < attestationsPerEpoch; i++ {
		// attestationSlot := (prevEpoch * slotsPerEpoch) + (i % slotsPerEpoch)
		aggregationBitfield, err := bitutil.SetBitfield(int(i)%committeeSize, committeeSize)
		if err != nil {
			panic(err)
		}

		crosslink := &pb.Crosslink{
			Shard:      i % params.BeaconConfig().ShardCount,
			StartEpoch: prevEpoch - 1,
			EndEpoch:   prevEpoch,
		}

		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Crosslink:       crosslink,
				SourceEpoch:     prevEpoch - 1,
				TargetEpoch:     prevEpoch,
				BeaconBlockRoot: params.BeaconConfig().ZeroHash[:],
				SourceRoot:      params.BeaconConfig().ZeroHash[:],
				TargetRoot:      params.BeaconConfig().ZeroHash[:],
			},
			AggregationBitfield: aggregationBitfield,
			InclusionDelay:      params.BeaconConfig().MinAttestationInclusionDelay,
		}
		prevAttestations = append(prevAttestations, attestation)
	}
	bState.PreviousEpochAttestations = prevAttestations

	var currentAttestations []*pb.PendingAttestation
	for i := uint64(0); i < attestationsPerEpoch; i++ {
		aggregationBitfield, err := bitutil.SetBitfield(int(i)%committeeSize, committeeSize)
		if err != nil {
			panic(err)
		}

		crosslink := &pb.Crosslink{
			Shard:      i % params.BeaconConfig().ShardCount,
			StartEpoch: currentEpoch - 1,
			EndEpoch:   currentEpoch,
		}

		attestation := &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Crosslink:       crosslink,
				SourceEpoch:     currentEpoch - 1,
				TargetEpoch:     currentEpoch,
				BeaconBlockRoot: params.BeaconConfig().ZeroHash[:],
				SourceRoot:      params.BeaconConfig().ZeroHash[:],
				TargetRoot:      params.BeaconConfig().ZeroHash[:],
			},
			AggregationBitfield: aggregationBitfield,
			InclusionDelay:      params.BeaconConfig().MinAttestationInclusionDelay,
		}

		currentAttestations = append(currentAttestations, attestation)
	}
	bState.CurrentEpochAttestations = currentAttestations

	// RANDAO
	var randaoHashes [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestRandaoMixesLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}
	bState.LatestRandaoMixes = randaoHashes

	// RANDAO
	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	bState.LatestBlockRoots = blockRoots

	// LatestSlashedBalances
	latestSlashedBalances := make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
	for i := 0; i < len(latestSlashedBalances); i++ {
		latestSlashedBalances[i] = uint64(i) * params.BeaconConfig().MaxDepositAmount
	}
	bState.LatestSlashedBalances = latestSlashedBalances

	// Exits and Activations
	exitCount := uint64(40)
	ejectionCount := uint64(40)
	activationCount := uint64(40)
	initiateActivationCount := uint64(40)
	for index, val := range bState.ValidatorRegistry {
		if uint64(index)%(validatorCount/ejectionCount) == 0 {
			// Ejections
			val.EffectiveBalance = params.BeaconConfig().EjectionBalance - 1
		}
		if uint64(index)%(validatorCount/exitCount)-3 == 0 {
			// Exits
			val.ExitEpoch = currentEpoch
			val.WithdrawableEpoch = currentEpoch + 4
			val.EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		} else if uint64(index)%(validatorCount/activationCount)-5 == 0 {
			// Activations
			activationEpoch := currentEpoch - 1 - params.BeaconConfig().ActivationExitDelay
			val.Slashed = false
			val.ExitEpoch = params.BeaconConfig().FarFutureEpoch
			val.ActivationEpoch = params.BeaconConfig().FarFutureEpoch
			val.ActivationEligibilityEpoch = activationEpoch
			val.EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		} else if uint64(index)%(validatorCount/initiateActivationCount)-7 == 0 {
			// Initiations
			val.Slashed = false
			val.ExitEpoch = params.BeaconConfig().FarFutureEpoch
			val.ActivationEpoch = params.BeaconConfig().FarFutureEpoch
			val.ActivationEligibilityEpoch = params.BeaconConfig().FarFutureEpoch
			val.EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		}
	}

	return bState
}

func createGenesisState(numDeposits uint64) *pb.BeaconState {
	setBenchmarkConfig()
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		pubkey := []byte{}
		pubkey = make([]byte, params.BeaconConfig().BLSPubkeyLength)
		copy(pubkey[:], []byte(strconv.FormatUint(uint64(i), 10)))

		depositData := &pb.DepositData{
			Pubkey:                pubkey,
			Amount:                params.BeaconConfig().MaxDepositAmount,
			WithdrawalCredentials: []byte{1},
		}
		deposits[i] = &pb.Deposit{
			Data:  depositData,
			Index: uint64(i),
		}
	}

	encodedDeposits := make([][]byte, len(deposits))
	for i := 0; i < len(encodedDeposits); i++ {
		hashedDeposit, err := ssz.HashTreeRoot(deposits[i].Data)
		if err != nil {
			panic(err)
		}
		encodedDeposits[i] = hashedDeposit[:]
	}

	depositTrie, err := trieutil.GenerateTrieFromItems(encodedDeposits, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		panic(err)
	}

	for i := range deposits {
		proof, err := depositTrie.MerkleProof(int(deposits[i].Index))
		if err != nil {
			panic(err)
		}
		deposits[i].Proof = proof
	}

	root := depositTrie.Root()
	eth1Data := &pb.Eth1Data{
		BlockHash:   root[:],
		DepositRoot: root[:],
	}

	genesisState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		panic(err)
	}

	return genesisState
}

func createCleanStates16K(num int) []*pb.BeaconState {
	cleanStates := make([]*pb.BeaconState, num)
	for i := 0; i < num; i++ {
		cleanStates[i] = proto.Clone(beaconState16K).(*pb.BeaconState)
	}
	return cleanStates
}

// func createCleanStates300K(num int) []*pb.BeaconState {
// 	cleanStates := make([]*pb.BeaconState, num)
// 	for i := 0; i < num; i++ {
// 		cleanStates[i] = proto.Clone(beaconState300K).(*pb.BeaconState)
// 	}
// 	return cleanStates
// }
