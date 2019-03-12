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
	historicalDepositsCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "beacondb_all_deposits",
		Help: "The number of total deposits in the beaconDB in-memory database",
	})
)

// InsertDeposit into the database. If deposit or block number are nil
// then this method does nothing.
func (db *BeaconDB) InsertDeposit(ctx context.Context, d *pb.Deposit, blockNum *big.Int) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.InsertDeposit")
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
	historicalDepositsCount.Inc()
}

// AllDeposits returns a list of deposits all historical deposits until the given block number
// (inclusive). If no block is specified then this method returns all historical deposits.
func (db *BeaconDB) AllDeposits(ctx context.Context, beforeBlk *big.Int) []*pb.Deposit {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.AllDeposits")
	defer span.End()
	db.depositsLock.RLock()
	defer db.depositsLock.RUnlock()

	var deposits []*pb.Deposit
	for _, ctnr := range db.deposits {
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
