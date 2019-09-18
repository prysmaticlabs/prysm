package epoch_test

import (
	"context"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"io/ioutil"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	e "github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state/stateutils"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/sirupsen/logrus"
)

var validatorCount = uint64(128)
var runAmount = 10
var conditions = "SML"

var deposits, privs = testutil.GenerateDeposits(&testing.B{}, uint64(validatorCount))

func setBenchmarkConfig() {
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
}

func cleanUpConfigs() {
	params.OverrideBeaconConfig(params.BeaconConfig())
}

func TestBenchmarkEpoch_PerformsSuccessfully(t *testing.T) {
	beaconState := createFullState(t)
	_, err := state.ProcessEpoch(context.Background(), beaconState)
	if err != nil {
		t.Fatalf("failed to process epoch, benchmarks will fail: %v", err)
	}
	cleanUpConfigs()
}

func BenchmarkProcessJustificationAndFinalization(b *testing.B) {
	beaconState := createFullState(b)
	beaconStates := createCleanStates(beaconState)
	prevEpoch := helpers.PrevEpoch(beaconState)
	currentEpoch := helpers.CurrentEpoch(beaconState)

	prevEpochAtts, err := e.MatchAttestations(beaconState, prevEpoch)
	if err != nil {
		b.Fatal(err)
	}
	currentEpochAtts, err := e.MatchAttestations(beaconState, currentEpoch)
	if err != nil {
		b.Fatal(err)
	}
	prevEpochAttestedBalance, err := e.AttestingBalance(beaconState, prevEpochAtts.Target)
	if err != nil {
		b.Fatal(err)
	}
	currentEpochAttestedBalance, err := e.AttestingBalance(beaconState, currentEpochAtts.Target)
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
	beaconState := createFullState(b)
	beaconStates := createCleanStates(beaconState)
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
	beaconState := createFullState(b)
	beaconStates := createCleanStates(beaconState)
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
	beaconState := createFullState(b)
	beaconStates := createCleanStates(beaconState)
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
	beaconState := createFullState(b)
	beaconStates := createCleanStates(beaconState)
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
	beaconState := createFullState(b)
	beaconStates := createCleanStates(beaconState)
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
	beaconState := createFullState(b)
	beaconStates := createCleanStates(beaconState)
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
	beaconState := createFullState(b)
	currentEpoch := helpers.CurrentEpoch(beaconState)

	b.N = 100
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := helpers.ActiveValidatorIndices(beaconState, currentEpoch)
		if err != nil {
			b.Fatal(err)
		}
	}
	cleanUpConfigs()
}

func BenchmarkValidatorIndexMap(b *testing.B) {
	beaconState := createFullState(b)
	b.N = 100
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stateutils.ValidatorIndexMap(beaconState)
	}
	cleanUpConfigs()
}

func createFullState(b testing.TB) *pb.BeaconState {
	bState := createBeaconState(b)

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

	prevCommitteeCount, err := helpers.CommitteeCount(bState, prevEpoch)
	if err != nil {
		panic(err)
	}
	prevValidatorCount, err := helpers.ActiveValidatorCount(bState, prevEpoch)
	if err != nil {
		panic(err)
	}
	prevCommitteeSize := prevValidatorCount / prevCommitteeCount

	attestationsPerEpoch := slotsPerEpoch * params.BeaconConfig().MaxAttestations

	prevRoot, err := helpers.BlockRoot(bState, prevEpoch)
	if err != nil {
		panic(err)
	}

	var prevAttestations []*pb.PendingAttestation
	for i := uint64(0); i < attestationsPerEpoch; i++ {
		aggregationBits := bitfield.NewBitlist(prevCommitteeSize)
		aggregationBits.SetBitAt(i%prevCommitteeSize, true)

		crosslink := &ethpb.Crosslink{
			Shard:      i % params.BeaconConfig().ShardCount,
			StartEpoch: prevEpoch - 1,
			EndEpoch:   prevEpoch,
		}

		attestation := &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Crosslink: crosslink,
				Source: &ethpb.Checkpoint{
					Epoch: prevEpoch - 1,
					Root:  prevRoot,
				},
				Target: &ethpb.Checkpoint{
					Epoch: prevEpoch,
					Root:  prevRoot,
				},
				BeaconBlockRoot: params.BeaconConfig().ZeroHash[:],
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

	curCommitteeCount, err := helpers.CommitteeCount(bState, currentEpoch)
	if err != nil {
		panic(err)
	}
	curValidatorCount, err := helpers.ActiveValidatorCount(bState, currentEpoch)
	if err != nil {
		panic(err)
	}
	curCommitteeSize := curValidatorCount / curCommitteeCount

	var currentAttestations []*pb.PendingAttestation
	currentRoot, err := helpers.BlockRoot(bState, currentEpoch)
	if err != nil {
		panic(err)
	}
	for i := uint64(0); i < attestationsPerEpoch; i++ {
		aggregationBits := bitfield.NewBitlist(curCommitteeSize)
		aggregationBits.SetBitAt(i%curCommitteeSize, true)

		crosslink := &ethpb.Crosslink{
			Shard:      i % params.BeaconConfig().ShardCount,
			StartEpoch: currentEpoch - 1,
			EndEpoch:   currentEpoch,
		}

		attestation := &pb.PendingAttestation{
			Data: &ethpb.AttestationData{
				Crosslink: crosslink,
				Source: &ethpb.Checkpoint{
					Epoch: currentEpoch - 1,
					Root:  currentRoot,
				},
				Target: &ethpb.Checkpoint{
					Epoch: currentEpoch,
					Root:  currentRoot,
				},
				BeaconBlockRoot: params.BeaconConfig().ZeroHash[:],
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

func createBeaconState(b testing.TB) *pb.BeaconState {
	setBenchmarkConfig()
	eth1Data := testutil.GenerateEth1Data(b, deposits)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), eth1Data)
	if err != nil {
		b.Fatal(err)
	}

	beaconState.Slot = params.BeaconConfig().PersistentCommitteePeriod*8 + (params.BeaconConfig().SlotsPerEpoch / 2)
	beaconState.CurrentJustifiedCheckpoint.Epoch = helpers.PrevEpoch(beaconState)
	crosslinks := make([]*ethpb.Crosslink, params.BeaconConfig().ShardCount)
	for i := 0; i < len(crosslinks); i++ {
		crosslinks[i] = &ethpb.Crosslink{
			Shard:      uint64(i),
			StartEpoch: helpers.PrevEpoch(beaconState) - 1,
			EndEpoch:   helpers.PrevEpoch(beaconState),
			DataRoot:   params.BeaconConfig().ZeroHash[:],
		}
	}
	beaconState.CurrentCrosslinks = crosslinks

	beaconState.LatestBlockHeader = &ethpb.BeaconBlockHeader{
		Slot: beaconState.Slot,
	}

	return beaconState
}

func createCleanStates(beaconState *pb.BeaconState) []*pb.BeaconState {
	cleanStates := make([]*pb.BeaconState, 42)
	for i := 0; i < len(cleanStates); i++ {
		cleanStates[i] = proto.Clone(beaconState).(*pb.BeaconState)
	}
	return cleanStates
}
