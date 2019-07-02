package epoch_test

import (
	"context"
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
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
	"github.com/sirupsen/logrus"
)

func setBenchmarkConfig(conditions string, validatorCount uint64) {
	logrus.Printf("Running epoch benchmarks for %d validators", validatorCount)
	logrus.SetLevel(logrus.PanicLevel)
	logrus.SetOutput(ioutil.Discard)
	c := params.DemoBeaconConfig()
	if conditions == "BIG" {
		c.MaxAttestations = 128
		c.MaxDeposits = 16
		c.MaxVoluntaryExits = 16
	} else if conditions == "SML" {
		c.MaxAttesterSlashings = 1
		c.MaxProposerSlashings = 1
		c.MaxAttestations = 16
		c.MaxDeposits = 2
		c.MaxVoluntaryExits = 2
	}
	params.OverrideBeaconConfig(c)

	featureCfg := &featureconfig.FeatureFlagConfig{
		EnableCrosslinks: false,
	}
	featureconfig.InitFeatureConfig(featureCfg)
}

func cleanUpConfigs() {
	params.OverrideBeaconConfig(params.BeaconConfig())
}

func BenchmarkProcessJustificationAndFinalization(b *testing.B) {
	genesisBeaconState := createFullState()
	beaconStates := createCleanStates(genesisBeaconState)
	prevEpoch := helpers.PrevEpoch(genesisBeaconState)
	currentEpoch := helpers.CurrentEpoch(genesisBeaconState)

	prevEpochAtts, err := e.MatchAttestations(genesisBeaconState, prevEpoch)
	if err != nil {
		b.Fatal(err)
	}
	currentEpochAtts, err := e.MatchAttestations(genesisBeaconState, currentEpoch)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochAttestedBalance, err := e.AttestingBalance(genesisBeaconState, prevEpochAtts.Target)
	if err != nil {
		b.Fatal(err)
	}
	currentEpochAttestedBalance, err := e.AttestingBalance(genesisBeaconState, currentEpochAtts.Target)
	if err != nil {
		b.Fatal(err)
	}

	b.N = 42
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := e.ProcessJustificationAndFinalization(
			beaconStates[i],
			prevEpochAttestedBalance,
			currentEpochAttestedBalance,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUpConfigs()
}

func BenchmarkProcessCrosslinks(b *testing.B) {
	genesisBeaconState := createFullState()
	beaconStates := createCleanStates(genesisBeaconState)
	b.N = 42
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := epoch.ProcessCrosslinks(beaconStates[i])
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUpConfigs()
}

func BenchmarkProcessRewardsAndPenalties(b *testing.B) {
	genesisBeaconState := createFullState()
	beaconStates := createCleanStates(genesisBeaconState)
	b.N = 42
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := epoch.ProcessRewardsAndPenalties(beaconStates[i])
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUpConfigs()
}

func BenchmarkProcessRegistryUpdates(b *testing.B) {
	genesisBeaconState := createFullState()
	beaconStates := createCleanStates(genesisBeaconState)
	b.N = 42
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := epoch.ProcessRegistryUpdates(beaconStates[i])
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUpConfigs()
}

func BenchmarkProcessSlashings(b *testing.B) {
	genesisBeaconState := createFullState()
	beaconStates := createCleanStates(genesisBeaconState)
	b.N = 42
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := epoch.ProcessSlashings(beaconStates[i])
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUpConfigs()
}

func BenchmarkProcessFinalUpdates(b *testing.B) {
	genesisBeaconState := createFullState()
	beaconStates := createCleanStates(genesisBeaconState)
	b.N = 42
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := epoch.ProcessFinalUpdates(beaconStates[i])
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUpConfigs()
}

func BenchmarkProcessEpoch(b *testing.B) {
	genesisBeaconState := createFullState()
	beaconStates := createCleanStates(genesisBeaconState)
	b.N = 42
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := state.ProcessEpoch(context.Background(), beaconStates[i])
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUpConfigs()
}

func BenchmarkActiveValidatorIndices(b *testing.B) {
	genesisBeaconState := createFullState()
	currentEpoch := helpers.CurrentEpoch(genesisBeaconState)

	b.N = 100
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := helpers.ActiveValidatorIndices(genesisBeaconState, currentEpoch)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUpConfigs()
}

func BenchmarkValidatorIndexMap(b *testing.B) {
	genesisBeaconState := createFullState()
	b.N = 100
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stateutils.ValidatorIndexMap(genesisBeaconState)
	}
	cleanUpConfigs()
}

