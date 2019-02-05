package validators

import (
	"github.com/prysmaticlabs/prysm/shared/params"
	"reflect"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestAttestationParticipants_ok(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	validators := make([]*pb.ValidatorRecord, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}

	attestationData := &pb.AttestationData{}

	tests := []struct {
		attestationSlot uint64
		stateSlot       uint64
		shard           uint64
		bitfield        []byte
		wanted          []uint64
	}{
		{
			attestationSlot: 2,
			stateSlot:       5,
			shard:           256,
			bitfield:        []byte{0xFF},
			wanted:          []uint64{766, 752},
		},
		{
			attestationSlot: 1,
			stateSlot:       10,
			shard:           128,
			bitfield:        []byte{77},
			wanted:          []uint64{511},
		},
		{
			attestationSlot: 10,
			stateSlot:       20,
			shard:           383,
			bitfield:        []byte{0xFF},
			wanted:          []uint64{3069, 2608},
		},
		{
			attestationSlot: 64,
			stateSlot:       100,
			shard:           0,
			bitfield:        []byte{0xFF},
			wanted:          []uint64{237, 224},
		},
		{
			attestationSlot: 999,
			stateSlot:       1000,
			shard:           1023,
			bitfield:        []byte{99},
			wanted:          []uint64{10494},
		},
	}

	for _, tt := range tests {
		state.Slot = tt.stateSlot
		attestationData.Slot = tt.attestationSlot
		attestationData.Shard = tt.shard

		result, err := AttestationParticipants(state, attestationData, tt.bitfield)
		if err != nil {
			t.Errorf("Failed to get attestation participants: %v", err)
		}

		if !reflect.DeepEqual(tt.wanted, result) {
			t.Errorf(
				"Result indices was an unexpected value. Wanted %d, got %d",
				tt.wanted,
				result,
			)
		}
	}
}

func TestAttestationParticipants_IncorrectBitfield(t *testing.T) {
	if params.BeaconConfig().EpochLength != 64 {
		t.Errorf("EpochLength should be 64 for these tests to pass")
	}

	validators := make([]*pb.ValidatorRecord, params.BeaconConfig().DepositsForChainStart)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitEpoch: params.BeaconConfig().FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}
	attestationData := &pb.AttestationData{}

	if _, err := AttestationParticipants(state, attestationData, []byte{}); err == nil {
		t.Error("attestation participants should have failed with incorrect bitfield")
	}
}
