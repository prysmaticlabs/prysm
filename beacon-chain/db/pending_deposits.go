package db

import (
	"context"
	"math/big"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"

	pb "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
)

var (
	depositsCount = promauto.NewGauge(prometheus.GaugeOpts{
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
	db.deposits = append(db.deposits, &depositContainer{deposit: d, block: blockNum})
	depositsCount.Inc()
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
	for _, ctnr := range db.deposits {
		if beforeBlk == nil || beforeBlk.Cmp(ctnr.block) > -1 {
			deposits = append(deposits, ctnr.deposit)
		}
	}

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
	for i, ctnr := range db.deposits {
		if ctnr.deposit.MerkleTreeIndex == d.MerkleTreeIndex {
			idx = i
			break
		}
	}

	if idx >= 0 {
		db.deposits = append(db.deposits[:idx], db.deposits[idx+1:]...)
		depositsCount.Dec()
	}
}
