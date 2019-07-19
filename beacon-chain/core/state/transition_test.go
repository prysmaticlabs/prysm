package state_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/go-ssz"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/blocks"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/state"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bls"
	"github.com/prysmaticlabs/prysm/shared/featureconfig"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
	"github.com/sirupsen/logrus"
)

func init() {
	featureconfig.InitFeatureConfig(&featureconfig.FeatureFlagConfig{
		EnableCrosslinks: true,
	})
}

func TestExecuteStateTransition_IncorrectSlot(t *testing.T) {
	beaconState := &pb.BeaconState{
		Slot: 5,
	}
	block := &pb.BeaconBlock{
		Slot: 4,
	}
	want := "expected state.slot"
	if _, err := state.ExecuteStateTransition(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestExecuteStateTransition_FullProcess(t *testing.T) {
	deposits, _ := testutil.SetupInitialDeposits(t, 100, true)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), nil)
	if err != nil {
		t.Fatal(err)
	}
	eth1Data := &pb.Eth1Data{
		DepositCount: 100,
		DepositRoot:  []byte{2},
	}
	beaconState.Slot = params.BeaconConfig().SlotsPerEpoch - 1
	beaconState.Eth1Data.DepositCount = 100
	beaconState.LatestBlockHeader = &pb.BeaconBlockHeader{Slot: beaconState.Slot}
	beaconState.Eth1DataVotes = []*pb.Eth1Data{eth1Data}

	oldMix := beaconState.RandaoMixes[1]
	oldStartShard := beaconState.StartShard

	parentRoot, err := ssz.SigningRoot(beaconState.LatestBlockHeader)
	if err != nil {
		t.Error(err)
	}

	block := &pb.BeaconBlock{
		Slot:       beaconState.Slot + 1,
		ParentRoot: parentRoot[:],
		Body: &pb.BeaconBlockBody{
			RandaoReveal: []byte{'A', 'B', 'C'},
			Eth1Data:     eth1Data,
		},
	}
	beaconState, err = state.ExecuteStateTransition(context.Background(), beaconState, block, state.DefaultConfig())
	if err != nil {
		t.Error(err)
	}

	if beaconState.Slot != 64 {
		t.Errorf("Unexpected Slot number, expected: 64, received: %d", beaconState.Slot)
	}

	if bytes.Equal(beaconState.RandaoMixes[1], oldMix) {
		t.Errorf("Did not expect new and old randao mix to equal, %#x == %#x", beaconState.RandaoMixes[0], oldMix)
	}

	if beaconState.StartShard == oldStartShard {
		t.Errorf("Did not expect new and old start shard to equal, %#x == %#x", beaconState.StartShard, oldStartShard)
	}
}

