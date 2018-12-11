package blocks

import (
	"io/ioutil"
	"testing"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/testutil"
	"github.com/sirupsen/logrus"
	logTest "github.com/sirupsen/logrus/hooks/test"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(ioutil.Discard)
}

func TestIncorrectProcessProposerSlashings(t *testing.T) {
	hook := logTest.NewGlobal()

	// We test exceeding the proposer slashing threshold.
	slashings := make([]*pb.ProposerSlashing, params.BeaconConfig().MaxProposerSlashings+1)
	registry := []*pb.ValidatorRecord{}
	currentSlot := uint64(0)
	ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	)
	currentSlot++
	testutil.WaitForLog(t, hook, "number of proposer slashings exceeds threshold")

	// We now test the case when slots, shards, and
	// block hashes in the proposal signed data values
	// are different.
	registry = []*pb.ValidatorRecord{
		{
			Status: uint64(params.ExitedWithPenalty),
		},
	}
	slashings = []*pb.ProposerSlashing{
		{
			ProposerIndex: 0,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:        1,
				Shard:       0,
				BlockHash32: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:        0,
				Shard:       1,
				BlockHash32: []byte{1, 1, 0},
			},
		},
	}

	registry = ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	)
	testutil.WaitForLog(t, hook, "slashing proposal data slots do not match")
	testutil.WaitForLog(t, hook, "slashing proposal data shards do not match")
	testutil.WaitForLog(t, hook, "slashing proposal data block hashes do not match")
}

func TestCorrectlyProcessProposerSlashings(t *testing.T) {
	// We test the case when data is correct and verify the validator
	// registry has been updated.
	registry := []*pb.ValidatorRecord{
		{
			Status:                 uint64(params.ExitedWithPenalty),
			LatestStatusChangeSlot: 0,
		},
		{
			Status:                 uint64(params.ExitedWithPenalty),
			LatestStatusChangeSlot: 0,
		},
	}
	slashings := []*pb.ProposerSlashing{
		{
			ProposerIndex: 1,
			ProposalData_1: &pb.ProposalSignedData{
				Slot:        1,
				Shard:       1,
				BlockHash32: []byte{0, 1, 0},
			},
			ProposalData_2: &pb.ProposalSignedData{
				Slot:        1,
				Shard:       1,
				BlockHash32: []byte{0, 1, 0},
			},
		},
	}
	currentSlot := uint64(1)

	registry = ProcessProposerSlashings(
		registry,
		slashings,
		currentSlot,
	)
	if registry[1].Status != uint64(params.ExitedWithPenalty) {
		t.Errorf("Proposer with index 1 did not ExitWithPenalty in validator registry: %v", registry[1].Status)
	}
}
