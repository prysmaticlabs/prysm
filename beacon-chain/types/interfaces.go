package types

import (
	"context"
	"hash"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
)

// SyncService is the interface for the sync service.
type SyncService interface {
	ReceiveBlockHash(hash.Hash)
	ReceiveBlock(*Block) error
}

// NetworkService is the interface for the p2p network.
type NetworkService interface {
	BroadcastBlockHash(hash.Hash) error
	BroadcastBlock(*Block) error
	RequestBlock(hash.Hash) error
}

// ChainService is the interface for the local beacon chain.
type ChainService interface {
	ProcessBlock(*Block) error
	ContainsBlock(hash.Hash) bool
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