func TestProcessBlock_IncorrectProposerSlashing(t *testing.T) {
	helpers.ClearAllCaches()
	deposits, privKeys := testutil.SetupInitialDeposits(t, 100, true)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), nil)
	if err != nil {
		t.Fatal(err)
	}
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestBlockHeader = &pb.BeaconBlockHeader{
		Slot:       genesisBlock.Slot,
		ParentRoot: genesisBlock.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	parentRoot, err := ssz.SigningRoot(beaconState.LatestBlockHeader)
	if err != nil {
		t.Fatal(err)
	}
	slashing := &pb.ProposerSlashing{
		Header_1: &pb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch},
		Header_2: &pb.BeaconBlockHeader{Slot: params.BeaconConfig().SlotsPerEpoch * 2},
	}

	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := helpers.CreateRandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	blkDeposits := make([]*pb.Deposit, 0)
	block := &pb.BeaconBlock{
		ParentRoot: parentRoot[:],
		Slot:       0,
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      randaoReveal,
			ProposerSlashings: []*pb.ProposerSlashing{slashing},
			Eth1Data: &pb.Eth1Data{
				DepositRoot: []byte{2},
				BlockHash:   []byte{3},
			},
			Deposits: blkDeposits,
		},
	}
	want := "could not process block proposer slashing"
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProcessBlockAttestations(t *testing.T) {
	deposits, privKeys := testutil.SetupInitialDeposits(t, 100, true)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), nil)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slashings = make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)
	proposerSlashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 3,
			Header_1: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("B"),
			},
		},
	}
	attesterSlashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 0},
					Target: &pb.Checkpoint{Epoch: 0},
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 1},
					Target: &pb.Checkpoint{Epoch: 0},
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
		},
	}

	attestation := &pb.Attestation{
		Data: &pb.AttestationData{
			Target:    &pb.Checkpoint{Epoch: 0},
			Crosslink: &pb.Crosslink{},
		},
		AggregationBits: bitfield.NewBitlist(0),
		CustodyBits:     bitfield.NewBitlist(0),
	}
	epoch := helpers.CurrentEpoch(beaconState)
	randaoReveal, err := helpers.CreateRandaoReveal(beaconState, epoch, privKeys)
	if err != nil {
		t.Fatal(err)
	}
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestBlockHeader = &pb.BeaconBlockHeader{
		Slot:       genesisBlock.Slot,
		ParentRoot: genesisBlock.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	parentRoot, err := ssz.SigningRoot(beaconState.LatestBlockHeader)
	if err != nil {
		t.Fatal(err)
	}
	block := &pb.BeaconBlock{
		ParentRoot: parentRoot[:],
		Slot:       0,
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      randaoReveal,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      []*pb.Attestation{attestation},
			Eth1Data: &pb.Eth1Data{
				DepositRoot: []byte{2},
				BlockHash:   []byte{3},
			},
			Deposits: make([]*pb.Deposit, 0),
		},
	}
	want := "could not process block attestations"
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessBlock_IncorrectProcessExits(t *testing.T) {
	helpers.ClearAllCaches()

	deposits, _ := testutil.SetupInitialDeposits(t, params.BeaconConfig().MinGenesisActiveValidatorCount/8, false)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), nil)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.Slashings = make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)
	proposerSlashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 3,
			Header_1: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("B"),
			},
		},
	}
	attesterSlashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 0},
					Target: &pb.Checkpoint{Epoch: 0},
					Crosslink: &pb.Crosslink{
						Shard: 4,
					}},
				CustodyBit_0Indices: []uint64{0, 1},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 1},
					Target: &pb.Checkpoint{Epoch: 0},
					Crosslink: &pb.Crosslink{
						Shard: 4,
					}},
				CustodyBit_0Indices: []uint64{0, 1},
			},
		},
	}
	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().HistoricalRootsLimit; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	beaconState.BlockRoots = blockRoots
	beaconState.CurrentCrosslinks = []*pb.Crosslink{
		{
			DataRoot: []byte{1},
		},
	}
	blockAtt := &pb.Attestation{
		Data: &pb.AttestationData{
			Source: &pb.Checkpoint{Epoch: 0},
			Target: &pb.Checkpoint{Epoch: 0, Root: []byte("hello-world")},
			Crosslink: &pb.Crosslink{
				Shard:      0,
				StartEpoch: 0,
			},
		},
		AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
		CustodyBits:     bitfield.Bitlist{0x00, 0x00, 0x00, 0x00, 0x01},
	}
	attestations := []*pb.Attestation{blockAtt}
	var exits []*pb.VoluntaryExit
	for i := uint64(0); i < params.BeaconConfig().MaxVoluntaryExits+1; i++ {
		exits = append(exits, &pb.VoluntaryExit{})
	}
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestBlockHeader = &pb.BeaconBlockHeader{
		Slot:       genesisBlock.Slot,
		ParentRoot: genesisBlock.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	parentRoot, err := ssz.SigningRoot(beaconState.LatestBlockHeader)
	if err != nil {
		t.Fatal(err)
	}
	block := &pb.BeaconBlock{
		ParentRoot: parentRoot[:],
		Slot:       1,
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      []byte{},
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			VoluntaryExits:    exits,
			Eth1Data: &pb.Eth1Data{
				DepositRoot: []byte{2},
				BlockHash:   []byte{3},
			},
		},
	}
	beaconState.Slot += params.BeaconConfig().MinAttestationInclusionDelay
	beaconState.CurrentCrosslinks = []*pb.Crosslink{
		{
			Shard:      0,
			StartEpoch: 0,
		},
	}
	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}

	encoded, err := ssz.HashTreeRoot(beaconState.CurrentCrosslinks[0])
	if err != nil {
		t.Fatal(err)
	}
	block.Body.Attestations[0].Data.Crosslink.ParentRoot = encoded[:]
	block.Body.Attestations[0].Data.Crosslink.DataRoot = params.BeaconConfig().ZeroHash[:]
	if _, err := state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig()); err == nil {
		t.Error("Expected err, received nil")
	}
}

