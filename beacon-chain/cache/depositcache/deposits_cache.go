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
	"github.com/prysmaticlabs/go-ssz"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/bytesutil"
	"github.com/prysmaticlabs/prysm/shared/params"
	"github.com/prysmaticlabs/prysm/shared/trieutil"
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
	AllDeposits(ctx context.Context, untilBlk *big.Int) []*ethpb.Deposit
	DepositByPubkey(ctx context.Context, pubKey []byte) (*ethpb.Deposit, *big.Int)
	DepositsNumberAndRootAtHeight(ctx context.Context, blockHeight *big.Int) (uint64, [32]byte)
	FinalizedDeposits(ctx context.Context) *FinalizedDeposits
	NonFinalizedDeposits(ctx context.Context, untilBlk *big.Int) []*ethpb.Deposit
}

// FinalizedDeposits stores the trie of deposits that have been included
// in the beacon state up to the latest finalized checkpoint.
type FinalizedDeposits struct {
	Deposits        *trieutil.SparseMerkleTrie
	MerkleTrieIndex int64
}

// DepositCache stores all in-memory deposit objects. This
// stores all the deposit related data that is required by the beacon-node.
type DepositCache struct {
	// Beacon chain deposits in memory.
	pendingDeposits    []*dbpb.DepositContainer
	deposits           []*dbpb.DepositContainer
	finalizedDeposits  *FinalizedDeposits
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

// InsertFinalizedDeposits inserts deposits up to eth1DepositIndex (exclusive) into the finalized deposits cache.
func (dc *DepositCache) InsertFinalizedDeposits(ctx context.Context, eth1DepositIndex int64) {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.InsertFinalizedDeposits")
	defer span.End()
	dc.depositsLock.Lock()
	defer dc.depositsLock.Unlock()

	var depositTrie *trieutil.SparseMerkleTrie

	if dc.finalizedDeposits != nil {
		depositTrie = dc.finalizedDeposits.Deposits

		insertIndex := dc.finalizedDeposits.MerkleTrieIndex + 1
		for _, d := range dc.deposits {
			if d.Index <= dc.finalizedDeposits.MerkleTrieIndex {
				continue
			}
			if d.Index > eth1DepositIndex {
				break
			}
			depHash, err := ssz.HashTreeRoot(d.Deposit.Data)
			if err != nil {
				log.WithError(err).Error("Could not hash deposit data. Finalized deposit cache not updated.")
				return
			}
			depositTrie.Insert(depHash[:], int(insertIndex))
			insertIndex++
		}
	} else {
		var finalizedDeposits [][]byte
		for _, d := range dc.deposits {
			if d.Index > eth1DepositIndex {
				break
			}
			hash, err := ssz.HashTreeRoot(d.Deposit.Data)
			if err != nil {
				log.WithError(err).Error("Could not hash deposit data. Finalized deposit cache not updated.")
				return
			}
			finalizedDeposits = append(finalizedDeposits, hash[:])
		}

		var err error
		depositTrie, err = trieutil.GenerateTrieFromItems(finalizedDeposits, int(params.BeaconConfig().DepositContractTreeDepth))
		if err != nil {
			log.WithError(err).Error("Could not generate deposit trie. Finalized deposit cache not updated.")
			return
		}
	}

	dc.finalizedDeposits = &FinalizedDeposits{
		Deposits:        depositTrie,
		MerkleTrieIndex: eth1DepositIndex,
	}
}

// AllDepositContainers returns all historical deposit containers.
func (dc *DepositCache) AllDepositContainers(ctx context.Context) []*dbpb.DepositContainer {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.AllDepositContainers")
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

// AllDeposits returns a list of historical deposits until the given block number
// (inclusive). If no block is specified then this method returns all historical deposits.
func (dc *DepositCache) AllDeposits(ctx context.Context, untilBlk *big.Int) []*ethpb.Deposit {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.AllDeposits")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	var deposits []*ethpb.Deposit
	for _, ctnr := range dc.deposits {
		if untilBlk == nil || untilBlk.Uint64() >= ctnr.Eth1BlockHeight {
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

// FinalizedDeposits returns the list of finalized deposits.
func (dc *DepositCache) FinalizedDeposits(ctx context.Context) *FinalizedDeposits {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.FinalizedDeposits")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	return dc.finalizedDeposits
}

// NonFinalizedDeposits returns the list of non-finalized deposits until the given block number (inclusive).
// If no block is specified then this method returns all non-finalized deposits.
func (dc *DepositCache) NonFinalizedDeposits(ctx context.Context, untilBlk *big.Int) []*ethpb.Deposit {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.NonFinalizedDeposits")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	if dc.finalizedDeposits == nil {
		return dc.AllDeposits(ctx, untilBlk)
	}

	lastFinalizedDepositIndex := dc.finalizedDeposits.MerkleTrieIndex
	var deposits []*ethpb.Deposit
	for _, d := range dc.deposits {
		if (d.Index > lastFinalizedDepositIndex) && (untilBlk == nil || untilBlk.Uint64() >= d.Eth1BlockHeight) {
			deposits = append(deposits, d.Deposit)
		}
	}

	return deposits
}
