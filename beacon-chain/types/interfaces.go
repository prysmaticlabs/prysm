package types

import (
	"context"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

// P2P defines a struct that can subscribe to feeds, request data, and broadcast data.
type P2P interface {
	Feed(msg interface{}) *event.Feed
	Send(msg interface{}, peer p2p.Peer)
	Broadcast(msg interface{})
}

// ChainService is the interface for the local beacon chain.
type ChainService interface {
	ProcessedHashes() [][32]byte
	ProcessBlock(b *Block) error
	ContainsBlock(h [32]byte) bool
}

// StateFetcher defines a struct that can fetch the latest canonical beacon state of a node.
type StateFetcher interface {
	CurrentActiveState() *ActiveState
	CurrentCrystallizedState() *CrystallizedState
}

// POWChainService is an interface for a proof-of-work chain web3 service.
type POWChainService interface {
	LatestBlockHash() common.Hash
}

// Reader defines a struct that can fetch latest header events from a web3 endpoint.
type Reader interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *gethTypes.Header) (ethereum.Subscription, error)
}

// POWBlockFetcher defines a struct that can retrieve mainchain blocks.
type POWBlockFetcher interface {
	BlockByHash(ctx context.Context, hash common.Hash) (*gethTypes.Block, error)
}

// Logger subscribe filtered log on the PoW chain
type Logger interface {
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- gethTypes.Log) (ethereum.Subscription, error)
}

// POWChainClient defines a struct that combines all relevant PoW mainchain interactions required
// by the beacon chain node.
type POWChainClient interface {
	Reader
	POWBlockFetcher
	Logger
}
