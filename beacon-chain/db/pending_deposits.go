package db

import (
	"context"
	"math/big"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

var (
	pendingDepositsCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "beacondb_pending_deposits",
		Help: "The number of pending deposits in the beaconDB in-memory database",
	})
)

// Container object for holding the deposit and a reference to the block in
// which the deposit transaction was included in the proof of work chain.
type depositContainer struct {
	deposit *pb.Deposit
	block   *big.Int
}

// InsertPendingDeposit into the database. If deposit or block number are nil
// then this method does nothing.
func (db *BeaconDB) InsertPendingDeposit(ctx context.Context, d *pb.Deposit, blockNum *big.Int) {
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
	db.pendingDeposits = append(db.pendingDeposits, &depositContainer{deposit: d, block: blockNum})
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

	var deposits []*pb.Deposit
	for _, ctnr := range db.pendingDeposits {
		if beforeBlk == nil || beforeBlk.Cmp(ctnr.block) > -1 {
			deposits = append(deposits, ctnr.deposit)
		}
	}
	// Sort the deposits by Merkle index.
	sort.SliceStable(deposits, func(i, j int) bool {
		return deposits[i].MerkleTreeIndex < deposits[j].MerkleTreeIndex
	})
	return deposits
}

// RemovePendingDeposit from the database. The deposit is indexed by the
// MerkleTreeIndex. This method does nothing if deposit ptr is nil.
func (db *BeaconDB) RemovePendingDeposit(ctx context.Context, d *pb.Deposit) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.RemovePendingDeposit")
	defer span.End()

	if d == nil {
		log.Debug("Ignoring nil deposit removal")
		return
	}

	db.depositsLock.Lock()
	defer db.depositsLock.Unlock()

	idx := -1
	for i, ctnr := range db.pendingDeposits {
		if ctnr.deposit.MerkleTreeIndex == d.MerkleTreeIndex {
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
func (db *BeaconDB) PrunePendingDeposits(ctx context.Context, merkleTreeIndex uint64) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.PrunePendingDeposits")
	defer span.End()

	if merkleTreeIndex == 0 {
		log.Debug("Ignoring 0 deposit removal")
		return
	}

	db.depositsLock.Lock()
	defer db.depositsLock.Unlock()

	var cleanDeposits []*depositContainer
	for _, dp := range db.pendingDeposits {
		if dp.deposit.MerkleTreeIndex >= merkleTreeIndex {
			cleanDeposits = append(cleanDeposits, dp)
		}
	}

	db.pendingDeposits = cleanDeposits
	pendingDepositsCount.Set(float64(len(db.pendingDeposits)))
}
