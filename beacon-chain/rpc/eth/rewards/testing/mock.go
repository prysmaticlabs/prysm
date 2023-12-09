package testing

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/rewards"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	"github.com/prysmaticlabs/prysm/v4/network/httputil"
)

type MockBlockRewardFetcher struct {
	Rewards *rewards.BlockRewards
	Error   *httputil.DefaultErrorJson
	State   state.BeaconState
}

func (m *MockBlockRewardFetcher) GetBlockRewardsData(_ context.Context, _ interfaces.ReadOnlySignedBeaconBlock) (*rewards.BlockRewards, *httputil.DefaultErrorJson) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Rewards, nil
}

func (m *MockBlockRewardFetcher) GetStateForRewards(_ context.Context, _ interfaces.ReadOnlySignedBeaconBlock) (state.BeaconState, *httputil.DefaultErrorJson) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.State, nil
}
