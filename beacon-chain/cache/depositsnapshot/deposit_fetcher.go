package depositsnapshot

import (
	"context"
	"math/big"
	"sort"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prysmaticlabs/prysm/v4/beacon-chain/cache"
	fieldparams "github.com/prysmaticlabs/prysm/v4/config/fieldparams"
	ethpb "github.com/prysmaticlabs/prysm/v4/proto/prysm/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/wealdtech/go-bytesutil"
	"go.opencensus.io/trace"
)

var (
	pendingDepositsCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacondb_pending_deposits_eip4881",
		Help: "The number of pending deposits in the beaconDB in-memory database",
	})
)

type Cache struct {
	pendingDeposits   []*ethpb.DepositContainer
	deposits          []*ethpb.DepositContainer
	finalizedDeposits finalizedDepositsContainer
	depositsByKey     map[[fieldparams.BLSPubkeyLength]byte][]*ethpb.DepositContainer
	depositsLock      sync.RWMutex
}

func (c *Cache) PendingDeposits(ctx context.Context, untilBlk *big.Int) []*ethpb.Deposit {
	//TODO implement me
	panic("implement me")
}

func (c *Cache) PendingContainers(ctx context.Context, untilBlk *big.Int) []*ethpb.DepositContainer {
	//TODO implement me
	panic("implement me")
}

// finalizedDepositsContainer stores the trie of deposits that have been included
// in the beacon state up to the latest finalized checkpoint.
type finalizedDepositsContainer struct {
	depositTree     *DepositTree
	merkleTrieIndex int64
}