func TestProcessBlock_PassesProcessingConditions(t *testing.T) {
	deposits, _ := testutil.SetupInitialDeposits(t, params.BeaconConfig().MinGenesisActiveValidatorCount/8, false)
	beaconState, err := state.GenesisBeaconState(deposits, uint64(0), nil)
	if err != nil {
		t.Fatal(err)
	}
	genesisBlock := blocks.NewGenesisBlock([]byte{})
	bodyRoot, err := ssz.HashTreeRoot(genesisBlock)
	if err != nil {
		t.Fatal(err)
	}
	beaconState.LatestBlockHeader = &pb.BeaconBlockHeader{
		Slot:       genesisBlock.Slot,
		ParentRoot: genesisBlock.ParentRoot,
		BodyRoot:   bodyRoot[:],
	}
	beaconState.Slashings = make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector)
	proposerSlashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 3,
			Header_1: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      1,
				Signature: []byte("B"),
			},
		},
	}
	attesterSlashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 0, Root: []byte{'A'}},
					Target: &pb.Checkpoint{Epoch: 0},
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Source: &pb.Checkpoint{Epoch: 0, Root: []byte{'B'}},
					Target: &pb.Checkpoint{Epoch: 0},
					Crosslink: &pb.Crosslink{
						Shard: 4,
					},
				},
				CustodyBit_0Indices: []uint64{0, 1},
			},
		},
	}
	var blockRoots [][]byte
	for i := uint64(0); i < params.BeaconConfig().HistoricalRootsLimit; i++ {
		blockRoots = append(blockRoots, []byte{byte(i)})
	}
	beaconState.BlockRoots = blockRoots
	beaconState.CurrentCrosslinks = []*pb.Crosslink{
		{
			DataRoot: []byte{1},
		},
	}
	slotsPerEpoch := params.BeaconConfig().SlotsPerEpoch
	beaconState.Slot = (params.BeaconConfig().PersistentCommitteePeriod * slotsPerEpoch) + params.BeaconConfig().MinAttestationInclusionDelay
	blockAtt := &pb.Attestation{
		Data: &pb.AttestationData{
			Target: &pb.Checkpoint{Epoch: helpers.SlotToEpoch(beaconState.Slot)},
			Source: &pb.Checkpoint{
				Epoch: 0,
				Root:  []byte("hello-world"),
			},
			Crosslink: &pb.Crosslink{
				Shard:    0,
				EndEpoch: 64,
			},
		},
		AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0x01},
		CustodyBits:     bitfield.Bitlist{0x00, 0x00, 0x00, 0x00, 0x01},
	}
	attestations := []*pb.Attestation{blockAtt}
	exits := []*pb.VoluntaryExit{
		{
			ValidatorIndex: 10,
			Epoch:          0,
		},
	}
	parentRoot, err := ssz.SigningRoot(beaconState.LatestBlockHeader)
	if err != nil {
		t.Fatal(err)
	}
	block := &pb.BeaconBlock{
		ParentRoot: parentRoot[:],
		Slot:       beaconState.Slot,
		Body: &pb.BeaconBlockBody{
			RandaoReveal:      []byte{},
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Attestations:      attestations,
			VoluntaryExits:    exits,
			Eth1Data: &pb.Eth1Data{
				DepositRoot: []byte{2},
				BlockHash:   []byte{3},
			},
		},
	}
	beaconState.CurrentCrosslinks = []*pb.Crosslink{
		{
			Shard:      0,
			StartEpoch: helpers.SlotToEpoch(beaconState.Slot),
		},
	}
	beaconState.CurrentJustifiedCheckpoint.Root = []byte("hello-world")
	beaconState.CurrentEpochAttestations = []*pb.PendingAttestation{}
	encoded, err := ssz.HashTreeRoot(beaconState.CurrentCrosslinks[0])
	if err != nil {
		t.Fatal(err)
	}
	block.Body.Attestations[0].Data.Crosslink.ParentRoot = encoded[:]
	block.Body.Attestations[0].Data.Crosslink.DataRoot = params.BeaconConfig().ZeroHash[:]
	beaconState, err = state.ProcessBlock(context.Background(), beaconState, block, state.DefaultConfig())
	if err != nil {
		t.Errorf("Expected block to pass processing conditions: %v", err)
	}

	if !beaconState.Validators[proposerSlashings[0].ProposerIndex].Slashed {
		t.Errorf("Expected validator at index %d to be slashed, received false", proposerSlashings[0].ProposerIndex)
	}

	if !beaconState.Validators[1].Slashed {
		t.Error("Expected validator at index 1 to be slashed, received false")
	}

	received := beaconState.Validators[exits[0].ValidatorIndex].ExitEpoch
	wanted := params.BeaconConfig().FarFutureEpoch
	if received == wanted {
		t.Errorf("Expected validator at index %d to be exiting, did not expect: %d", exits[0].ValidatorIndex, wanted)
	}
}

