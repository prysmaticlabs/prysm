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

var RunAmount = 32

var conditions = "MIN"

var genesisBeaconState16K = createFullState(16384)
var genesisBeaconState100K = createFullState(100000)

var beaconStates16K = createCleanStates16K(RunAmount)
var beaconStates100K = createCleanStates100K(RunAmount)

func setBenchmarkConfig() {
	c := params.DemoBeaconConfig()
	if conditions == "MAX" {
		c.MaxAttestations = 128
		c.MaxDeposits = 16
		c.MaxVoluntaryExits = 16
	} else if conditions == "MIN" {
		c.MaxAttestations = 4
		// c.MaxDeposits = 2
		// c.MaxVoluntaryExits = 2
	}
	params.OverrideBeaconConfig(c)

	featureCfg := &featureconfig.FeatureFlagConfig{
		EnableCrosslinks: false,
	}
	featureconfig.InitFeatureConfig(featureCfg)
}

func BenchmarkActiveValidatorIndices(b *testing.B) {
	currentEpoch := helpers.CurrentEpoch(genesisBeaconState16K)

	var err error
	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = helpers.ActiveValidatorIndices(genesisBeaconState16K, currentEpoch)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("100K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = helpers.ActiveValidatorIndices(genesisBeaconState100K, currentEpoch)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkValidatorIndexMap(b *testing.B) {
	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = stateutils.ValidatorIndexMap(genesisBeaconState16K)
		}
	})

	b.Run("100K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = stateutils.ValidatorIndexMap(genesisBeaconState100K)
		}
	})
}

