package testing

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/prysmaticlabs/prysm/async/event"
	"github.com/prysmaticlabs/prysm/beacon-chain/powchain/types"
	"github.com/prysmaticlabs/prysm/beacon-chain/state"
	v1 "github.com/prysmaticlabs/prysm/beacon-chain/state/v1"
	"github.com/prysmaticlabs/prysm/container/trie"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
)

// FaultyMockPOWChain defines an incorrectly functioning powchain service.
type FaultyMockPOWChain struct {
	ChainFeed      *event.Feed
	HashesByHeight map[int][]byte
}

// Eth2GenesisPowchainInfo --
func (f *FaultyMockPOWChain) Eth2GenesisPowchainInfo() (uint64, *big.Int) {
	return 0, big.NewInt(0)
}

// LatestBlockHeight --
func (f *FaultyMockPOWChain) LatestBlockHeight() *big.Int {
	return big.NewInt(0)
}

// BlockExists --
func (f *FaultyMockPOWChain) BlockExists(_ context.Context, _ common.Hash) (bool, *big.Int, error) {
	if f.HashesByHeight == nil {
		return false, big.NewInt(1), errors.New("failed")
	}

	return true, big.NewInt(1), nil
}

// BlockHashByHeight --
func (f *FaultyMockPOWChain) BlockHashByHeight(_ context.Context, _ *big.Int) (common.Hash, error) {
	return [32]byte{}, errors.New("failed")
}

// BlockTimeByHeight --
func (f *FaultyMockPOWChain) BlockTimeByHeight(_ context.Context, _ *big.Int) (uint64, error) {
	return 0, errors.New("failed")
}

// BlockByTimestamp --
func (f *FaultyMockPOWChain) BlockByTimestamp(_ context.Context, _ uint64) (*types.HeaderInfo, error) {
	return &types.HeaderInfo{Number: big.NewInt(0)}, nil
}

// DepositRoot --
func (f *FaultyMockPOWChain) DepositRoot() [32]byte {
	return [32]byte{}
}

// DepositTrie --
func (f *FaultyMockPOWChain) DepositTrie() *trie.SparseMerkleTrie {
	return &trie.SparseMerkleTrie{}
}

// ChainStartDeposits --
func (f *FaultyMockPOWChain) ChainStartDeposits() []*ethpb.Deposit {
	return []*ethpb.Deposit{}
}

// ChainStartEth1Data --
func (f *FaultyMockPOWChain) ChainStartEth1Data() *ethpb.Eth1Data {
	return &ethpb.Eth1Data{}
}

// PreGenesisState --
func (f *FaultyMockPOWChain) PreGenesisState() state.BeaconState {
	return &v1.BeaconState{}
}

// ClearPreGenesisData --
func (f *FaultyMockPOWChain) ClearPreGenesisData() {
	// no-op
}

// IsConnectedToETH1 --
func (f *FaultyMockPOWChain) IsConnectedToETH1() bool {
	return true
}

// BlockExistsWithCache --
func (f *FaultyMockPOWChain) BlockExistsWithCache(ctx context.Context, hash common.Hash) (bool, *big.Int, error) {
	return f.BlockExists(ctx, hash)
}
