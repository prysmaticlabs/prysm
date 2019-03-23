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

// MarkPubkeyForChainstart sets the pubkey deposit status to true.
func (db *BeaconDB) MarkPubkeyForChainstart(ctx context.Context, pubkey string) {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.MarkPubkeyForChainstart")
	defer span.End()
	db.chainstartPubkeysLock.Lock()
	defer db.chainstartPubkeysLock.Unlock()
	db.chainstartPubkeys[pubkey] = true
}

// PubkeyInChainstart returns bool for whether the pubkey passed in has deposited.
func (db *BeaconDB) PubkeyInChainstart(ctx context.Context, pubkey string) bool {
	ctx, span := trace.StartSpan(ctx, "BeaconDB.PubkeyInChainstart")
	defer span.End()
	db.chainstartPubkeysLock.Lock()
	defer db.chainstartPubkeysLock.Unlock()
	if db.chainstartPubkeys != nil {
		return db.chainstartPubkeys[pubkey]
	}
	db.chainstartPubkeys = make(map[string]bool)
	return false
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
