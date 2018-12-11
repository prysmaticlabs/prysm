package blocks

import (
	"bytes"

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
) []*pb.ValidatorRecord {
	if uint64(len(proposerSlashings)) > params.BeaconConfig().MaxProposerSlashings {
		log.Debugf("number of proposer slashings exceeds threshold")
		return nil
	}
	// TODO(#781): Verify BLS according to the spec.
	for _, slashing := range proposerSlashings {
		proposer := validatorRegistry[slashing.GetProposerIndex()]
		if slashing.GetProposalData_1().GetSlot() != slashing.GetProposalData_2().GetSlot() {
			log.Debugf("slashing proposal data slots do not match")
		}
		if slashing.GetProposalData_1().GetShard() != slashing.GetProposalData_2().GetShard() {
			log.Debugf("slashing proposal data shards do not match")
		}
		if !bytes.Equal(
			slashing.GetProposalData_1().GetBlockHash32(),
			slashing.GetProposalData_2().GetBlockHash32(),
		) {
			log.Debugf("slashing proposal data block hashes do not match")
		}
		if proposer.Status != uint64(params.ExitedWithPenalty) {
			validatorRegistry[slashing.GetProposerIndex()] = v.ExitValidator(proposer, currentSlot, true)
		}
	}
	return validatorRegistry
}
