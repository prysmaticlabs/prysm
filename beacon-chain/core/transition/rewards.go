package transition

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/rewards"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
)

func ProposerRewards(ctx context.Context, st state.BeaconState, b interfaces.ReadOnlySignedBeaconBlock) (*rewards.ProposerRewards, error) {
	_, _, r, err := ExecuteStateTransitionNoVerifyAnySig(ctx, st, b)
	if err != nil {
		return nil, errors.Wrap(err, "could not perform state transition")
	}
	return r, nil
}
