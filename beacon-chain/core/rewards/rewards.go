package rewards

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/altair"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/core/helpers"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v3/consensus-types/interfaces"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
)

type BlockRewardsInfo struct {
	ProposerIndex     types.ValidatorIndex
	Total             uint64
	Attestations      uint64
	SyncAggregate     uint64
	ProposerSlashings uint64
	AttesterSlashings uint64
}

func BlockRewards(ctx context.Context, st state.BeaconState, b interfaces.ReadOnlySignedBeaconBlock) (*BlockRewardsInfo, error) {
	var err error
	rewards := &BlockRewardsInfo{}

	st, err = attestationsReward(ctx, st, b, rewards)
	if err != nil {
		return nil, errors.Wrap(err, "could not calculate attestations part of block reward")
	}
	st, err = processProposerSlashings(ctx, st, b.Block().Body().ProposerSlashings(), slashValidator, rewards)
	if err != nil {
		return nil, errors.Wrap(err, "could not calculate proposer slashings part of block reward")
	}
	st, err = processAttesterSlashings(ctx, st, b.Block().Body().AttesterSlashings(), slashValidator, rewards)
	if err != nil {
		return nil, errors.Wrap(err, "could not calculate attester slashings part of block reward")
	}
	totalActiveBalance, err := helpers.TotalActiveBalance(st)
	if err != nil {
		return nil, errors.Wrap(err, "could not get state's total active balance")
	}
	rewards.SyncAggregate, _, err = altair.SyncRewards(totalActiveBalance)
	if err != nil {
		return nil, errors.Wrap(err, "could not calculate sync committee part of block reward")
	}
	rewards.Total = rewards.Attestations + rewards.AttesterSlashings + rewards.ProposerSlashings + rewards.SyncAggregate
	rewards.ProposerIndex = b.Block().ProposerIndex()

	return rewards, nil
}