func TestProcessEpoch_CantGetTgtAttsPrevEpoch(t *testing.T) {
	atts := []*pb.PendingAttestation{{Data: &pb.AttestationData{Target: &pb.Checkpoint{Epoch: 1}}}}
	_, err := state.ProcessEpoch(context.Background(), &pb.BeaconState{CurrentEpochAttestations: atts})
	if !strings.Contains(err.Error(), "could not get target atts prev epoch") {
		t.Fatal("Did not receive wanted error")
	}
}

func TestProcessEpoch_CantGetTgtAttsCurrEpoch(t *testing.T) {
	epoch := uint64(1)

	atts := []*pb.PendingAttestation{{Data: &pb.AttestationData{Crosslink: &pb.Crosslink{Shard: 100}}}}
	_, err := state.ProcessEpoch(context.Background(), &pb.BeaconState{
		Slot:                     epoch * params.BeaconConfig().SlotsPerEpoch,
		BlockRoots:               make([][]byte, 128),
		RandaoMixes:              make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:         make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CurrentEpochAttestations: atts})
	if !strings.Contains(err.Error(), "could not get target atts current epoch") {
		t.Fatal("Did not receive wanted error")
	}
}

func TestProcessEpoch_CanProcess(t *testing.T) {
	helpers.ClearAllCaches()
	epoch := uint64(1)

	atts := []*pb.PendingAttestation{{Data: &pb.AttestationData{Crosslink: &pb.Crosslink{Shard: 0}, Target: &pb.Checkpoint{}}}}
	var crosslinks []*pb.Crosslink
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		crosslinks = append(crosslinks, &pb.Crosslink{
			StartEpoch: 0,
			DataRoot:   []byte{'A'},
		})
	}
	newState, err := state.ProcessEpoch(context.Background(), &pb.BeaconState{
		Slot:                       epoch*params.BeaconConfig().SlotsPerEpoch + 1,
		BlockRoots:                 make([][]byte, 128),
		Slashings:                  []uint64{0, 1e9, 1e9},
		RandaoMixes:                make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:           make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CompactCommitteesRoots:     make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CurrentCrosslinks:          crosslinks,
		CurrentEpochAttestations:   atts,
		FinalizedCheckpoint:        &pb.Checkpoint{},
		JustificationBits:          bitfield.Bitvector4{0x00},
		CurrentJustifiedCheckpoint: &pb.Checkpoint{},
	})
	if err != nil {
		t.Fatal(err)
	}

	wanted := uint64(0)
	if newState.Slashings[2] != wanted {
		t.Errorf("Wanted slashed balance: %d, got: %d", wanted, newState.Slashings[2])
	}
}

func TestProcessEpoch_NotPanicOnEmptyActiveValidatorIndices(t *testing.T) {
	newState := &pb.BeaconState{
		ActiveIndexRoots: make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		Slashings:        make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		RandaoMixes:      make([][]byte, params.BeaconConfig().SlotsPerEpoch),
	}

	state.ProcessEpoch(context.Background(), newState)
}

