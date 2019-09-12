package testing

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
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

// ChainStartFeed --
func (f *FaultyMockPOWChain) ChainStartFeed() *event.Feed {
	return f.ChainFeed
}

// LatestBlockHeight --
func (f *FaultyMockPOWChain) LatestBlockHeight() *big.Int {
	return big.NewInt(0)
}

// BlockExists --
func (f *FaultyMockPOWChain) BlockExists(_ context.Context, hash common.Hash) (bool, *big.Int, error) {
	if f.HashesByHeight == nil {
		return false, big.NewInt(1), errors.New("failed")
	}

	return true, big.NewInt(1), nil
}

// BlockHashByHeight --
func (f *FaultyMockPOWChain) BlockHashByHeight(_ context.Context, height *big.Int) (common.Hash, error) {
	return [32]byte{}, errors.New("failed")
}

// BlockTimeByHeight --
func (f *FaultyMockPOWChain) BlockTimeByHeight(_ context.Context, height *big.Int) (uint64, error) {
	return 0, errors.New("failed")
}

// BlockNumberByTimestamp --
func (f *FaultyMockPOWChain) BlockNumberByTimestamp(_ context.Context, _ uint64) (*big.Int, error) {
	return big.NewInt(0), nil
}

// DepositRoot --
func (f *FaultyMockPOWChain) DepositRoot() [32]byte {
	return [32]byte{}
}

// DepositTrie --
func (f *FaultyMockPOWChain) DepositTrie() *trieutil.MerkleTrie {
	return &trieutil.MerkleTrie{}
}

// ChainStartDeposits --
func (f *FaultyMockPOWChain) ChainStartDeposits() []*ethpb.Deposit {
	return []*ethpb.Deposit{}
}

// ChainStartDepositHashes --
func (f *FaultyMockPOWChain) ChainStartDepositHashes() ([][]byte, error) {
	return [][]byte{}, errors.New("hashing failed")
}

// ChainStartEth1Data --
func (f *FaultyMockPOWChain) ChainStartEth1Data() *ethpb.Eth1Data {
	return &ethpb.Eth1Data{}
}
