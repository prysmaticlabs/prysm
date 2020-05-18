package mock

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	beaconstate "github.com/prysmaticlabs/prysm/beacon-chain/state"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

// FaultyChain defines an incorrectly functioning powchain service.
type FaultyChain struct {
	ChainFeed      *event.Feed
	HashesByHeight map[int][]byte
}

// Eth2GenesisPowchainInfo --
func (f *FaultyChain) Eth2GenesisPowchainInfo() (uint64, *big.Int) {
	return 0, big.NewInt(0)
}

// LatestBlockHeight --
func (f *FaultyChain) LatestBlockHeight() *big.Int {
	return big.NewInt(0)
}

// BlockExists --
func (f *FaultyChain) BlockExists(_ context.Context, hash common.Hash) (bool, *big.Int, error) {
	if f.HashesByHeight == nil {
		return false, big.NewInt(1), errors.New("failed")
	}

	return true, big.NewInt(1), nil
}

// BlockHashByHeight --
func (f *FaultyChain) BlockHashByHeight(_ context.Context, height *big.Int) (common.Hash, error) {
	return [32]byte{}, errors.New("failed")
}

// BlockTimeByHeight --
func (f *FaultyChain) BlockTimeByHeight(_ context.Context, height *big.Int) (uint64, error) {
	return 0, errors.New("failed")
}

// BlockNumberByTimestamp --
func (f *FaultyChain) BlockNumberByTimestamp(_ context.Context, _ uint64) (*big.Int, error) {
	return big.NewInt(0), nil
}

// DepositRoot --
func (f *FaultyChain) DepositRoot() [32]byte {
	return [32]byte{}
}

// DepositTrie --
func (f *FaultyChain) DepositTrie() *trieutil.SparseMerkleTrie {
	return &trieutil.SparseMerkleTrie{}
}

// ChainStartDeposits --
func (f *FaultyChain) ChainStartDeposits() []*ethpb.Deposit {
	return []*ethpb.Deposit{}
}

// ChainStartEth1Data --
func (f *FaultyChain) ChainStartEth1Data() *ethpb.Eth1Data {
	return &ethpb.Eth1Data{}
}

// PreGenesisState --
func (f *FaultyChain) PreGenesisState() *beaconstate.BeaconState {
	return &beaconstate.BeaconState{}
}

// ClearPreGenesisData --
func (f *FaultyChain) ClearPreGenesisData() {
	//no-op
}

// IsConnectedToETH1 --
func (f *FaultyChain) IsConnectedToETH1() bool {
	return true
}