func BenchmarkProcessEpoch65536Validators(b *testing.B) {
	logrus.SetLevel(logrus.PanicLevel)

	helpers.ClearAllCaches()
	epoch := uint64(1)

	validatorCount := params.BeaconConfig().MinGenesisActiveValidatorCount * 4
	shardCount := validatorCount / params.BeaconConfig().TargetCommitteeSize
	validators := make([]*pb.Validator, validatorCount)
	balances := make([]uint64, validatorCount)

	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	var atts []*pb.PendingAttestation
	for i := uint64(0); i < shardCount; i++ {
		atts = append(atts, &pb.PendingAttestation{
			Data: &pb.AttestationData{
				Crosslink: &pb.Crosslink{
					Shard: i,
				},
			},
			AggregationBits: []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			InclusionDelay: 1,
		})
	}

	var crosslinks []*pb.Crosslink
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		crosslinks = append(crosslinks, &pb.Crosslink{
			StartEpoch: 0,
			DataRoot:   []byte{'A'},
		})
	}

	s := &pb.BeaconState{
		Slot:                      epoch*params.BeaconConfig().SlotsPerEpoch + 1,
		Validators:                validators,
		Balances:                  balances,
		StartShard:                512,
		FinalizedCheckpoint:       &pb.Checkpoint{},
		BlockRoots:                make([][]byte, 254),
		Slashings:                 []uint64{0, 1e9, 0},
		RandaoMixes:               make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		ActiveIndexRoots:          make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CurrentCrosslinks:         crosslinks,
		PreviousEpochAttestations: atts,
	}

	// Precache the shuffled indices
	for i := uint64(0); i < shardCount; i++ {
		if _, err := helpers.CrosslinkCommittee(s, 0, i); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := state.ProcessEpoch(context.Background(), s)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessBlk_65536Validators_FullBlock(b *testing.B) {
	logrus.SetLevel(logrus.PanicLevel)
	helpers.ClearAllCaches()
	testConfig := params.BeaconConfig()
	testConfig.MaxTransfers = 1

	validatorCount := params.BeaconConfig().MinGenesisActiveValidatorCount * 4
	shardCount := validatorCount / params.BeaconConfig().TargetCommitteeSize
	validators := make([]*pb.Validator, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.Validator{
			EffectiveBalance:           params.BeaconConfig().MaxEffectiveBalance,
			ExitEpoch:                  params.BeaconConfig().FarFutureEpoch,
			WithdrawableEpoch:          params.BeaconConfig().FarFutureEpoch,
			ActivationEligibilityEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}
	validatorBalances := make([]uint64, len(validators))
	for i := 0; i < len(validatorBalances); i++ {
		validatorBalances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	randaoMixes := make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector)
	for i := 0; i < len(randaoMixes); i++ {
		randaoMixes[i] = params.BeaconConfig().ZeroHash[:]
	}

	var crosslinks []*pb.Crosslink
	for i := uint64(0); i < params.BeaconConfig().ShardCount; i++ {
		crosslinks = append(crosslinks, &pb.Crosslink{
			StartEpoch: 0,
			DataRoot:   []byte{'A'},
		})
	}

	s := &pb.BeaconState{
		Slot:              20,
		LatestBlockHeader: &pb.BeaconBlockHeader{},
		BlockRoots:        make([][]byte, 254),
		RandaoMixes:       randaoMixes,
		Validators:        validators,
		Balances:          validatorBalances,
		Slashings:         make([]uint64, params.BeaconConfig().EpochsPerSlashingsVector),
		ActiveIndexRoots:  make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CurrentJustifiedCheckpoint: &pb.Checkpoint{
			Root: []byte("hello-world"),
		},
		Fork: &pb.Fork{
			PreviousVersion: []byte{0, 0, 0, 0},
			CurrentVersion:  []byte{0, 0, 0, 0},
		},
		CurrentCrosslinks: crosslinks,
	}

	c := &state.TransitionConfig{
		VerifySignatures: true,
	}

	// Set up proposer slashing object for block
	proposerSlashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 1,
			Header_1: &pb.BeaconBlockHeader{
				Slot:      0,
				Signature: []byte("A"),
			},
			Header_2: &pb.BeaconBlockHeader{
				Slot:      0,
				Signature: []byte("B"),
			},
		},
	}

	// Set up attester slashing object for block
	attesterSlashings := []*pb.AttesterSlashing{
		{
			Attestation_1: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Crosslink: &pb.Crosslink{
						Shard: 5,
					},
				},
				CustodyBit_0Indices: []uint64{2, 3},
			},
			Attestation_2: &pb.IndexedAttestation{
				Data: &pb.AttestationData{
					Crosslink: &pb.Crosslink{
						Shard: 5,
					},
				},
				CustodyBit_0Indices: []uint64{2, 3},
			},
		},
	}

	// Set up deposit object for block
	deposit := &pb.Deposit{
		Data: &pb.DepositData{
			Pubkey: []byte{1, 2, 3},
			Amount: params.BeaconConfig().MaxEffectiveBalance,
		},
	}
	leaf, err := ssz.HashTreeRoot(deposit.Data)
	if err != nil {
		b.Fatal(err)
	}
	depositTrie, err := trieutil.GenerateTrieFromItems([][]byte{leaf[:]}, int(params.BeaconConfig().DepositContractTreeDepth))
	if err != nil {
		b.Fatalf("Could not generate trie: %v", err)
	}
	proof, err := depositTrie.MerkleProof(0)
	if err != nil {
		b.Fatalf("Could not generate proof: %v", err)
	}
	deposit.Proof = proof
	root := depositTrie.Root()

	// Set up randao reveal object for block
	proposerIdx, err := helpers.BeaconProposerIndex(s)
	if err != nil {
		b.Fatal(err)
	}
	priv, err := bls.RandKey(rand.Reader)
	if err != nil {
		b.Fatal(err)
	}
	s.Validators[proposerIdx].Pubkey = priv.PublicKey().Marshal()
	buf := make([]byte, 32)
	binary.LittleEndian.PutUint64(buf, 0)
	domain := helpers.Domain(s, 0, params.BeaconConfig().DomainRandao)
	epochSignature := priv.Sign(buf, domain)

	// Set up transfer object for block
	transfers := []*pb.Transfer{
		{
			Slot:      s.Slot,
			Sender:    3,
			Recipient: 4,
			Fee:       params.BeaconConfig().MinDepositAmount,
			Amount:    params.BeaconConfig().MinDepositAmount,
			Pubkey:    []byte("A"),
		},
	}
	buf = []byte{params.BeaconConfig().BLSWithdrawalPrefixByte}
	pubKey := []byte("A")
	hashed := hashutil.Hash(pubKey)
	buf = append(buf, hashed[:]...)
	s.Validators[3].WithdrawalCredentials = buf

	// Set up attestations obj for block.
	encoded, err := ssz.HashTreeRoot(s.CurrentCrosslinks[0])
	if err != nil {
		b.Fatal(err)
	}

	attestations := make([]*pb.Attestation, 128)
	for i := 0; i < len(attestations); i++ {
		attestations[i] = &pb.Attestation{
			Data: &pb.AttestationData{
				Source: &pb.Checkpoint{Root: []byte("hello-world")},
				Crosslink: &pb.Crosslink{
					Shard:      uint64(i),
					ParentRoot: encoded[:],
					DataRoot:   params.BeaconConfig().ZeroHash[:],
				},
			},
			AggregationBits: bitfield.Bitlist{0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0,
				0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0xC0, 0x01},
			CustodyBits: bitfield.NewBitlist(0),
		}
	}

	blk := &pb.BeaconBlock{
		Slot: s.Slot,
		Body: &pb.BeaconBlockBody{
			Eth1Data: &pb.Eth1Data{
				DepositRoot: root[:],
				BlockHash:   root[:],
			},
			RandaoReveal:      epochSignature.Marshal(),
			Attestations:      attestations,
			ProposerSlashings: proposerSlashings,
			AttesterSlashings: attesterSlashings,
			Transfers:         transfers,
		},
	}

	// Precache the shuffled indices
	for i := uint64(0); i < shardCount; i++ {
		if _, err := helpers.CrosslinkCommittee(s, 0, i); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := state.ProcessBlock(context.Background(), s, blk, c)
		if err != nil {
			b.Fatal(err)
		}
		// Reset state fields to process block again
		s.Validators[1].Slashed = false
		s.Validators[2].Slashed = false
		s.Balances[3] += 2 * params.BeaconConfig().MinDepositAmount
	}
}