func createFullState() *pb.BeaconState {
	validatorCount := uint64(65536)
	bState := createGenesisState(validatorCount)

	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	epochsPerHistoricalRoot := params.BeaconConfig().SlotsPerHistoricalRoot / params.BeaconConfig().SlotsPerEpoch
	bState.Slot = epochsPerHistoricalRoot*4*slotsPerEpoch - 1
	bState.FinalizedCheckpoint.Epoch = helpers.SlotToEpoch(bState.Slot) - 2
	bState.PreviousJustifiedCheckpoint.Epoch = helpers.SlotToEpoch(bState.Slot) - 2
	bState.CurrentJustifiedCheckpoint.Epoch = helpers.SlotToEpoch(bState.Slot) - 1
	bState.JustificationBits = []byte{4}

	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().SlotsPerHistoricalRoot; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	bState.BlockRoots = blockRoots

	var randaoHashes [][]byte
	for i := uint64(0); i < params.BeaconConfig().EpochsPerHistoricalVector; i++ {
		randaoHashes = append(randaoHashes, []byte{byte(i)})
	}
	bState.RandaoMixes = randaoHashes

	slashedBalances := make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)
	for i := 0; i < len(slashedBalances); i++ {
		slashedBalances[i] = uint64(i) * params.BeaconConfig().MaxEffectiveBalance
	}
	bState.Slashings = slashedBalances

	prevEpoch := helpers.PrevEpoch(bState)
	currentEpoch := helpers.CurrentEpoch(bState)

	// Registry Changes
	var exitCount = uint64(40)
	var slashCount = uint64(40)
	var ejectionCount = uint64(40)
	var activationCount = uint64(40)
	var initiateActivationCount = uint64(40)
	for index, val := range bState.Validators {
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
			val.WithdrawableEpoch = currentEpoch + params.BeaconConfig().EpochsPerSlashingsVector/2
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
		aggregationBits, err := bitutil.SetBitfield(int(i)%prevCommitteeSize, prevCommitteeSize)
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
			AggregationBits: aggregationBits,
			InclusionDelay:  params.BeaconConfig().MinAttestationInclusionDelay,
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
		aggregationBits, err := bitutil.SetBitfield(int(i)%curCommitteeSize, curCommitteeSize)
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
			AggregationBits: aggregationBits,
			InclusionDelay:  params.BeaconConfig().MinAttestationInclusionDelay,
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
	setBenchmarkConfig("SML", numDeposits)
	deposits := make([]*pb.Deposit, numDeposits)
	for i := 0; i < len(deposits); i++ {
		pubkey := []byte{}
		pubkey = make([]byte, params.BeaconConfig().BLSPubkeyLength)
		copy(pubkey[:], []byte(strconv.FormatUint(uint64(i), 10)))

		depositData := &pb.DepositData{
			Pubkey:                pubkey,
			Amount:                params.BeaconConfig().MaxEffectiveBalance,
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

func createCleanStates(beaconState *pb.BeaconState) []*pb.BeaconState {
	cleanStates := make([]*pb.BeaconState, 42)
	for i := 0; i < len(cleanStates); i++ {
		cleanStates[i] = proto.Clone(beaconState).(*pb.BeaconState)
	}
	return cleanStates
}
