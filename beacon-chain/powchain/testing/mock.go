package testing

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
)

// POWChain defines a properly functioning mock for the powchain service.
type POWChain struct {
	ChainFeed           *event.Feed
	LatestBlockNumber   *big.Int
	HashesByHeight      map[int][]byte
	TimesByHeight       map[int]uint64
	BlockNumberByHeight map[uint64]*big.Int
	Eth1Data            *ethpb.Eth1Data
	GenesisEth1Block    *big.Int
}

// ChainStartFeed --
func (m *POWChain) ChainStartFeed() *event.Feed {
	return m.ChainFeed
}

// Eth2GenesisPowchainInfo --
func (m *POWChain) Eth2GenesisPowchainInfo() (uint64, *big.Int) {
	blk := m.GenesisEth1Block
	if blk == nil {
		blk = big.NewInt(0)
	}
	return uint64(time.Unix(0, 0).Unix()), blk
}

// DepositTrie --
func (m *POWChain) DepositTrie() *trieutil.MerkleTrie {
	return &trieutil.MerkleTrie{}
}

// BlockExists --
func (m *POWChain) BlockExists(_ context.Context, hash common.Hash) (bool, *big.Int, error) {
	// Reverse the map of heights by hash.
	heightsByHash := make(map[[32]byte]int)
	for k, v := range m.HashesByHeight {
		h := bytesutil.ToBytes32(v)
		heightsByHash[h] = k
	}
	val, ok := heightsByHash[hash]
	if !ok {
		return false, nil, fmt.Errorf("could not fetch height for hash: %#x", hash)
	}
	return true, big.NewInt(int64(val)), nil
}

// BlockHashByHeight --
func (m *POWChain) BlockHashByHeight(_ context.Context, height *big.Int) (common.Hash, error) {
	k := int(height.Int64())
	val, ok := m.HashesByHeight[k]
	if !ok {
		return [32]byte{}, fmt.Errorf("could not fetch hash for height: %v", height)
	}
	return bytesutil.ToBytes32(val), nil
}

// BlockTimeByHeight --
func (m *POWChain) BlockTimeByHeight(_ context.Context, height *big.Int) (uint64, error) {
	h := int(height.Int64())
	return m.TimesByHeight[h], nil
}

// BlockNumberByTimestamp --
func (m *POWChain) BlockNumberByTimestamp(_ context.Context, time uint64) (*big.Int, error) {
	return m.BlockNumberByHeight[time], nil
}

// DepositRoot --
func (m *POWChain) DepositRoot() [32]byte {
	root := []byte("depositroot")
	return bytesutil.ToBytes32(root)
}

// ChainStartDeposits --
func (m *POWChain) ChainStartDeposits() []*ethpb.Deposit {
	return []*ethpb.Deposit{}
}

// ChainStartDepositHashes --
func (m *POWChain) ChainStartDepositHashes() ([][]byte, error) {
	return [][]byte{}, nil
}

// ChainStartEth1Data --
func (m *POWChain) ChainStartEth1Data() *ethpb.Eth1Data {
	return m.Eth1Data
}