func BenchmarkProcessJustificationAndFinalization(b *testing.B) {
	var err error
	prevEpoch := helpers.PrevEpoch(genesisBeaconState16K)
	currentEpoch := helpers.CurrentEpoch(genesisBeaconState16K)

	prevEpochAtts, err := e.MatchAttestations(genesisBeaconState16K, prevEpoch)
	if err != nil {
		b.Fatal(err)
	}
	currentEpochAtts, err := e.MatchAttestations(genesisBeaconState16K, currentEpoch)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochAttestedBalance, err := e.AttestingBalance(genesisBeaconState16K, prevEpochAtts.Target)
	if err != nil {
		b.Fatal(err)
	}
	currentEpochAttestedBalance, err := e.AttestingBalance(genesisBeaconState16K, currentEpochAtts.Target)
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

	prevEpochAtts, err = e.MatchAttestations(genesisBeaconState100K, prevEpoch)
	if err != nil {
		b.Fatal(err)
	}
	currentEpochAtts, err = e.MatchAttestations(genesisBeaconState100K, currentEpoch)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochAttestedBalance, err = e.AttestingBalance(genesisBeaconState100K, prevEpochAtts.Target)
	if err != nil {
		b.Fatal(err)
	}
	currentEpochAttestedBalance, err = e.AttestingBalance(genesisBeaconState100K, currentEpochAtts.Target)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("100K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = e.ProcessJustificationAndFinalization(
				beaconStates100K[i],
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
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = epoch.ProcessCrosslinks(beaconStates16K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("100K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := epoch.ProcessCrosslinks(beaconStates100K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkProcessRewardsAndPenalties(b *testing.B) {
	var err error

	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = epoch.ProcessRewardsAndPenalties(beaconStates16K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("100K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = epoch.ProcessRewardsAndPenalties(beaconStates100K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})
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

	b.Run("100K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = epoch.ProcessRegistryUpdates(beaconStates100K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})
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

	b.Run("100K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = epoch.ProcessSlashings(beaconStates100K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})
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

	b.Run("100K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err = epoch.ProcessFinalUpdates(beaconStates100K[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkProcessEpoch(b *testing.B) {
	var beaconStates16KEpoch = createCleanStates16K(RunAmount)

	b.Run("16K", func(b *testing.B) {
		b.N = RunAmount
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := state.ProcessEpoch(context.Background(), beaconStates16KEpoch[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	var beaconStates100KEpoch = createCleanStates100K(RunAmount)

	b.Run("100K", func(b *testing.B) {
		b.N = 10
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := state.ProcessEpoch(context.Background(), beaconStates100KEpoch[i])
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func createFullState(validatorCount uint64) *pb.BeaconState {
	bState := createGenesisState(validatorCount)

	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	epochsPerHistoricalRoot := params.BeaconConfig().SlotsPerHistoricalRoot / params.BeaconConfig().SlotsPerEpoch
	bState.Slot = epochsPerHistoricalRoot*4*slotsPerEpoch - 1
	bState.FinalizedEpoch = helpers.SlotToEpoch(bState.Slot) - 2
	bState.PreviousJustifiedEpoch = helpers.SlotToEpoch(bState.Slot) - 2
	bState.CurrentJustifiedEpoch = helpers.SlotToEpoch(bState.Slot) - 1
	bState.JustificationBitfield = 4

	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	bState.LatestBlockRoots = blockRoots

	var randaoHashes [][]byte
	for i := uint64(0); i < params.BeaconConfig().LatestRandaoMixesLength; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}
	bState.LatestRandaoMixes = randaoHashes

	latestSlashedBalances := make([]uint64, params.BeaconConfig().LatestSlashedExitLength)
	for i := 0; i < len(latestSlashedBalances); i++ {
		latestSlashedBalances[i] = uint64(i) * params.BeaconConfig().MaxDepositAmount
	}
	bState.LatestSlashedBalances = latestSlashedBalances

	prevEpoch := helpers.PrevEpoch(bState)
	currentEpoch := helpers.CurrentEpoch(bState)

	// Exits and Activations
	exitCount := uint64(40)
	slashCount := uint64(40)
	ejectionCount := uint64(40)
	activationCount := uint64(40)
	initiateActivationCount := uint64(40)
	for index, val := range bState.ValidatorRegistry {
		if uint64(index)%(validatorCount/ejectionCount) == 0 {
			// Ejections
			val.Slashed = false
			val.EffectiveBalance = params.BeaconConfig().EjectionBalance - 1
		}
		if uint64(index)%(validatorCount/exitCount)-1 == 0 {
			// Exits
			val.Slashed = false
			val.ExitEpoch = currentEpoch
			val.WithdrawableEpoch = currentEpoch + 4
			val.EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		} else if uint64(index)%(validatorCount/activationCount)-2 == 0 {
			// Activations
			activationEpoch := currentEpoch - 1 - params.BeaconConfig().ActivationExitDelay
			val.Slashed = false
			val.ExitEpoch = params.BeaconConfig().FarFutureEpoch
			val.ActivationEpoch = params.BeaconConfig().FarFutureEpoch
			val.ActivationEligibilityEpoch = activationEpoch
			val.EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		} else if uint64(index)%(validatorCount/initiateActivationCount)-3 == 0 {
			// Initiations
			val.Slashed = false
			val.ExitEpoch = params.BeaconConfig().FarFutureEpoch
			val.ActivationEpoch = params.BeaconConfig().FarFutureEpoch
			val.ActivationEligibilityEpoch = params.BeaconConfig().FarFutureEpoch
			val.EffectiveBalance = params.BeaconConfig().MaxEffectiveBalance
		} else if uint64(index)%(validatorCount/slashCount)-4 == 0 {
			// Slashes
			val.Slashed = true
			val.WithdrawableEpoch = currentEpoch + params.BeaconConfig().LatestSlashedExitLength/2
		}
	}

	prevCommitteeCount, err := helpers.EpochCommitteeCount(bState, prevEpoch)
	if err != nil {
		panic(err)
	}
	prevValidatorCount, err := helpers.ActiveValidatorCount(bState, prevEpoch)
	if err != nil {
		panic(err)
	}
	prevCommitteeSize := int(prevValidatorCount / prevCommitteeCount)

	attestationsPerEpoch := slotsPerEpoch * params.BeaconConfig().MaxAttestations

	prevRoot, err := helpers.BlockRoot(bState, prevEpoch)
	if err != nil {
		panic(err)
	}

	var prevAttestations []*pb.PendingAttestation
	for i := uint64(0); i < attestationsPerEpoch; i++ {
		aggregationBitfield, err := bitutil.SetBitfield(int(i)%prevCommitteeSize, prevCommitteeSize)
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
				SourceRoot:      prevRoot,
				TargetRoot:      prevRoot,
			},
			AggregationBitfield: aggregationBitfield,
			InclusionDelay:      params.BeaconConfig().MinAttestationInclusionDelay,
		}

		slot, err := helpers.AttestationDataSlot(bState, attestation.Data)
		if err != nil {
			panic(err)
		}
		headRoot, err := helpers.BlockRootAtSlot(bState, slot)
		if err != nil {
			panic(err)
		}

		attestation.Data.BeaconBlockRoot = headRoot

		prevAttestations = append(prevAttestations, attestation)
	}
	bState.PreviousEpochAttestations = prevAttestations

	curCommitteeCount, err := helpers.EpochCommitteeCount(bState, currentEpoch)
	if err != nil {
		panic(err)
	}
	curValidatorCount, err := helpers.ActiveValidatorCount(bState, currentEpoch)
	if err != nil {
		panic(err)
	}
	curCommitteeSize := int(curValidatorCount / curCommitteeCount)

	var currentAttestations []*pb.PendingAttestation
	currentRoot, err := helpers.BlockRoot(bState, currentEpoch)
	if err != nil {
		panic(err)
	}
	for i := uint64(0); i < attestationsPerEpoch; i++ {
		aggregationBitfield, err := bitutil.SetBitfield(int(i)%curCommitteeSize, curCommitteeSize)
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
				SourceRoot:      currentRoot,
				TargetRoot:      currentRoot,
			},
			AggregationBitfield: aggregationBitfield,
			InclusionDelay:      params.BeaconConfig().MinAttestationInclusionDelay,
		}

		slot, err := helpers.AttestationDataSlot(bState, attestation.Data)
		if err != nil {
			panic(err)
		}
		headRoot, err := helpers.BlockRootAtSlot(bState, slot)
		if err != nil {
			panic(err)
		}

		attestation.Data.BeaconBlockRoot = headRoot

		currentAttestations = append(currentAttestations, attestation)
	}
	bState.CurrentEpochAttestations = currentAttestations

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
			Data: depositData,
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
		proof, err := depositTrie.MerkleProof(i)
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
		cleanStates[i] = proto.Clone(genesisBeaconState16K).(*pb.BeaconState)
	}
	return cleanStates
}

func createCleanStates100K(num int) []*pb.BeaconState {
	cleanStates := make([]*pb.BeaconState, num)
	for i := 0; i < num; i++ {
		cleanStates[i] = proto.Clone(genesisBeaconState100K).(*pb.BeaconState)
	}
	return cleanStates
}
