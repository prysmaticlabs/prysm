package helpers

import (
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

func TestEpochCommitteeCount_Ok(t *testing.T) {
	// this defines the # of validators required to have 1 committee
	// per slot for epoch length.
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	tests := []struct {
		validatorCount uint64
		committeeCount uint64
	}{
		{0, config.EpochLength},
		{1000, config.EpochLength},
		{2 * validatorsPerEpoch, 2 * config.EpochLength},
		{5 * validatorsPerEpoch, 5 * config.EpochLength},
		{16 * validatorsPerEpoch, 16 * config.EpochLength},
		{32 * validatorsPerEpoch, 16 * config.EpochLength},
	}
	for _, test := range tests {
		if test.committeeCount != EpochCommitteeCount(test.validatorCount) {
			t.Errorf("wanted: %d, got: %d",
				test.committeeCount, EpochCommitteeCount(test.validatorCount))
		}
	}
}
func TestCurrentEpochCommitteeCount_Ok(t *testing.T) {
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	committeesPerEpoch := uint64(8)
	// set curr epoch total validators count to 8 committees per slot.
	validators := make([]*pb.ValidatorRecord, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitEpoch: config.FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}

	if CurrentEpochCommitteeCount(state) != committeesPerEpoch*config.EpochLength {
		t.Errorf("Incorrect current epoch committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, CurrentEpochCommitteeCount(state))
	}
}

func TestPrevEpochCommitteeCount_Ok(t *testing.T) {
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	committeesPerEpoch := uint64(3)
	// set prev epoch total validators count to 3 committees per slot.
	validators := make([]*pb.ValidatorRecord, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitEpoch: config.FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}

	if PrevEpochCommitteeCount(state) != committeesPerEpoch*config.EpochLength {
		t.Errorf("Incorrect prev epoch committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, PrevEpochCommitteeCount(state))
	}
}

func TestNextEpochCommitteeCount_Ok(t *testing.T) {
	validatorsPerEpoch := config.EpochLength * config.TargetCommitteeSize
	committeesPerEpoch := uint64(6)
	// set prev epoch total validators count to 3 committees per slot.
	validators := make([]*pb.ValidatorRecord, committeesPerEpoch*validatorsPerEpoch)
	for i := 0; i < len(validators); i++ {
		validators[i] = &pb.ValidatorRecord{
			ExitEpoch: config.FarFutureEpoch,
		}
	}

	state := &pb.BeaconState{
		ValidatorRegistry: validators,
	}
	if NextEpochCommitteeCount(state) != committeesPerEpoch*config.EpochLength {
		t.Errorf("Incorrect next epoch committee count per slot. Wanted: %d, got: %d",
			committeesPerEpoch, NextEpochCommitteeCount(state))
	}
}
