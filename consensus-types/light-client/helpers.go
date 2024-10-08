package light_client

import (
	"fmt"

	fieldparams "github.com/prysmaticlabs/prysm/v5/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v5/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v5/encoding/bytesutil"
)

type branchConstraint interface {
	~interfaces.LightClientExecutionBranch | ~interfaces.LightClientSyncCommitteeBranch | ~interfaces.LightClientFinalityBranch
}

func createBranch[T branchConstraint](name string, input [][]byte, depth int) (T, error) {
	var zero T

	if len(input) != depth {
		return zero, fmt.Errorf("%s branch has %d leaves instead of expected %d", name, len(input), depth)
	}
	var branch T
	for i, leaf := range input {
		if len(leaf) != fieldparams.RootLength {
			return zero, fmt.Errorf("%s branch leaf at index %d has length %d instead of expected %d", name, i, len(leaf), fieldparams.RootLength)
		}
		branch[i] = bytesutil.ToBytes32(leaf)
	}

	return branch, nil
}
