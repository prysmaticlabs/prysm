package state

import (
	"fmt"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/params"
)

// applyProposerSlashing is one of the operations performed
// on each processed beacon block.
func applyProposerSlashing(proposerSlashings []*pb.ProposerSlashing) error {
	if uint64(len(proposerSlashings)) > params.BeaconConfig().MaxProposerSlashings {
		return fmt.Errorf("number of proposer slashings exceeds threshold")
	}
	return nil
}
