package depositsnapshot

import (
	"context"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/wealdtech/go-bytesutil"
	"go.opencensus.io/trace"
	"math/big"
	"sync"
)

type Cache struct {
	pendingDeposits []*ethpb.DepositContainer
	deposits        []*ethpb.DepositContainer
	depositsByKey   map[[fieldparams.BLSPubkeyLength]byte][]*ethpb.DepositContainer
	depositsLock    sync.RWMutex
}

// FinalizedDeposits stores the trie of deposits that have been included
// in the beacon state up to the latest finalized checkpoint.
type FinalizedDeposits struct {
	Deposits        *depositTree
	MerkleTrieIndex int64
}

func (c *Cache) AllDeposits(ctx context.Context, untilBlk *big.Int) []*ethpb.Deposit {
	ctx, span := trace.StartSpan(ctx, "Cache.AllDeposits")
	defer span.End()
	c.depositsLock.RLock()
	defer c.depositsLock.RUnlock()

	return c.allDeposits(untilBlk)
}

func (c *Cache) allDeposits(untilBlk *big.Int) []*ethpb.Deposit {
	var deposits []*ethpb.Deposit
	for _, ctnr := range c.deposits {
		if untilBlk == nil || untilBlk.Uint64() >= ctnr.Eth1BlockHeight {
			deposits = append(deposits, ctnr.Deposit)
		}
	}
	return deposits
}

func (c *Cache) DepositByPubkey(ctx context.Context, pubKey []byte) (*ethpb.Deposit, *big.Int) {

	ctx, span := trace.StartSpan(ctx, "DepositsCache.DepositByPubkey")
	defer span.End()
	c.depositsLock.RLock()
	defer c.depositsLock.RUnlock()

	var deposit *ethpb.Deposit
	var blockNum *big.Int
	deps, ok := c.depositsByKey[bytesutil.ToBytes48(pubKey)]
	if !ok || len(deps) == 0 {
		return deposit, blockNum
	}
	// We always return the first deposit if a particular
	// validator key has multiple deposits assigned to
	// it.
	deposit = deps[0].Deposit
	blockNum = big.NewInt(int64(deps[0].Eth1BlockHeight))
	return deposit, blockNum
}

func (c *Cache) DepositsNumberAndRootAtHeight(ctx context.Context, blockHeight *big.Int) (uint64, [32]byte) {
	return 0, [32]byte{}
}

func (c *Cache) FinalizedDeposits(ctx context.Context) *FinalizedDeposits {
	return nil
}

func (c *Cache) NonFinalizedDeposits(ctx context.Context, lastFinalizedIndex int64, untilBlk *big.Int) []*ethpb.Deposit {
	return nil
}
