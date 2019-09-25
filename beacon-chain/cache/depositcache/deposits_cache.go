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
	ethpb "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
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
	pendingDeposits       []*DepositContainer
	deposits              []*DepositContainer
	depositsLock          sync.RWMutex
	chainStartDeposits    []*ethpb.Deposit
	chainstartPubkeys     map[string]bool
	chainstartPubkeysLock sync.RWMutex
}

// DepositContainer object for holding the deposit and a reference to the block in
// which the deposit transaction was included in the proof of work chain.
type DepositContainer struct {
	Deposit     *ethpb.Deposit
	Block       *big.Int
	Index       int
	depositRoot [32]byte
}

// NewDepositCache instantiates a new deposit cache
func NewDepositCache() *DepositCache {
	return &DepositCache{
		pendingDeposits:    []*DepositContainer{},
		deposits:           []*DepositContainer{},
		chainstartPubkeys:  make(map[string]bool),
		chainStartDeposits: make([]*ethpb.Deposit, 0),
	}
}

// InsertDeposit into the database. If deposit or block number are nil
// then this method does nothing.
func (dc *DepositCache) InsertDeposit(ctx context.Context, d *ethpb.Deposit, blockNum *big.Int, index int, depositRoot [32]byte) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.InsertDeposit")
	defer span.End()
	if d == nil || blockNum == nil {
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
	// keep the slice sorted on insertion in order to avoid costly sorting on retrival.
	heightIdx := sort.Search(len(dc.deposits), func(i int) bool { return dc.deposits[i].Index >= index })
	newDeposits := append([]*DepositContainer{{Deposit: d, Block: blockNum, depositRoot: depositRoot, Index: index}}, dc.deposits[heightIdx:]...)
	dc.deposits = append(dc.deposits[:heightIdx], newDeposits...)
	historicalDepositsCount.Inc()
}

// MarkPubkeyForChainstart sets the pubkey deposit status to true.
func (dc *DepositCache) MarkPubkeyForChainstart(ctx context.Context, pubkey string) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.MarkPubkeyForChainstart")
	defer span.End()
	dc.chainstartPubkeysLock.Lock()
	defer dc.chainstartPubkeysLock.Unlock()
	dc.chainstartPubkeys[pubkey] = true
}

// PubkeyInChainstart returns bool for whether the pubkey passed in has deposited.
func (dc *DepositCache) PubkeyInChainstart(ctx context.Context, pubkey string) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.PubkeyInChainstart")
	defer span.End()
	dc.chainstartPubkeysLock.Lock()
	defer dc.chainstartPubkeysLock.Unlock()
	if dc.chainstartPubkeys != nil {
		return dc.chainstartPubkeys[pubkey]
	}
	dc.chainstartPubkeys = make(map[string]bool)
	return false
}

// AllDeposits returns a list of deposits all historical deposits until the given block number
// (inclusive). If no block is specified then this method returns all historical deposits.
func (dc *DepositCache) AllDeposits(ctx context.Context, beforeBlk *big.Int) []*ethpb.Deposit {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.AllDeposits")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	var deposits []*ethpb.Deposit
	for _, ctnr := range dc.deposits {
		if beforeBlk == nil || beforeBlk.Cmp(ctnr.Block) > -1 {
			deposits = append(deposits, ctnr.Deposit)
		}
	}
	return deposits
}

// DepositsNumberAndRootAtHeight returns number of deposits made prior to blockheight and the
// root that corresponds to the latest deposit at that blockheight.
func (dc *DepositCache) DepositsNumberAndRootAtHeight(ctx context.Context, blockHeight *big.Int) (uint64, [32]byte) {
	ctx, span := trace.StartSpan(ctx, "Beacondb.DepositsNumberAndRootAtHeight")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()
	heightIdx := sort.Search(len(dc.deposits), func(i int) bool { return dc.deposits[i].Block.Cmp(blockHeight) > 0 })
	// send the deposit root of the empty trie, if eth1follow distance is greater than the time of the earliest
	// deposit.
	if heightIdx == 0 {
		return 0, [32]byte{}
	}
	return uint64(heightIdx), dc.deposits[heightIdx-1].depositRoot
}

// DepositByPubkey looks through historical deposits and finds one which contains
// a certain public key within its deposit data.
func (dc *DepositCache) DepositByPubkey(ctx context.Context, pubKey []byte) (*ethpb.Deposit, *big.Int) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.DepositByPubkey")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	var deposit *ethpb.Deposit
	var blockNum *big.Int
	for _, ctnr := range dc.deposits {
		if bytes.Equal(ctnr.Deposit.Data.PublicKey, pubKey) {
			deposit = ctnr.Deposit
			blockNum = ctnr.Block
			break
		}
	}
	return deposit, blockNum
}
