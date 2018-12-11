package blocks

import (
	"bytes"
	"fmt"

	"github.com/sirupsen/logrus"

	v "github.com/prysmaticlabs/prysm/beacon-chain/core/validators"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

var log = logrus.WithField("prefix", "state")

// ProcessProposerSlashings is one of the operations performed
// on each processed beacon block to penalize proposers based on
// slashing conditions if any slashable events occurred.
func ProcessProposerSlashings(
	validatorRegistry []*pb.ValidatorRecord,
	proposerSlashings []*pb.ProposerSlashing,
	currentSlot uint64,
) ([]*pb.ValidatorRecord, error) {
	if uint64(len(proposerSlashings)) > params.BeaconConfig().MaxProposerSlashings {
		return nil, fmt.Errorf("number of proposer slashings exceeds threshold: %v", len(proposerSlashings))
	}
	// TODO(#781): Verify BLS according to the specification in the "Proposer Slashings"
	// section of block operations.
	for idx, slashing := range proposerSlashings {
		if err := verifyProposerSlashing(validatorRegistry, slashing, currentSlot); err != nil {
			return nil, fmt.Errorf("could not verify proposer slashing #%d: %v", idx, err)
		}
		proposer := validatorRegistry[slashing.GetProposerIndex()]
		if proposer.Status != uint64(params.ExitedWithPenalty) {
			validatorRegistry[slashing.GetProposerIndex()] = v.ExitValidator(proposer, currentSlot, true)
		}
	}
	return validatorRegistry, nil
}

func verifyProposerSlashing(
	validatorRegistry []*pb.ValidatorRecord,
	slashing *pb.ProposerSlashing,
	currentSlot uint64,
) error {
	slot1 := slashing.GetProposalData_1().GetSlot()
	slot2 := slashing.GetProposalData_2().GetSlot()
	shard1 := slashing.GetProposalData_1().GetShard()
	shard2 := slashing.GetProposalData_2().GetShard()
	hash1 := slashing.GetProposalData_1().GetBlockHash32()
	hash2 := slashing.GetProposalData_2().GetBlockHash32()
	if slot1 != slot2 {
		return fmt.Errorf("slashing proposal data slots do not match: %v, %v", slot1, slot2)
	}
	if shard1 != shard2 {
		return fmt.Errorf("slashing proposal data shards do not match: %v, %v", shard1, shard2)
	}
	if !bytes.Equal(hash1, hash2) {
		return fmt.Errorf("slashing proposal data block hashes do not match: %x, %x", hash1, hash2)
	}
	return nil
}
