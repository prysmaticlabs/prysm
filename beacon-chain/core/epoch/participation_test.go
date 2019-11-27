package epoch_test

import (
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/prysmaticlabs/prysm/beacon-chain/core/epoch"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
)

func TestComputeValidatorParticipation_PreviousEpoch(t *testing.T) {
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	e := uint64(1)
	attestedBalance := uint64(20) * params.BeaconConfig().MaxEffectiveBalance
	validatorCount := uint64(100)

	validators := make([]*ethpb.Validator, validatorCount)
	balances := make([]uint64, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	blockRoots := make([][]byte, 256)
	for i := 0; i < len(blockRoots); i++ {
		slot := bytesutil.Bytes32(uint64(i))
		blockRoots[i] = slot
	}
	target := &ethpb.Checkpoint{
		Epoch: e,
		Root:  blockRoots[0],
	}

	atts := []*pb.PendingAttestation{
		{
			Data:            &ethpb.AttestationData{Target: target, Slot: 0},
			AggregationBits: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			Data:            &ethpb.AttestationData{Target: target, Slot: 1},
			AggregationBits: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			Data:            &ethpb.AttestationData{Target: target, Slot: 2},
			AggregationBits: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			Data:            &ethpb.AttestationData{Target: target, Slot: 3},
			AggregationBits: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			Data:            &ethpb.AttestationData{Target: target, Slot: 4},
			AggregationBits: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
	}

	s := &pb.BeaconState{
		Slot:                        e*params.BeaconConfig().SlotsPerEpoch + 1,
		Validators:                  validators,
		Balances:                    balances,
		BlockRoots:                  blockRoots,
		Slashings:                   []uint64{0, 1e9, 1e9},
		RandaoMixes:                 make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		PreviousEpochAttestations:   atts,
		FinalizedCheckpoint:         &ethpb.Checkpoint{},
		JustificationBits:           bitfield.Bitvector4{0x00},
		PreviousJustifiedCheckpoint: target,
	}

	res, err := epoch.ComputeValidatorParticipation(s, e-1)
	if err != nil {
		t.Fatal(err)
	}

	wanted := &ethpb.ValidatorParticipation{
		VotedEther:              attestedBalance,
		EligibleEther:           validatorCount * params.BeaconConfig().MaxEffectiveBalance,
		GlobalParticipationRate: float32(attestedBalance) / float32(validatorCount*params.BeaconConfig().MaxEffectiveBalance),
	}

	if !reflect.DeepEqual(res, wanted) {
		t.Errorf("Incorrect validator participation, wanted %v received %v", wanted, res)
	}
}

func TestComputeValidatorParticipation_CurrentEpoch(t *testing.T) {
	params.OverrideBeaconConfig(params.MinimalSpecConfig())
	e := uint64(1)
	attestedBalance := uint64(16) * params.BeaconConfig().MaxEffectiveBalance
	validatorCount := uint64(100)

	validators := make([]*ethpb.Validator, validatorCount)
	balances := make([]uint64, validatorCount)
	for i := 0; i < len(validators); i++ {
		validators[i] = &ethpb.Validator{
			ExitEpoch:        params.BeaconConfig().FarFutureEpoch,
			EffectiveBalance: params.BeaconConfig().MaxEffectiveBalance,
		}
		balances[i] = params.BeaconConfig().MaxEffectiveBalance
	}

	slot := e*params.BeaconConfig().SlotsPerEpoch + 4
	blockRoots := make([][]byte, 256)
	for i := 0; i < len(blockRoots); i++ {
		slot := bytesutil.Bytes32(uint64(i))
		blockRoots[i] = slot
	}
	target := &ethpb.Checkpoint{
		Epoch: e,
		Root:  blockRoots[params.BeaconConfig().SlotsPerEpoch],
	}

	atts := []*pb.PendingAttestation{
		{
			Data:            &ethpb.AttestationData{Target: target, Slot: slot - 4},
			AggregationBits: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			Data:            &ethpb.AttestationData{Target: target, Slot: slot - 3},
			AggregationBits: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			Data:            &ethpb.AttestationData{Target: target, Slot: slot - 2},
			AggregationBits: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
		{
			Data:            &ethpb.AttestationData{Target: target, Slot: slot - 1},
			AggregationBits: []byte{0xFF, 0xFF, 0xFF, 0xFF},
		},
	}

	s := &pb.BeaconState{
		Slot:                       slot,
		Validators:                 validators,
		Balances:                   balances,
		BlockRoots:                 blockRoots,
		Slashings:                  []uint64{0, 1e9, 1e9},
		RandaoMixes:                make([][]byte, params.BeaconConfig().EpochsPerHistoricalVector),
		CurrentEpochAttestations:   atts,
		FinalizedCheckpoint:        &ethpb.Checkpoint{},
		JustificationBits:          bitfield.Bitvector4{0x00},
		CurrentJustifiedCheckpoint: target,
	}

	res, err := epoch.ComputeValidatorParticipation(s, e)
	if err != nil {
		t.Fatal(err)
	}

	wanted := &ethpb.ValidatorParticipation{
		VotedEther:              attestedBalance,
		EligibleEther:           validatorCount * params.BeaconConfig().MaxEffectiveBalance,
		GlobalParticipationRate: float32(attestedBalance) / float32(validatorCount*params.BeaconConfig().MaxEffectiveBalance),
	}

	if !reflect.DeepEqual(res, wanted) {
		t.Errorf("Incorrect validator participation, wanted %v received %v", wanted, res)
	}
}
