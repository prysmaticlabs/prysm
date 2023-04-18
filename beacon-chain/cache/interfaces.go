package cache

import (
	"context"
	"math/big"

	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
)

type DepositCache interface {
	DepositFetcher
	DepositInserter
}

// DepositFetcher defines a struct which can retrieve deposit information from a store.
type DepositFetcher interface {
	AllDeposits(ctx context.Context, untilBlk *big.Int) []*ethpb.Deposit
	AllDepositContainers(ctx context.Context) []*ethpb.DepositContainer
	DepositByPubkey(ctx context.Context, pubKey []byte) (*ethpb.Deposit, *big.Int)
	DepositsNumberAndRootAtHeight(ctx context.Context, blockHeight *big.Int) (uint64, [32]byte)
	InsertPendingDeposit(ctx context.Context, d *ethpb.Deposit, blockNum uint64, index int64, depositRoot [32]byte)
	PendingDeposits(ctx context.Context, untilBlk *big.Int) []*ethpb.Deposit
	PendingContainers(ctx context.Context, untilBlk *big.Int) []*ethpb.DepositContainer
	PrunePendingDeposits(ctx context.Context, merkleTreeIndex int64)
	PruneProofs(ctx context.Context, untilDepositIndex int64) error
	FinalizedFetcher
}

// DepositInserter defines a struct which can insert deposit information from a store.
type DepositInserter interface {
	InsertDeposit(ctx context.Context, d *ethpb.Deposit, blockNum uint64, index int64, depositRoot [32]byte) error
	InsertDepositContainers(ctx context.Context, ctrs []*ethpb.DepositContainer)
	InsertFinalizedDeposits(ctx context.Context, eth1DepositIndex int64) error
}

type FinalizedFetcher interface {
	FinalizedDeposits(ctx context.Context) FinalizedDeposits
	NonFinalizedDeposits(ctx context.Context, lastFinalizedIndex int64, untilBlk *big.Int) []*ethpb.Deposit
}

type FinalizedDeposits interface {
	Deposits() MerkleTree
	MerkleTrieIndex() int64
}

type MerkleTree interface {
	HashTreeRoot() ([32]byte, error)
	NumOfItems() int
	Insert(item []byte, index int) error
	MerkleProof(index int) ([][]byte, error)
}
