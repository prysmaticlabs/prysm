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

// FaultymockChain defines an incorrectly functioning powchain service.
type FaultymockChain struct {
	ChainFeed      *event.Feed
	HashesByHeight map[int][]byte
}

// Eth2GenesisPowchainInfo --
func (f *FaultymockChain) Eth2GenesisPowchainInfo() (uint64, *big.Int) {
	return 0, big.NewInt(0)
}

// LatestBlockHeight --
func (f *FaultymockChain) LatestBlockHeight() *big.Int {
	return big.NewInt(0)
}

// BlockExists --
func (f *FaultymockChain) BlockExists(_ context.Context, hash common.Hash) (bool, *big.Int, error) {
	if f.HashesByHeight == nil {
		return false, big.NewInt(1), errors.New("failed")
	}

	return true, big.NewInt(1), nil
}

// BlockHashByHeight --
func (f *FaultymockChain) BlockHashByHeight(_ context.Context, height *big.Int) (common.Hash, error) {
	return [32]byte{}, errors.New("failed")
}

// BlockTimeByHeight --
func (f *FaultymockChain) BlockTimeByHeight(_ context.Context, height *big.Int) (uint64, error) {
	return 0, errors.New("failed")
}

// BlockNumberByTimestamp --
func (f *FaultymockChain) BlockNumberByTimestamp(_ context.Context, _ uint64) (*big.Int, error) {
	return big.NewInt(0), nil
}

// DepositRoot --
func (f *FaultymockChain) DepositRoot() [32]byte {
	return [32]byte{}
}

// DepositTrie --
func (f *FaultymockChain) DepositTrie() *trieutil.SparseMerkleTrie {
	return &trieutil.SparseMerkleTrie{}
}

// ChainStartDeposits --
func (f *FaultymockChain) ChainStartDeposits() []*ethpb.Deposit {
	return []*ethpb.Deposit{}
}

// ChainStartEth1Data --
func (f *FaultymockChain) ChainStartEth1Data() *ethpb.Eth1Data {
	return &ethpb.Eth1Data{}
}

// PreGenesisState --
func (f *FaultymockChain) PreGenesisState() *beaconstate.BeaconState {
	return &beaconstate.BeaconState{}
}

// ClearPreGenesisData --
func (f *FaultymockChain) ClearPreGenesisData() {
	//no-op
}

// IsConnectedToETH1 --
func (f *FaultymockChain) IsConnectedToETH1() bool {
	return true
}
