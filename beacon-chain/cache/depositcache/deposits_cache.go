// Package depositcache is the source of validator deposits maintained
// in-memory by the beacon node – deposits processed from the
// eth1 powchain are then stored in this cache to be accessed by
// any other service during a beacon node's runtime.
package depositcache

import (
	"bytes"
	"context"
	"encoding/hex"
	"math/big"
	"sort"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	historicalDepositsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "beacondb_all_deposits",
		Help: "The number of total deposits in the beaconDB in-memory database",
	})
)

// DepositFetcher defines a struct which can retrieve deposit information from a store.
type DepositFetcher interface {
	AllDeposits(ctx context.Context, beforeBlk *big.Int) []*ethpb.Deposit
	DepositByPubkey(ctx context.Context, pubKey []byte) (*ethpb.Deposit, *big.Int)
	DepositsNumberAndRootAtHeight(ctx context.Context, blockHeight *big.Int) (uint64, [32]byte)
}

// DepositCache stores all in-memory deposit objects. This
// stores all the deposit related data that is required by the beacon-node.
type DepositCache struct {
	// Beacon chain deposits in memory.
	pendingDeposits    []*dbpb.DepositContainer
	deposits           []*dbpb.DepositContainer
	depositsLock       sync.RWMutex
	chainStartDeposits []*ethpb.Deposit
	chainStartPubkeys  map[string]bool
}

// NewDepositCache instantiates a new deposit cache
func NewDepositCache() *DepositCache {
	return &DepositCache{
		pendingDeposits:    []*dbpb.DepositContainer{},
		deposits:           []*dbpb.DepositContainer{},
		chainStartPubkeys:  make(map[string]bool),
		chainStartDeposits: make([]*ethpb.Deposit, 0),
	}
}

// InsertDeposit into the database. If deposit or block number are nil
// then this method does nothing.
func (dc *DepositCache) InsertDeposit(ctx context.Context, d *ethpb.Deposit, blockNum uint64, index int64, depositRoot [32]byte) {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.InsertDeposit")
	defer span.End()
	if d == nil {
		log.WithFields(log.Fields{
			"block":        blockNum,
			"deposit":      d,
			"index":        index,
			"deposit root": hex.EncodeToString(depositRoot[:]),
		}).Warn("Ignoring nil deposit insertion")
		return
	}
	dc.depositsLock.Lock()
	defer dc.depositsLock.Unlock()
	// Keep the slice sorted on insertion in order to avoid costly sorting on retrieval.
	heightIdx := sort.Search(len(dc.deposits), func(i int) bool { return dc.deposits[i].Index >= index })
	newDeposits := append([]*dbpb.DepositContainer{{Deposit: d, Eth1BlockHeight: blockNum, DepositRoot: depositRoot[:], Index: index}}, dc.deposits[heightIdx:]...)
	dc.deposits = append(dc.deposits[:heightIdx], newDeposits...)
	historicalDepositsCount.Inc()
}

// InsertDepositContainers inserts a set of deposit containers into our deposit cache.
func (dc *DepositCache) InsertDepositContainers(ctx context.Context, ctrs []*dbpb.DepositContainer) {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.InsertDepositContainers")
	defer span.End()
	dc.depositsLock.Lock()
	defer dc.depositsLock.Unlock()

	sort.SliceStable(ctrs, func(i int, j int) bool { return ctrs[i].Index < ctrs[j].Index })
	dc.deposits = ctrs
	historicalDepositsCount.Add(float64(len(ctrs)))
}

// AllDepositContainers returns a list of deposits all historical deposit containers until the given block number.
func (dc *DepositCache) AllDepositContainers(ctx context.Context) []*dbpb.DepositContainer {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.AllDepositContainers")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	return dc.deposits
}

// MarkPubkeyForChainstart sets the pubkey deposit status to true.
func (dc *DepositCache) MarkPubkeyForChainstart(ctx context.Context, pubkey string) {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.MarkPubkeyForChainstart")
	defer span.End()
	dc.chainStartPubkeys[pubkey] = true
}

// PubkeyInChainstart returns bool for whether the pubkey passed in has deposited.
func (dc *DepositCache) PubkeyInChainstart(ctx context.Context, pubkey string) bool {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.PubkeyInChainstart")
	defer span.End()
	if dc.chainStartPubkeys != nil {
		return dc.chainStartPubkeys[pubkey]
	}
	dc.chainStartPubkeys = make(map[string]bool)
	return false
}

// AllDeposits returns a list of deposits all historical deposits until the given block number
// (inclusive). If no block is specified then this method returns all historical deposits.
func (dc *DepositCache) AllDeposits(ctx context.Context, beforeBlk *big.Int) []*ethpb.Deposit {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.AllDeposits")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	var deposits []*ethpb.Deposit
	for _, ctnr := range dc.deposits {
		if beforeBlk == nil || beforeBlk.Uint64() >= ctnr.Eth1BlockHeight {
			deposits = append(deposits, ctnr.Deposit)
		}
	}
	return deposits
}

// DepositsNumberAndRootAtHeight returns number of deposits made prior to blockheight and the
// root that corresponds to the latest deposit at that blockheight.
func (dc *DepositCache) DepositsNumberAndRootAtHeight(ctx context.Context, blockHeight *big.Int) (uint64, [32]byte) {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.DepositsNumberAndRootAtHeight")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()
	heightIdx := sort.Search(len(dc.deposits), func(i int) bool { return dc.deposits[i].Eth1BlockHeight > blockHeight.Uint64() })
	// send the deposit root of the empty trie, if eth1follow distance is greater than the time of the earliest
	// deposit.
	if heightIdx == 0 {
		return 0, [32]byte{}
	}
	return uint64(heightIdx), bytesutil.ToBytes32(dc.deposits[heightIdx-1].DepositRoot)
}

// DepositByPubkey looks through historical deposits and finds one which contains
// a certain public key within its deposit data.
func (dc *DepositCache) DepositByPubkey(ctx context.Context, pubKey []byte) (*ethpb.Deposit, *big.Int) {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.DepositByPubkey")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	var deposit *ethpb.Deposit
	var blockNum *big.Int
	for _, ctnr := range dc.deposits {
		if bytes.Equal(ctnr.Deposit.Data.PublicKey, pubKey) {
			deposit = ctnr.Deposit
			blockNum = big.NewInt(int64(ctnr.Eth1BlockHeight))
			break
		}
	}
	return deposit, blockNum
}