// New instantiates a new deposit cache
func New() (*Cache, error) {
	finalizedDepositsTrie := NewDepositTree()

	// finalizedDeposits.merkleTrieIndex is initialized to -1 because it represents the index of the last trie item.
	// Inserting the first item into the trie will set the value of the index to 0.
	return &Cache{
		pendingDeposits:   []*ethpb.DepositContainer{},
		deposits:          []*ethpb.DepositContainer{},
		depositsByKey:     map[[fieldparams.BLSPubkeyLength]byte][]*ethpb.DepositContainer{},
		finalizedDeposits: getFinalizedDeposits(finalizedDepositsTrie, -1),
	}, nil
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

// AllDepositContainers returns all historical deposit containers.
func (c *Cache) AllDepositContainers(ctx context.Context) []*ethpb.DepositContainer {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.AllDepositContainers")
	defer span.End()
	c.depositsLock.RLock()
	defer c.depositsLock.RUnlock()

	// Make a shallow copy of the deposits and return that. This way, the
	// caller can safely iterate over the returned list of deposits without
	// the possibility of new deposits showing up. If we were to return the
	// list without a copy, when a new deposit is added to the cache, it
	// would also be present in the returned value. This could result in a
	// race condition if the list is being iterated over.
	//
	// It's not necessary to make a deep copy of this list because the
	// deposits in the cache should never be modified. It is still possible
	// for the caller to modify one of the underlying deposits and modify
	// the cache, but that's not a race condition. Also, a deep copy would
	// take too long and use too much memory.
	deposits := make([]*ethpb.DepositContainer, len(c.deposits))
	copy(deposits, c.deposits)
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
	ctx, span := trace.StartSpan(ctx, "DepositsCache.DepositsNumberAndRootAtHeight")
	defer span.End()
	c.depositsLock.RLock()
	defer c.depositsLock.RUnlock()
	heightIdx := sort.Search(len(c.deposits), func(i int) bool { return c.deposits[i].Eth1BlockHeight > blockHeight.Uint64() })
	// send the deposit root of the empty trie, if eth1follow distance is greater than the time of the earliest
	// deposit.
	if heightIdx == 0 {
		return 0, [32]byte{}
	}
	return uint64(heightIdx), bytesutil.ToBytes32(c.deposits[heightIdx-1].DepositRoot)
}

func (c *Cache) FinalizedDeposits(ctx context.Context) cache.FinalizedDeposits {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.finalizedDepositsContainer")
	defer span.End()
	c.depositsLock.RLock()
	defer c.depositsLock.RUnlock()

	return &finalizedDepositsContainer{
		depositTree:     c.finalizedDeposits.depositTree,
		merkleTrieIndex: c.finalizedDeposits.merkleTrieIndex,
	}
}

func (c *Cache) NonFinalizedDeposits(ctx context.Context, lastFinalizedIndex int64, untilBlk *big.Int) []*ethpb.Deposit {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.NonFinalizedDeposits")
	defer span.End()
	c.depositsLock.RLock()
	defer c.depositsLock.RUnlock()

	if c.finalizedDeposits.depositTree == nil {
		return c.allDeposits(untilBlk)
	}

	var deposits []*ethpb.Deposit
	for _, d := range c.deposits {
		if (d.Index > lastFinalizedIndex) && (untilBlk == nil || untilBlk.Uint64() >= d.Eth1BlockHeight) {
			deposits = append(deposits, d.Deposit)
		}
	}

	return deposits
}

// PruneProofs removes proofs from all deposits whose index is equal or less than untilDepositIndex.
func (c *Cache) PruneProofs(ctx context.Context, untilDepositIndex int64) error {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.PruneProofs")
	defer span.End()
	c.depositsLock.Lock()
	defer c.depositsLock.Unlock()

	if untilDepositIndex >= int64(len(c.deposits)) {
		untilDepositIndex = int64(len(c.deposits) - 1)
	}

	for i := untilDepositIndex; i >= 0; i-- {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Finding a nil proof means that all proofs up to this deposit have been already pruned.
		if c.deposits[i].Deposit.Proof == nil {
			break
		}
		c.deposits[i].Deposit.Proof = nil
	}

	return nil
}

// PrunePendingDeposits removes any deposit which is older than the given deposit merkle tree index.
func (c *Cache) PrunePendingDeposits(ctx context.Context, merkleTreeIndex int64) {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.PrunePendingDeposits")
	defer span.End()

	if merkleTreeIndex == 0 {
		log.Debug("Ignoring 0 deposit removal")
		return
	}

	c.depositsLock.Lock()
	defer c.depositsLock.Unlock()

	cleanDeposits := make([]*ethpb.DepositContainer, 0, len(c.pendingDeposits))
	for _, dp := range c.pendingDeposits {
		if dp.Index >= merkleTreeIndex {
			cleanDeposits = append(cleanDeposits, dp)
		}
	}

	c.pendingDeposits = cleanDeposits
	pendingDepositsCount.Set(float64(len(c.pendingDeposits)))
}

// InsertPendingDeposit into the database. If deposit or block number are nil
// then this method does nothing.
func (c *Cache) InsertPendingDeposit(ctx context.Context, d *ethpb.Deposit, blockNum uint64, index int64, depositRoot [32]byte) {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.InsertPendingDeposit")
	defer span.End()
	if d == nil {
		log.WithFields(logrus.Fields{
			"block":   blockNum,
			"deposit": d,
		}).Debug("Ignoring nil deposit insertion")
		return
	}
	c.depositsLock.Lock()
	defer c.depositsLock.Unlock()
	c.pendingDeposits = append(c.pendingDeposits,
		&ethpb.DepositContainer{Deposit: d, Eth1BlockHeight: blockNum, Index: index, DepositRoot: depositRoot[:]})
	pendingDepositsCount.Inc()
	span.AddAttributes(trace.Int64Attribute("count", int64(len(c.pendingDeposits))))
}

func (fd *finalizedDepositsContainer) Deposits() cache.MerkleTree {
	return fd.depositTree
}

func (fd *finalizedDepositsContainer) MerkleTrieIndex() int64 {
	return fd.merkleTrieIndex
}

func getFinalizedDeposits(deposits *DepositTree, index int64) finalizedDepositsContainer {
	return finalizedDepositsContainer{
		depositTree:     deposits,
		merkleTrieIndex: index,
	}
}
