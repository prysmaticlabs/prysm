package types

import (
	"context"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/golang/protobuf/proto"
	"github.com/prysmaticlabs/prysm/shared/event"
	"github.com/prysmaticlabs/prysm/shared/p2p"
)

// P2P defines a struct that can subscribe to feeds, request data, and broadcast data.
type P2P interface {
	Subscribe(msg proto.Message, channel chan p2p.Message) event.Subscription
	Send(msg proto.Message, peer p2p.Peer)
	Broadcast(msg proto.Message)
}

// ChainService is the interface for the local beacon chain.
type ChainService interface {
	BlockChainService
}

// BlockChainService is the interface for block related functions in local beacon chain.
type BlockChainService interface {
	ProcessedBlockHashes() [][32]byte
	ProcessBlock(b *Block)
	HasStoredState() (bool, error)
	ContainsBlock(h [32]byte) (bool, error)
	SaveBlock(b *Block) error
	GetBlockSlotNumber(h [32]byte) (uint64, error)
}

// CrystallizedStateChainService is the interface for crystallized state related functions in local beacon chain.
type CrystallizedStateChainService interface {
	ProcessedCrystallizedStateHashes() [][32]byte
	ProcessCrystallizedState(c *CrystallizedState) error
	ContainsCrystallizedState(h [32]byte) bool
}

// ActiveStateChainService is the interface for active state related functions in local beacon chain.
type ActiveStateChainService interface {
	ProcessedActiveStateHashes() [][32]byte
	ProcessActiveState(a *ActiveState) error
	ContainsActiveState(h [32]byte) bool
}

// StateFetcher defines a struct that can fetch the latest canonical beacon state and genesis block of a node.
type StateFetcher interface {
	CurrentActiveState() *ActiveState
	CurrentCrystallizedState() *CrystallizedState
	GenesisBlock() (*Block, error)
	CurrentBeaconSlot() uint64
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

// Logger defines a struct that subscribes to filtered logs on the PoW chain.
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
