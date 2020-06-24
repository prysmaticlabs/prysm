package depositcache

import (
	"context"
	"math/big"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	dbpb "github.com/prysmaticlabs/prysm/proto/beacon/db"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	pendingDepositsCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacondb_pending_deposits",
		Help: "The number of pending deposits in the beaconDB in-memory database",
	})
)

// PendingDepositsFetcher specifically outlines a struct that can retrieve deposits
// which have not yet been included in the chain.
type PendingDepositsFetcher interface {
	PendingContainers(ctx context.Context, untilBlk *big.Int) []*dbpb.DepositContainer
}

// InsertPendingDeposit into the database. If deposit or block number are nil
// then this method does nothing.
func (dc *DepositCache) InsertPendingDeposit(ctx context.Context, d *ethpb.Deposit, blockNum uint64, index int64, depositRoot [32]byte) {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.InsertPendingDeposit")
	defer span.End()
	if d == nil {
		log.WithFields(log.Fields{
			"block":   blockNum,
			"deposit": d,
		}).Debug("Ignoring nil deposit insertion")
		return
	}
	dc.depositsLock.Lock()
	defer dc.depositsLock.Unlock()
	dc.pendingDeposits = append(dc.pendingDeposits,
		&dbpb.DepositContainer{Deposit: d, Eth1BlockHeight: blockNum, Index: index, DepositRoot: depositRoot[:]})
	pendingDepositsCount.Inc()
	span.AddAttributes(trace.Int64Attribute("count", int64(len(dc.pendingDeposits))))
}

// PendingDeposits returns a list of deposits until the given block number
// (inclusive). If no block is specified then this method returns all pending
// deposits.
func (dc *DepositCache) PendingDeposits(ctx context.Context, untilBlk *big.Int) []*ethpb.Deposit {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.PendingDeposits")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	var depositCntrs []*dbpb.DepositContainer
	for _, ctnr := range dc.pendingDeposits {
		if untilBlk == nil || untilBlk.Uint64() >= ctnr.Eth1BlockHeight {
			depositCntrs = append(depositCntrs, ctnr)
		}
	}
	// Sort the deposits by Merkle index.
	sort.SliceStable(depositCntrs, func(i, j int) bool {
		return depositCntrs[i].Index < depositCntrs[j].Index
	})

	var deposits []*ethpb.Deposit
	for _, dep := range depositCntrs {
		deposits = append(deposits, dep.Deposit)
	}

	span.AddAttributes(trace.Int64Attribute("count", int64(len(deposits))))

	return deposits
}

// PendingContainers returns a list of deposit containers until the given block number
// (inclusive).
func (dc *DepositCache) PendingContainers(ctx context.Context, untilBlk *big.Int) []*dbpb.DepositContainer {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.PendingDeposits")
	defer span.End()
	dc.depositsLock.RLock()
	defer dc.depositsLock.RUnlock()

	var depositCntrs []*dbpb.DepositContainer
	for _, ctnr := range dc.pendingDeposits {
		if untilBlk == nil || untilBlk.Uint64() >= ctnr.Eth1BlockHeight {
			depositCntrs = append(depositCntrs, ctnr)
		}
	}
	// Sort the deposits by Merkle index.
	sort.SliceStable(depositCntrs, func(i, j int) bool {
		return depositCntrs[i].Index < depositCntrs[j].Index
	})

	span.AddAttributes(trace.Int64Attribute("count", int64(len(depositCntrs))))

	return depositCntrs
}

// RemovePendingDeposit from the database. The deposit is indexed by the
// Index. This method does nothing if deposit ptr is nil.
func (dc *DepositCache) RemovePendingDeposit(ctx context.Context, d *ethpb.Deposit) {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.RemovePendingDeposit")
	defer span.End()

	if d == nil {
		log.Debug("Ignoring nil deposit removal")
		return
	}

	depRoot, err := hashutil.HashProto(d)
	if err != nil {
		log.Errorf("Could not remove deposit %v", err)
		return
	}

	dc.depositsLock.Lock()
	defer dc.depositsLock.Unlock()

	idx := -1
	for i, ctnr := range dc.pendingDeposits {
		hash, err := hashutil.HashProto(ctnr.Deposit)
		if err != nil {
			log.Errorf("Could not hash deposit %v", err)
			continue
		}
		if hash == depRoot {
			idx = i
			break
		}
	}

	if idx >= 0 {
		dc.pendingDeposits = append(dc.pendingDeposits[:idx], dc.pendingDeposits[idx+1:]...)
		pendingDepositsCount.Dec()
	}
}

// PrunePendingDeposits removes any deposit which is older than the given deposit merkle tree index.
func (dc *DepositCache) PrunePendingDeposits(ctx context.Context, merkleTreeIndex int64) {
	ctx, span := trace.StartSpan(ctx, "DepositsCache.PrunePendingDeposits")
	defer span.End()

	if merkleTreeIndex == 0 {
		log.Debug("Ignoring 0 deposit removal")
		return
	}

	dc.depositsLock.Lock()
	defer dc.depositsLock.Unlock()

	var cleanDeposits []*dbpb.DepositContainer
	for _, dp := range dc.pendingDeposits {
		if dp.Index >= merkleTreeIndex {
			cleanDeposits = append(cleanDeposits, dp)
		}
	}

	dc.pendingDeposits = cleanDeposits
	pendingDepositsCount.Set(float64(len(dc.pendingDeposits)))
}
