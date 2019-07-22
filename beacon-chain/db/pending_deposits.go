package db

import (
	"context"
	"math/big"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/prysmaticlabs/prysm/shared/hashutil"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	pendingDepositsCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacondb_pending_deposits",
		Help: "The number of pending deposits in the beaconDB in-memory database",
	})
)

// DepositContainer object for holding the deposit and a reference to the block in
// which the deposit transaction was included in the proof of work chain.
type DepositContainer struct {
	Deposit     *pb.Deposit
	Block       *big.Int
	Index       int
	depositRoot [32]byte
}

// InsertPendingDeposit into the database. If deposit or block number are nil
// then this method does nothing.
func (db *BeaconDB) InsertPendingDeposit(ctx context.Context, d *pb.Deposit, blockNum *big.Int, index int, depositRoot [32]byte) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.InsertPendingDeposit")
	defer span.End()
	if d == nil || blockNum == nil {
		log.WithFields(logrus.Fields{
			"block":   blockNum,
			"deposit": d,
		}).Debug("Ignoring nil deposit insertion")
		return
	}
	db.depositsLock.Lock()
	defer db.depositsLock.Unlock()
	db.pendingDeposits = append(db.pendingDeposits, &DepositContainer{Deposit: d, Block: blockNum, Index: index, depositRoot: depositRoot})
	pendingDepositsCount.Inc()
}

// PendingDeposits returns a list of deposits until the given block number
// (inclusive). If no block is specified then this method returns all pending
// deposits.
func (db *BeaconDB) PendingDeposits(ctx context.Context, beforeBlk *big.Int) []*pb.Deposit {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.PendingDeposits")
	defer span.End()
	db.depositsLock.RLock()
	defer db.depositsLock.RUnlock()

	var depositCntrs []*DepositContainer
	for _, ctnr := range db.pendingDeposits {
		if beforeBlk == nil || beforeBlk.Cmp(ctnr.Block) > -1 {
			depositCntrs = append(depositCntrs, ctnr)
		}
	}
	// Sort the deposits by Merkle index.
	sort.SliceStable(depositCntrs, func(i, j int) bool {
		return depositCntrs[i].Index < depositCntrs[j].Index
	})

	var deposits []*pb.Deposit
	for _, dep := range depositCntrs {
		deposits = append(deposits, dep.Deposit)
	}

	return deposits
}

// PendingContainers returns a list of deposit containers until the given block number
// (inclusive).
func (db *BeaconDB) PendingContainers(ctx context.Context, beforeBlk *big.Int) []*DepositContainer {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.PendingDeposits")
	defer span.End()
	db.depositsLock.RLock()
	defer db.depositsLock.RUnlock()

	var depositCntrs []*DepositContainer
	for _, ctnr := range db.pendingDeposits {
		if beforeBlk == nil || beforeBlk.Cmp(ctnr.Block) > -1 {
			depositCntrs = append(depositCntrs, ctnr)
		}
	}
	// Sort the deposits by Merkle index.
	sort.SliceStable(depositCntrs, func(i, j int) bool {
		return depositCntrs[i].Index < depositCntrs[j].Index
	})

	return depositCntrs
}

// RemovePendingDeposit from the database. The deposit is indexed by the
// Index. This method does nothing if deposit ptr is nil.
func (db *BeaconDB) RemovePendingDeposit(ctx context.Context, d *pb.Deposit) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.RemovePendingDeposit")
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

	db.depositsLock.Lock()
	defer db.depositsLock.Unlock()

	idx := -1
	for i, ctnr := range db.pendingDeposits {
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
		db.pendingDeposits = append(db.pendingDeposits[:idx], db.pendingDeposits[idx+1:]...)
		pendingDepositsCount.Dec()
	}
}

// PrunePendingDeposits removes any deposit which is older than the given deposit merkle tree index.
func (db *BeaconDB) PrunePendingDeposits(ctx context.Context, merkleTreeIndex int) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.PrunePendingDeposits")
	defer span.End()

	if merkleTreeIndex == 0 {
		log.Debug("Ignoring 0 deposit removal")
		return
	}

	db.depositsLock.Lock()
	defer db.depositsLock.Unlock()

	var cleanDeposits []*DepositContainer
	for _, dp := range db.pendingDeposits {
		if dp.Index >= merkleTreeIndex {
			cleanDeposits = append(cleanDeposits, dp)
		}
	}

	db.pendingDeposits = cleanDeposits
	pendingDepositsCount.Set(float64(len(db.pendingDeposits)))
}