func TestCanProcessEpoch_TrueOnEpochs(t *testing.T) {
	if params.BeaconConfig().SlotsPerEpoch != 64 {
		t.Errorf("SlotsPerEpoch should be 64 for these tests to pass")
	}

	tests := []struct {
		slot            uint64
		canProcessEpoch bool
	}{
		{
			slot:            1,
			canProcessEpoch: false,
		}, {
			slot:            63,
			canProcessEpoch: true,
		},
		{
			slot:            64,
			canProcessEpoch: false,
		}, {
			slot:            127,
			canProcessEpoch: true,
		}, {
			slot:            1000000000,
			canProcessEpoch: false,
		},
	}

	for _, tt := range tests {
		s := &pb.BeaconState{Slot: tt.slot}
		if state.CanProcessEpoch(s) != tt.canProcessEpoch {
			t.Errorf(
				"CanProcessEpoch(%d) = %v. Wanted %v",
				tt.slot,
				state.CanProcessEpoch(s),
				tt.canProcessEpoch,
			)
		}
	}
}

func TestProcessOperations_OverMaxProposerSlashings(t *testing.T) {
	maxSlashings := params.BeaconConfig().MaxProposerSlashings
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			ProposerSlashings: make([]*pb.ProposerSlashing, maxSlashings+1),
		},
	}

	want := fmt.Sprintf("number of proposer slashings (%d) in block body exceeds allowed threshold of %d",
		len(block.Body.ProposerSlashings), params.BeaconConfig().MaxProposerSlashings)
	if _, err := state.ProcessOperations(
		context.Background(),
		&pb.BeaconState{},
		block.Body,
		state.DefaultConfig(),
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperations_OverMaxAttesterSlashings(t *testing.T) {
	maxSlashings := params.BeaconConfig().MaxAttesterSlashings
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			AttesterSlashings: make([]*pb.AttesterSlashing, maxSlashings+1),
		},
	}

	want := fmt.Sprintf("number of attester slashings (%d) in block body exceeds allowed threshold of %d",
		len(block.Body.AttesterSlashings), params.BeaconConfig().MaxAttesterSlashings)
	if _, err := state.ProcessOperations(
		context.Background(),
		&pb.BeaconState{},
		block.Body,
		state.DefaultConfig(),
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperations_OverMaxAttestations(t *testing.T) {
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Attestations: make([]*pb.Attestation, params.BeaconConfig().MaxAttestations+1),
		},
	}

	want := fmt.Sprintf("number of attestations (%d) in block body exceeds allowed threshold of %d",
		len(block.Body.Attestations), params.BeaconConfig().MaxAttestations)
	if _, err := state.ProcessOperations(
		context.Background(),
		&pb.BeaconState{},
		block.Body,
		state.DefaultConfig(),
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperations_OverMaxTransfers(t *testing.T) {
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Transfers: make([]*pb.Transfer, params.BeaconConfig().MaxTransfers+1),
		},
	}

	want := fmt.Sprintf("number of transfers (%d) in block body exceeds allowed threshold of %d",
		len(block.Body.Transfers), params.BeaconConfig().MaxTransfers)
	if _, err := state.ProcessOperations(
		context.Background(),
		&pb.BeaconState{},
		block.Body,
		state.DefaultConfig(),
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperation_OverMaxVoluntaryExits(t *testing.T) {
	maxExits := params.BeaconConfig().MaxVoluntaryExits
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			VoluntaryExits: make([]*pb.VoluntaryExit, maxExits+1),
		},
	}

	want := fmt.Sprintf("number of voluntary exits (%d) in block body exceeds allowed threshold of %d",
		len(block.Body.VoluntaryExits), maxExits)
	if _, err := state.ProcessOperations(
		context.Background(),
		&pb.BeaconState{},
		block.Body,
		state.DefaultConfig(),
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperations_IncorrectDeposits(t *testing.T) {
	s := &pb.BeaconState{
		Eth1Data:         &pb.Eth1Data{DepositCount: 100},
		Eth1DepositIndex: 98,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Deposits: []*pb.Deposit{{}},
		},
	}

	want := fmt.Sprintf("incorrect outstanding deposits in block body, wanted: %d, got: %d",
		s.Eth1Data.DepositCount-s.Eth1DepositIndex, len(block.Body.Deposits))
	if _, err := state.ProcessOperations(
		context.Background(),
		s,
		block.Body,
		state.DefaultConfig(),
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}

func TestProcessOperation_DuplicateTransfer(t *testing.T) {
	testConfig := params.BeaconConfig()
	testConfig.MaxTransfers = 2
	transfers := []*pb.Transfer{
		{
			Amount: 1,
		},
		{
			Amount: 1,
		},
	}
	registry := []*pb.Validator{}
	s := &pb.BeaconState{
		Validators:       registry,
		Eth1Data:         &pb.Eth1Data{DepositCount: 100},
		Eth1DepositIndex: 98,
	}
	block := &pb.BeaconBlock{
		Body: &pb.BeaconBlockBody{
			Transfers: transfers,
			Deposits:  []*pb.Deposit{{}, {}},
		},
	}

	want := "duplicate transfer"
	if _, err := state.ProcessOperations(
		context.Background(),
		s,
		block.Body,
		state.DefaultConfig(),
	); !strings.Contains(err.Error(), want) {
		t.Errorf("Expected %s, received %v", want, err)
	}
}
