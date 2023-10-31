package testing

import (
	"context"

	"github.com/prysmaticlabs/prysm/v4/beacon-chain/rpc/eth/rewards"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/v4/consensus-types/interfaces"
	http2 "github.com/prysmaticlabs/prysm/v4/network/http"
)

type MockBlockRewardFetcher struct {
	Rewards *rewards.BlockRewards
	Error   *http2.DefaultErrorJson
	State   state.BeaconState
}

func (m *MockBlockRewardFetcher) GetBlockRewardsData(_ context.Context, _ interfaces.ReadOnlySignedBeaconBlock) (*rewards.BlockRewards, *http2.DefaultErrorJson) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Rewards, nil
}

func (m *MockBlockRewardFetcher) GetStateForRewards(_ context.Context, _ interfaces.ReadOnlySignedBeaconBlock) (state.BeaconState, *http2.DefaultErrorJson) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.State, nil
}
